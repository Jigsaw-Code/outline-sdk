// Copyright 2023 Jigsaw Operations LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package socks5

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

// ReplyCode is a byte-unsigned number that represents a SOCKS error as indicated in the REP field of the server response.
type ReplyCode byte

// SOCKS reply codes, as enumerated in https://datatracker.ietf.org/doc/html/rfc1928#section-6.
const (
	ErrGeneralServerFailure          = ReplyCode(0x01)
	ErrConnectionNotAllowedByRuleset = ReplyCode(0x02)
	ErrNetworkUnreachable            = ReplyCode(0x03)
	ErrHostUnreachable               = ReplyCode(0x04)
	ErrConnectionRefused             = ReplyCode(0x05)
	ErrTTLExpired                    = ReplyCode(0x06)
	ErrCommandNotSupported           = ReplyCode(0x07)
	ErrAddressTypeNotSupported       = ReplyCode(0x08)
)

var _ error = (ReplyCode)(0)

// Error returns a human-readable description of the error, based on the SOCKS5 RFC.
func (e ReplyCode) Error() string {
	switch e {
	case ErrGeneralServerFailure:
		return "general SOCKS server failure"
	case ErrConnectionNotAllowedByRuleset:
		return "connection not allowed by ruleset"
	case ErrNetworkUnreachable:
		return "network unreachable"
	case ErrHostUnreachable:
		return "host unreachable"
	case ErrConnectionRefused:
		return "connection refused"
	case ErrTTLExpired:
		return "TTL expired"
	case ErrCommandNotSupported:
		return "command not supported"
	case ErrAddressTypeNotSupported:
		return "address type not supported"
	default:
		return "reply code " + strconv.Itoa(int(e))
	}
}

// SOCKS address types defined at https://datatracker.ietf.org/doc/html/rfc1928#section-5
const (
	// Address is an IPv4 address (SOCKS4, SOCKS4a and SOCKS5).
	addrTypeIPv4 = 0x01
	// Address is a domain name (SOCKS4a and SOCKS5)
	addrTypeDomainName = 0x03
	// Address is an IPv6 address (SOCKS5 only).
	addrTypeIPv6 = 0x04
)

// writeSOCKS5Address adds the address to buffer b in SOCKS5 format,
// as specified in https://datatracker.ietf.org/doc/html/rfc1928#section-4
func writeSOCKS5Address(w io.Writer, address string) error {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	portNum, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}
	// The SOCKS address format is as follows:
	//     +------+----------+----------+
	//     | ATYP | DST.ADDR | DST.PORT |
	//     +------+----------+----------+
	//     |  1   | Variable |    2     |
	//     +------+----------+----------+
	// See https://datatracker.ietf.org/doc/html/rfc1928#section-5 for DST.ADDR details.
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			w.Write([]byte{addrTypeIPv4})
			w.Write(ip4)
		} else if ip6 := ip.To16(); ip6 != nil {
			w.Write([]byte{addrTypeIPv6})
			w.Write(ip6)
		} else {
			// This should never happen.
			return errors.New("IP address not IPv4 or IPv6")
		}
	} else {
		if len(host) > 255 {
			return fmt.Errorf("domain name length = %v is over 255", len(host))
		}
		w.Write([]byte{addrTypeDomainName, byte(len(host))})
		w.Write([]byte(host))
	}
	w.Write([]byte{byte(portNum >> 8), byte(portNum)})
	return nil
}
