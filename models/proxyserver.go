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
	"net/url"
	"strings"
	"sync"
	"time"
)

const HTTP200 = "HTTP/1.1 200 Connection Established\r\n\r\n"
const HTTP407 = "407 ProxyServer Authentication Required"

type ProxyServer struct {
	ListenAddress string
	Whitelist     string
	Sessions      map[string]ExitNode
	ExitNodes     struct {
		All          []ExitNode
		ByRegion     map[string][]ExitNode
		ByInstanceID map[string]ExitNode
	}
	SessionMutex *sync.Mutex
	Users        []User
	Mutex        *sync.Mutex
	ConfigFiles  struct {
		Users string
		Nodes string
	}
	Timeout    int
	IsVerbose  bool
	IsUpstream bool
}

func (ps *ProxyServer) GetExitNode(requestContext RequestContext) (ExitNode, string) {
	var exitNode ExitNode
	if requestContext.Instance != "" {
		exitNode, _ = ps.ByInstanceID(requestContext.Instance)
	} else if requestContext.Session != "" {
		ps.SessionMutex.Lock()
		// ToDo: validate backend/exitnode still exists
		if val, ok := ps.Sessions[requestContext.Session]; ok {
			exitNode = val
		}
		ps.SessionMutex.Unlock()
	} else if requestContext.Region != "" {
		exitNode, _ = ps.ByRegion(requestContext.Region)
	}

	// get one at random (default) or if the others failed
	if exitNode.Interface == "" && exitNode.Upstream == "" {
		exitNode, _ = ps.ByRandom()
	}
	backend := exitNode.Interface
	if ps.IsUpstream == true {
		backend = exitNode.Upstream
	}

	return exitNode, backend
}

func (ps *ProxyServer) setDialer(requestContext RequestContext) (ExitNode, string, proxy.Dialer) {
	exitNode, backend := ps.GetExitNode(requestContext)

	// ToDo: implement this in a cleaner way
	network := "tcp4"
	if strings.Contains(backend, ":") && len(backend) > 15 && ps.IsUpstream == false {
		network = "tcp6"
	}
	addr, _ := net.ResolveTCPAddr(network, fmt.Sprintf("%s:0", backend))

	thisDialer := &net.Dialer{
		LocalAddr: addr,
		Timeout:   time.Duration(ps.Timeout) * time.Second,
	}

	if requestContext.Session != "" {
		ps.SessionMutex.Lock()
		if _, ok := ps.Sessions[requestContext.Session]; !ok {
			ps.Sessions[requestContext.Session] = exitNode
		}
		ps.SessionMutex.Unlock()
	}
	return exitNode, network, thisDialer
}

func (ps *ProxyServer) isInWhitelist(requestAddress string) bool {
	if ps.Whitelist == "" {
		return true
	}
	if addressParts := strings.Split(requestAddress, ":"); len(addressParts) > 0 {
		parsedRequestAddress := addressParts[0]
		for _, whitelistItem := range strings.Split(ps.Whitelist, ",") {
			if strings.HasPrefix(parsedRequestAddress, whitelistItem) {
				return true
			}
		}
	}
	return false
}

func (ps *ProxyServer) handleRequest(responseWriter http.ResponseWriter, request *http.Request) {
	if ps.isInWhitelist(request.RemoteAddr) == false {
		return
	}
	requestContext := RequestContext{}
	requestContext.FromRequest(request)

	if requestContext.Authenticated == true {
		if request.Method == http.MethodConnect {
			ps.handleTunnel(responseWriter, request, requestContext)
		} else {
			ps.handleHTTP(responseWriter, request, requestContext)
		}
	} else {
		ps.handleProxyAuthRequired(responseWriter, request)
		if ps.IsVerbose == true {
			log.Println("invalid request")
		}
	}
}

func (ps *ProxyServer) getUpstreamProxyURL(requestContext RequestContext, upstream string) func(r *http.Request) (*url.URL, error) {
	upstreamToken := "Upstreamed"
	baseURL := fmt.Sprintf(`http://%s:%s@%s`, requestContext.RawUsername, upstreamToken, upstream)
	u, _ := url.Parse(baseURL)
	return http.ProxyURL(u)
}

func (ps *ProxyServer) handleHTTP(responseWriter http.ResponseWriter, request *http.Request, requestContext RequestContext) {
	exitNode, _, thisDialer := ps.setDialer(requestContext)
	transport := http.Transport{
		DialContext: thisDialer.(interface {
			DialContext(context context.Context, network, address string) (net.Conn, error)
		}).DialContext,
	}

	if ps.IsUpstream == true {
		transport.Proxy = ps.getUpstreamProxyURL(requestContext, exitNode.Upstream)
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

func (ps *ProxyServer) handleTunnel(responseWriter http.ResponseWriter, request *http.Request, requestContext RequestContext) {
	exitNode, network, thisDialer := ps.setDialer(requestContext)
	var destinationConnection net.Conn
	var err error
	if ps.IsUpstream == true {
		transport := http.Transport{
			DialContext: thisDialer.(interface {
				DialContext(context context.Context, network, address string) (net.Conn, error)
			}).DialContext,
		}
		transport.Proxy = ps.getUpstreamProxyURL(requestContext, exitNode.Upstream)
		ctx := context.Background()
		destinationConnection, err = transport.DialContext(ctx, network, request.Host)
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

func (ps *ProxyServer) handleProxyAuthRequired(responseWriter http.ResponseWriter, request *http.Request) {
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
			"ProxyServer-Authenticate": []string{"Basic"},
			"ProxyServer-Connection":   []string{"close"},
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

func (ps *ProxyServer) Run() {
	ps.ExitNodesFromDisk(ps.ConfigFiles.Nodes)
	Users{}.Load(ps.ConfigFiles.Users)
	err := http.ListenAndServe(ps.ListenAddress, http.HandlerFunc(ps.handleRequest))
	if err != nil {
		log.Println(err)
	}
}
