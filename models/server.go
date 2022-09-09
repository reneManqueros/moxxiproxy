package models

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"io/ioutil"
	"log"
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
	IsUpstream    bool
	BackendsFile  string
	ListenAddress string
	Backends      []string
	Sessions      map[string]string
	ExitNodes     struct {
		All          []ExitNode
		ByRegion     map[string][]ExitNode
		ByInstanceID map[string]ExitNode
	}
	SessionMutex *sync.Mutex
	Username     string
	Password     string
	Whitelist    string
	Dialer       proxy.Dialer
	Mutex        *sync.Mutex
	Timeout      int
	IsVerbose    bool
}

func (p *Proxy) GetExitNode(requestContext RequestContext) (ExitNode, string) {
	var exitNode ExitNode
	if requestContext.Instance != "" {
		exitNode, _ = p.ByInstanceID(requestContext.Instance)
	} else if requestContext.Region != "" {
		exitNode, _ = p.ByRegion(requestContext.Region)
	}

	// get one at random (default) or if the others failed
	if exitNode.Interface == "" && exitNode.Upstream == "" {
		exitNode, _ = p.ByRandom()
	}
	backend := exitNode.Interface
	if p.IsUpstream == true {
		backend = exitNode.Upstream
	}

	return exitNode, backend
}

func (p *Proxy) setDialer(requestContext RequestContext, isClearText bool) (ExitNode, string, proxy.Dialer) {
	exitNode, backend := p.GetExitNode(requestContext)

	network := "tcp4"
	// ToDo: implement this in a cleaner way
	//if strings.Contains(backend, ":") && len(backend) > 15 && ps.IsUpstream == false {
	//	network = "tcp6"
	//}

	format := `%s:0`
	if p.IsUpstream && isClearText == false {
		format = `%s`
	}
	addr, _ := net.ResolveTCPAddr(network, fmt.Sprintf(format, backend))

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
	if p.isInWhitelist(request.RemoteAddr) == false {
		return
	}
	requestContext := RequestContext{}

	passedAuthentication := false
	if p.Username == "" && p.Password == "" {
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

		if p.IsVerbose == true {
			log.Println("invalid request")
		}
	}
}

func (p *Proxy) handleHTTP(responseWriter http.ResponseWriter, request *http.Request, requestContext RequestContext) {
	exitNode, _, thisDialer := p.setDialer(requestContext, true)
	transport := http.Transport{
		DialContext: thisDialer.(interface {
			DialContext(context context.Context, network, address string) (net.Conn, error)
		}).DialContext,
	}

	if p.IsUpstream {
		u, _ := url.Parse(exitNode.Upstream)
		transport.Proxy = http.ProxyURL(u)
	}
	response, err := transport.RoundTrip(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	copyHeader(responseWriter.Header(), response.Header)
	responseWriter.WriteHeader(response.StatusCode)
	_, _ = io.Copy(responseWriter, response.Body)
}

func (p *Proxy) getUpstream(upstream string, addr string) (net.Conn, error) {
	network := "tcp"

	connectReq := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: make(http.Header),
	}

	c, err := net.DialTimeout(network, upstream, time.Duration(10)*time.Second)
	if err != nil {
		return nil, err
	}
	connectReq.Write(c)
	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, connectReq)
	if err != nil {
		c.Close()
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
		destinationConnection, err = p.getUpstream(exitNode.Upstream, request.Host)
	} else {
		destinationConnection, err = thisDialer.Dial(network, request.Host)
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

	go copyIO(sourceConnection, destinationConnection)
	go copyIO(destinationConnection, sourceConnection)
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
		Body:          ioutil.NopCloser(bytes.NewBuffer([]byte(HTTP407))),
		ContentLength: int64(len(HTTP407)),
	}

	sourceConnection, _, err := hijacker.Hijack()
	_ = authRequiredResponse.Write(sourceConnection)
	_ = sourceConnection.Close()
	if err != nil {
		log.Println("Cannot hijack connection " + err.Error())
	}
}

func (p *Proxy) Run() {
	p.ExitNodesFromDisk()
	err := http.ListenAndServe(p.ListenAddress, http.HandlerFunc(p.handleRequest))
	if err != nil {
		log.Println(err)
	}
}
