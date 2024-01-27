package models

import (
	"bufio"
	"bytes"
	"context"
	b64 "encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const HTTP200 = "HTTP/1.1 200 Connection Established\r\n\r\n"
const HTTP407 = "407 Proxy Authentication Required"

type Proxy struct {
	PrometheusAddress      string
	MetricsLogger          string
	ExitNodesFile          string
	AuthenticatedUsersFile string
	ListenAddress          string
	Username               string
	Password               string
	Whitelist              string
	Backends               []string
	Sessions               map[string]ExitNode
	ExitNodes              struct {
		All          []ExitNode
		ByRegion     map[string][]ExitNode
		ByInstanceID map[string]ExitNode
	}
	SessionMutex *sync.Mutex
	Dialer       proxy.Dialer
	Mutex        *sync.Mutex
	Timeout      int
	LogMetrics   bool
	IsUpstream   bool
	AuthUpstream bool
}

func (p *Proxy) GetExitNode(requestContext RequestContext) (ExitNode, string) {
	var exitNode ExitNode
	if requestContext.Instance != "" {
		exitNode, _ = p.ByInstanceID(requestContext.Instance)
	} else if requestContext.Region != "" {
		exitNode, _ = p.ByRegion(requestContext.Region)
	} else if requestContext.Session != "" {
		exitNode, _ = p.BySession(requestContext.UserID, requestContext.Session)
	}

	// get one at random (default) or if the others failed
	if exitNode.Interface == "" && exitNode.Upstream == "" {
		exitNode, _ = p.ByRandom()
	}
	backend := exitNode.Interface
	if p.IsUpstream == true {
		backend = exitNode.Upstream
	}
	log.Trace().
		Str("exitNode", backend).
		Str("userID", requestContext.UserID).
		Str("session", requestContext.Session).
		Str("project", requestContext.Project).
		Str("region", requestContext.Region).
		Bool("upstream", p.IsUpstream).
		Msg("ExitNode selected")
	return exitNode, backend
}

func (p *Proxy) setDialer(requestContext RequestContext, isClearText bool) (ExitNode, string, proxy.Dialer) {
	exitNode, backend := p.GetExitNode(requestContext)

	network := "tcp4"
	format := `%s:0`
	if p.IsUpstream == true && isClearText == false {
		format = `%s`
	}

	if len(strings.Split(backend, ":")) > 4 && len(backend) > 15 && p.IsUpstream == false {
		network = "tcp6"
		format = `%s`
	}

	addr, err := net.ResolveTCPAddr(network, fmt.Sprintf(format, backend))
	if err != nil {
		log.Trace().Err(err).Str("backend", backend).Msg("Resolve")
	}
	thisDialer := &net.Dialer{
		LocalAddr: addr,
		Timeout:   time.Duration(p.Timeout) * time.Second,
	}

	return exitNode, network, thisDialer
}
func (p *Proxy) isInWhitelist(requestAddress string) bool {
	if p.Whitelist == "" {
		return true
	}
	if addressParts := strings.Split(requestAddress, ":"); len(addressParts) > 0 {
		parsedRequestAddress := addressParts[0]
		for _, whitelistItem := range strings.Split(p.Whitelist, ",") {
			if strings.HasPrefix(parsedRequestAddress, whitelistItem) {
				return true
			}
		}
	}
	return false
}

func (p *Proxy) handleRequest(responseWriter http.ResponseWriter, request *http.Request) {
	defer func() {
		// Delete hop by hop headers
		for _, v := range []string{
			"Proxy-Connection",
			"Proxy-Authorization",
			"Proxy-Authenticate",
			"Te",
			"Trailers",
		} {
			request.Header.Del(v)
		}
	}()
	if p.isInWhitelist(request.RemoteAddr) == false {
		return
	}
	requestContext := RequestContext{}

	passedAuthentication := false
	if len(UserMap) == 0 {
		passedAuthentication = true
	}

	if passedAuthentication == false {
		requestContext.FromRequest(request)
		if requestContext.Authenticated == true {
			passedAuthentication = true
		}
	}

	if passedAuthentication == true {
		if request.Method == http.MethodConnect {
			p.handleTunnel(responseWriter, request, requestContext)
		} else {
			p.handleHTTP(responseWriter, request, requestContext)
		}
	} else {
		p.handleProxyAuthRequired(responseWriter, request)
	}
}

func (p *Proxy) handleHTTP(responseWriter http.ResponseWriter, request *http.Request, requestContext RequestContext) {
	var buf bytes.Buffer
	tee := io.TeeReader(request.Body, &buf)
	bodySize, _ := io.ReadAll(tee)
	urlSize := len(request.URL.String())

	headersSize := 0
	for k, v := range request.Header {
		headersSize += len(k) + len(v)
	}
	requestSize := len(bodySize) + urlSize + headersSize

	exitNode, _, thisDialer := p.setDialer(requestContext, true)
	transport := http.Transport{
		DialContext: thisDialer.(interface {
			DialContext(context context.Context, network, address string) (net.Conn, error)
		}).DialContext,
	}

	if p.IsUpstream {
		u, err := url.Parse("http://" + exitNode.Upstream)
		if err != nil {
			log.Err(err).Str("upstream", exitNode.Upstream).Msg("error parsing upstream")
			return
		}
		transport.Proxy = http.ProxyURL(u)
	}
	response, err := transport.RoundTrip(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	copyHeader(responseWriter.Header(), response.Header)
	responseWriter.WriteHeader(response.StatusCode)

	bytesTransferred, _ := io.Copy(responseWriter, response.Body)
	go func() {
		p.LogPayload(MetricPayload{
			Protocol:         "http",
			UserID:           requestContext.UserID,
			BytesTransferred: int64(requestSize),
			Direction:        "tx",
			Region:           requestContext.Region,
			Host:             request.Host,
		})

		p.LogPayload(MetricPayload{
			Protocol:         "http",
			UserID:           requestContext.UserID,
			BytesTransferred: bytesTransferred,
			Direction:        "rx",
			Region:           requestContext.Region,
			Host:             request.Host,
		})
	}()
}

func (p *Proxy) getUpstream(upstream string, addr string, requestContext RequestContext) (net.Conn, error) {
	network := "tcp"
	hdr := make(http.Header)
	upstream = strings.TrimPrefix(upstream, "https://")
	upstream = strings.TrimPrefix(upstream, "http://")
	if upstreamParts := strings.Split(upstream, "@"); len(upstreamParts) > 1 {
		upstream = upstreamParts[1]
		auth := b64.StdEncoding.EncodeToString([]byte(upstreamParts[0]))
		hdr.Add("Proxy-Authorization", fmt.Sprintf("Basic %s", auth))
	}
	if p.AuthUpstream == true {
		hdr.Add("Proxy-Authorization", fmt.Sprintf("Basic %s", requestContext.RawCreds))
	}

	connectReq := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: hdr,
	}

	c, err := net.DialTimeout(network, upstream, time.Duration(10)*time.Second)
	if err != nil {
		return nil, err
	}
	_ = connectReq.Write(c)
	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, connectReq)
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	defer resp.Body.Close()

	return c, nil
}

