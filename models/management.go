package models

import (
	"log"
	"net"
	"strings"
	"time"
)

type Management struct {
	Server        *Proxy
	ListenAddress string
}

func (m Management) Send(message string) {
	conn, err := net.DialTimeout("tcp", m.ListenAddress, 300*time.Millisecond)
	if err != nil {
		log.Println(err)
	}
	_, err = conn.Write([]byte(message))
	if err != nil {
		log.Println(err)
	}
}

func (m Management) Listen() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", m.ListenAddress)
	if err != nil {
		log.Println(err)
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Println(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
		}

		buf := make([]byte, 512)
		n, err := conn.Read(buf[0:])
		if err != nil {
			log.Println(err)
			continue
		}

		message := string(buf[:n])
		message = strings.TrimSpace(message)
		if len(message) < 7 {
			continue
		}
		if strings.HasPrefix(message, "+") {
			address := strings.TrimPrefix(message, "+")
			m.Server.Add(address)
		} else if strings.HasPrefix(message, "-") {
			address := strings.TrimPrefix(message, "-")
			m.Server.Remove(address)
		}
	}
}
