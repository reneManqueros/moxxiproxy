package models

import (
	"encoding/base64"
	"github.com/rs/zerolog/log"
	"net/http"

	"strings"
)

type RequestContext struct {
	RawUsername   string
	UserID        string
	Region        string
	Project       string
	Session       string
	Instance      string
	Authenticated bool
}

func (rc *RequestContext) FromRequest(request *http.Request) {
	if len(UserMap) == 0 {
		rc.Authenticated = true
	}

	if value, ok := request.Header["Proxy-Authorization"]; ok && len(value) > 0 {
		authHeader := value[0]
		authHeader = strings.TrimPrefix(authHeader, "Basic ")
		data, err := base64.StdEncoding.DecodeString(authHeader)
		if err != nil {
			log.Err(err).Str("method", "FromRequest").Msg("decoding auth header")
		}
		if userParts := strings.Split(string(data), ":"); len(userParts) > 1 {
			rc.RawUsername = userParts[0]
			authToken := userParts[1]
			rc.ParseUsername(rc.RawUsername)
			thisUser, userExists := Users{}.ByID(rc.UserID)
			if userExists == true && thisUser.UserID == rc.UserID && thisUser.AuthToken == authToken {
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
			rc.UserID = v
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
}
