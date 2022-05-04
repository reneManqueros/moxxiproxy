package models

import (
	"context"
	"encoding/base64"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const HTTP200 = "HTTP/1.1 200 Connection Established\r\n\r\n"

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
	p.setDialer()
	destinationConnection, err := p.Dialer.Dial("tcp", request.Host)
	if err != nil {
		_ = sourceConnection.Close()
		return
	}
	_, _ = sourceConnection.Write([]byte(HTTP200))

	go copyIO(sourceConnection, destinationConnection)
	go copyIO(destinationConnection, sourceConnection)
}

func (p *Proxy) setDialer() {
	be, _ := p.GetBackend()
	addr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", be))

	p.Dialer = &net.Dialer{
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
		if p.IsVerbose == true {
			log.Println("invalid request")
		}
	}
}

func copyHeader(dest, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dest.Add(k, v)
		}
	}
}

func copyIO(src, dest net.Conn) {
	defer func(src net.Conn) {
		err := src.Close()
		if err != nil {
			return
		}
	}(src)

	defer func(dest net.Conn) {
		err := dest.Close()
		if err != nil {
			return
		}
	}(dest)

	_, err := io.Copy(src, dest)
	if err != nil {
		return
	}
}

func (p *Proxy) Run() {
	p.getBackends()
	err := http.ListenAndServe(p.ListenAddress, http.HandlerFunc(p.handleRequest))
	if err != nil {
		log.Println(err)
	}
}