func (p *Proxy) handleTunnel(responseWriter http.ResponseWriter, request *http.Request, requestContext RequestContext) {
	var destinationConnection net.Conn
	var err error
	exitNode, network, thisDialer := p.setDialer(requestContext, false)

	if p.IsUpstream == true {
		destinationConnection, err = p.getUpstream(exitNode.Upstream, request.Host, requestContext)
	} else {
		destinationConnection, err = thisDialer.Dial(network, request.Host)
	}
	if err != nil {
		log.Trace().Err(err).Msg("HandleTunnel")
	}

	hijacker, ok := responseWriter.(http.Hijacker)
	if !ok {
		return
	}

	sourceConnection, _, err := hijacker.Hijack()
	if err != nil {
		return
	}

	if err != nil {
		_ = sourceConnection.Close()
		return
	}
	_, _ = sourceConnection.Write([]byte(HTTP200))

	go p.copyIO(sourceConnection, destinationConnection, "rx", requestContext, request.Host)
	go p.copyIO(destinationConnection, sourceConnection, "tx", requestContext, request.Host)
}

func (p *Proxy) handleProxyAuthRequired(responseWriter http.ResponseWriter, request *http.Request) {
	hijacker, ok := responseWriter.(http.Hijacker)
	if !ok {
		return
	}

	authRequiredResponse := &http.Response{
		StatusCode: 407,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Request:    request,
		Header: http.Header{
			"Proxy-Authenticate": []string{"Basic"},
			"Proxy-Connection":   []string{"close"},
		},
		Body:          io.NopCloser(bytes.NewBuffer([]byte(HTTP407))),
		ContentLength: int64(len(HTTP407)),
	}

	sourceConnection, _, err := hijacker.Hijack()
	_ = authRequiredResponse.Write(sourceConnection)
	_ = sourceConnection.Close()
	if err != nil {
		log.Err(err).Msg("Cannot hijack connection")
	}
}

func (p *Proxy) Run() {
	if p.MetricsLogger == "prometheus" {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			if err := http.ListenAndServe(p.PrometheusAddress, nil); err != nil {
				log.Fatal().Err(err).Msg("Prometheus handler")
			}
		}()
	}

	p.ExitNodesFromDisk()
	err := http.ListenAndServe(p.ListenAddress, http.HandlerFunc(p.handleRequest))
	if err != nil {
		log.Fatal().Err(err).Msg("ListenAndServe")
	}
}

func (p *Proxy) copyIO(src, dest net.Conn, direction string, requestContext RequestContext, host string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	defer func(src net.Conn) {
		if src == nil {
			err = errors.New("nil connection")
			return
		}

		err = src.Close()
		if err != nil {
			return
		}
	}(src)

	defer func(dest net.Conn) {
		if dest == nil {
			err = errors.New("nil connection")
			return
		}

		err = dest.Close()
		if err != nil {
			return
		}
	}(dest)

	if src == nil || dest == nil {
		err = errors.New("nil connection")
		return
	}

	bx, err := io.Copy(src, dest)
	if err != nil {
		//log.Trace().Err(err).Msg("copy")
	}
	go func() {
		p.LogPayload(MetricPayload{
			Protocol:         "https",
			UserID:           requestContext.UserID,
			BytesTransferred: bx,
			Direction:        direction,
			Region:           requestContext.Region,
			Host:             host,
		})
	}()
	return
}
