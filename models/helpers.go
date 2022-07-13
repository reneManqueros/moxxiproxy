package models

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
)

func ExpandCIDRRange(cidrRange string) (ips []string, err error) {
	prefix, err := netip.ParsePrefix(cidrRange)
	if err != nil {
		log.Println(err)
	}

	for addr := prefix.Addr(); prefix.Contains(addr); addr = addr.Next() {
		ips = append(ips, addr.String())
	}

	if len(ips) < 2 {
		return ips, nil
	}

	return ips[1 : len(ips)-1], nil
}

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
			log.Println("UNHANDLED ERROR! :", err)
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
