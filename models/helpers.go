package models

import (
	"errors"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"net/http"
)

func copyHeader(dest, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dest.Add(k, v)
		}
	}
}

func copyIO(src, dest net.Conn) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			log.Err(err).Str("method", "copyIO").Msg("UNHANDLED ERROR!")
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
	_, err = io.Copy(src, dest)
	if err != nil {
		return
	}
	return
}
