package models

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"
)

type RequestContext struct {
	Region        string
	Project       string
	Username      string
	Session       string
	Instance      string
	Authenticated bool
}

func (rc *RequestContext) FromRequest(proxy *Proxy, request *http.Request) {
	if value, ok := request.Header["Proxy-Authorization"]; ok && len(value) > 0 {
		authHeader := value[0]
		authHeader = strings.TrimPrefix(authHeader, "Basic ")
		data, err := base64.StdEncoding.DecodeString(authHeader)
		if err != nil {
			log.Println("error:", err)
		}
		if userParts := strings.Split(string(data), ":"); len(userParts) > 1 {
			userRaw := userParts[0]
			password := userParts[1]
			rc.ParseUsername(userRaw)

			if rc.Username == proxy.Username && password == proxy.Password {
				rc.Authenticated = true
			}
		}
		request.Header.Del("Proxy-Connection")
		request.Header.Del("Proxy-Authorization")
	}
}

func (rc *RequestContext) ParseUsername(userRaw string) {
	for index, v := range strings.Split(userRaw, "_") {
		if index == 0 {
			rc.Username = v
			continue
		}
		if kv := strings.Split(v, "-"); len(kv) == 2 {
			if kv[0] == "project" {
				rc.Project = kv[1]
			} else if kv[0] == "region" {
				rc.Region = kv[1]
			} else if kv[0] == "session" {
				rc.Session = kv[1]
			} else if kv[0] == "instance" {
				rc.Instance = kv[1]
			}
		}
	}
	log.Println(rc)
}
