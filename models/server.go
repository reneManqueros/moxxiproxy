package models

import (
	"bytes"
	"context"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const HTTP200 = "HTTP/1.1 200 Connection Established\r\n\r\n"
const HTTP407 = "407 Proxy Authentication Required"

type Proxy struct {
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

func (p *Proxy) setDialer(requestContext RequestContext) (string, string, proxy.Dialer) {
	var be string
	if requestContext.Instance != "" {
		be, _ = p.GetBackendByInstanceID(requestContext.Instance)
	} else if requestContext.Session != "" {
		p.SessionMutex.Lock()
		// ToDo: validate backend/exitnode still exists
		if val, ok := p.Sessions[requestContext.Session]; ok {
			be = val
		}
		p.SessionMutex.Unlock()
	} else if requestContext.Region != "" {
		be, _ = p.GetBackendByRegion(requestContext.Region)
	}

	// get one at random (default) or if the others failed
	if be == "" {
		be, _ = p.GetBackend()
	}

	network := "tcp4"
	if strings.Contains(be, ":") && len(be) > 15 {
		network = "tcp6"
	}
	addr, _ := net.ResolveTCPAddr(network, fmt.Sprintf("%s:0", be))

	if requestContext.Session != "" {
		p.SessionMutex.Lock()
		if _, ok := p.Sessions[requestContext.Session]; !ok {
			p.Sessions[requestContext.Session] = be
		}
		p.SessionMutex.Unlock()
	}
	return be, network, &net.Dialer{
		LocalAddr: addr,
		Timeout:   time.Duration(p.Timeout) * time.Second,
	}
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
		requestContext.FromRequest(p, request)
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
	p.setDialer(requestContext)
	transport := http.Transport{
		DialContext: p.Dialer.(interface {
			DialContext(context context.Context, network, address string) (net.Conn, error)
		}).DialContext,
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

func (p *Proxy) handleTunnel(responseWriter http.ResponseWriter, request *http.Request, requestContext RequestContext) {
	hijacker, ok := responseWriter.(http.Hijacker)
	if !ok {
		return
	}

	sourceConnection, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	_, network, thisDialer := p.setDialer(requestContext)
	destinationConnection, err := thisDialer.Dial(network, request.Host)
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
	p.getBackends()
	err := http.ListenAndServe(p.ListenAddress, http.HandlerFunc(p.handleRequest))
	if err != nil {
		log.Println(err)
	}
}
