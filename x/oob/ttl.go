package oob

import (
	"fmt"
	"net"
	"net/netip"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func setTtl(conn *net.TCPConn, ttl int) (oldTTL int, err error) {
	addr, err := netip.ParseAddrPort(conn.RemoteAddr().String())
	if err != nil {
		return 0, err
	}

	switch {
	case addr.Addr().Is4():
		conn := ipv4.NewConn(conn)
		oldTTL, _ = conn.TTL()
		err = conn.SetTTL(ttl)
	case addr.Addr().Is6():
		conn := ipv6.NewConn(conn)
		oldTTL, _ = conn.HopLimit()
		err = conn.SetHopLimit(ttl)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to change TTL: %w", err)
	}

	if oldTTL == 0 {
		oldTTL = defaultTTL
	}

	return
}
