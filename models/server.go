package models

import (
	"bytes"
	"context"
	"encoding/base64"
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
	Username      string
	Password      string
	Whitelist     string
	Dialer        proxy.Dialer
	Mutex         *sync.Mutex
	Timeout       int
	IsVerbose     bool
}

func (p *Proxy) setDialer() (string, string) {
	be, _ := p.GetBackend()
	network := "tcp4"
	if strings.Contains(be, ":") && len(be) > 15 {
		network = "tcp6"
	}
	addr, _ := net.ResolveTCPAddr(network, fmt.Sprintf("%s:0", be))

	p.Dialer = &net.Dialer{
		LocalAddr: addr,
		Timeout:   time.Duration(p.Timeout) * time.Second,
	}
	return be, network
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

	isValid := false
	if p.Username == "" && p.Password == "" {
		isValid = true
	}

	if value, ok := request.Header["Proxy-Authorization"]; ok && len(value) > 0 && isValid == false {
		authHeader := value[0]
		authHeader = strings.TrimPrefix(authHeader, "Basic ")
		data, err := base64.StdEncoding.DecodeString(authHeader)
		if err != nil {
			log.Println("error:", err)
		}
		if userParts := strings.Split(string(data), ":"); len(userParts) > 1 {
			username := userParts[0]
			password := userParts[1]
			if username == p.Username && password == p.Password {
				isValid = true
			}
		}
		request.Header.Del("Proxy-Connection")
		request.Header.Del("Proxy-Authorization")
	}

	if isValid == true {
		if request.Method == http.MethodConnect {
			p.handleTunnel(responseWriter, request)
		} else {
			p.handleHTTP(responseWriter, request)
		}
	} else {
		p.handleProxyAuthRequired(responseWriter, request)

		if p.IsVerbose == true {
			log.Println("invalid request")
		}
	}
}

func (p *Proxy) handleHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	p.setDialer()
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

func (p *Proxy) handleTunnel(responseWriter http.ResponseWriter, request *http.Request) {
	hijacker, ok := responseWriter.(http.Hijacker)
	if !ok {
		return
	}

	sourceConnection, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	_, network := p.setDialer()
	destinationConnection, err := p.Dialer.Dial(network, request.Host)
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
	p.getBackends()
	err := http.ListenAndServe(p.ListenAddress, http.HandlerFunc(p.handleRequest))
	if err != nil {
		log.Println(err)
	}
}
