package models

import (
	"log"
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
