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

package tlsfrag

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

// tlsFragDialer is a [transport.StreamDialer] that uses clientHelloFragWriter to fragment the first Client Hello
// record in a TLS session.
type tlsFragDialer struct {
	dialer transport.StreamDialer
	frag   FragFunc
	config *DialerConfiguration
}

// Compilation guard against interface implementation
var _ transport.StreamDialer = (*tlsFragDialer)(nil)

// FragFunc takes the content of the first [handshake record] in a TLS session as input, and returns an integer that
// represents the fragmentation point index. The input content excludes the 5-byte record header. The returned integer
// should be in range 0 to len(record)-1. The record will then be fragmented into two parts: record[:n] and record[n:].
// If the returned index is either ≤ 0 or ≥ len(record), no fragmentation will occur.
//
// [handshake record]: https://datatracker.ietf.org/doc/html/rfc8446#section-5.1
type FragFunc func(record []byte) int

// DialerConfiguration is an internal type used to configure the [transport.StreamDialer] created by
// [NewStreamDialerFunc]. You don't need to work with it directly. Instead, use the provided configuration functions
// like [WithTLSHostPortList].
type DialerConfiguration struct {
	addrs []*tlsAddrEntry
}

// DialerConfigurer updates the settings in the internal DialerConfiguration object. You can use the configuration
// functions such as [WithTLSHostPortList] to create configurers and then pass them to NewStreamDialerFunc to create a
// [transport.StreamDialer] with your desired configuration.
type DialerConfigurer func(*DialerConfiguration) error

// NewStreamDialerFunc creates a [transport.StreamDialer] that intercepts the initial [TLS Client Hello]
// [handshake record] and splits it into two separate records before sending them. The split point is determined by the
// callback function frag. The dialer then adds appropriate headers to each record and transmits them sequentially
// using the base dialer. Following the fragmented Client Hello, all subsequent data is passed through directly without
// modification.
//
// NewStreamDialerFunc allows specifying additional options to customize its behavior. By default, if no options are
// specified, the fragmentation only affects TLS Client Hello messages targeting port 443. All other network traffic,
// including non-TLS or non-Client Hello messages, or those targeting other ports, are passed through without any
// modification.
//
// [TLS Client Hello]: https://datatracker.ietf.org/doc/html/rfc8446#section-4.1.2
// [handshake record]: https://datatracker.ietf.org/doc/html/rfc8446#section-5.1
func NewStreamDialerFunc(base transport.StreamDialer, frag FragFunc, options ...DialerConfigurer) (transport.StreamDialer, error) {
	if base == nil {
		return nil, errors.New("base dialer must not be nil")
	}
	if frag == nil {
		return nil, errors.New("frag function must not be nil")
	}
	config := &DialerConfiguration{
		addrs: []*tlsAddrEntry{{"", 443}},
	}
	for _, opt := range options {
		if opt != nil {
			if err := opt(config); err != nil {
				return nil, err
			}
		}
	}
	return &tlsFragDialer{base, frag, config}, nil
}

// WithTLSHostPortList tells the [transport.StreamDialer] which connections to treat as TLS. Only connections matching
// entries in the tlsAddrs list will be treated as TLS traffic and fragmented accordingly.
//
// Each entry in the tlsAddrs list should be in the format "host:port", where "host" can be an IP address or a domain
// name, and "port" must be a valid port number. You can use empty string "" as the "host" to only match based on the
// port, and "0" as the "port" to match any port.
//
// The default list only includes ":443", meaning all traffic on port 443 is treated as TLS. This function overrides
// the entire list. So if you want to add entries, you need to include ":443" along with your additional entries.
//
// Matching for "host" is case-insensitive and strict. For example, "google.com:123" will only match "google.com" and
// not "www.google.com". Subdomain wildcards are not supported.
func WithTLSHostPortList(tlsAddrs []string) DialerConfigurer {
	return func(c *DialerConfiguration) error {
		addrs := make([]*tlsAddrEntry, 0, len(tlsAddrs))
		for _, hostport := range tlsAddrs {
			addr, err := parseTLSAddrEntry(hostport)
			if err != nil {
				return err
			}
			addrs = append(addrs, addr)
		}
		c.addrs = addrs
		return nil
	}
}

// Dial implements [transport.StreamConn].Dial. It establishes a connection to raddr in the format "host-or-ip:port".
//
// If raddr matches an entry in the valid TLS address list (which can be configured using [WithTLSHostPortList]), the
// initial TLS Client Hello record sent through the connection will be fragmented.
//
// If raddr is not listed in the valid TLS address list, the function simply utilizes the underlying base dialer's Dial
// function to establish the connection without any fragmentation.
func (d *tlsFragDialer) Dial(ctx context.Context, raddr string) (conn transport.StreamConn, err error) {
	conn, err = d.dialer.Dial(ctx, raddr)
	if err != nil {
		return
	}
	for _, addr := range d.config.addrs {
		if addr.matches(raddr) {
			w, err := newClientHelloFragWriter(conn, d.frag)
			if err != nil {
				return nil, err
			}
			return transport.WrapConn(conn, conn, w), nil
		}
	}
	return
}

// tlsAddrEntry reprsents an entry of the TLS traffic list. See [WithTLSHostPortList].
type tlsAddrEntry struct {
	host string
	port int
}

// parseTLSAddrEntry parses hostport in format "host:port" and returns the corresponding tlsAddrEntry.
func parseTLSAddrEntry(hostport string) (*tlsAddrEntry, error) {
	host, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("port must be within 0-65535: %w", strconv.ErrRange)
	}
	return &tlsAddrEntry{host, port}, nil
}

// matches returns whether raddr matches this entry.
func (e *tlsAddrEntry) matches(raddr string) bool {
	if len(e.host) == 0 && e.port == 0 {
		return true
	}
	host, portStr, err := net.SplitHostPort(raddr)
	if err != nil {
		return false
	}
	if len(e.host) > 0 && !strings.EqualFold(e.host, host) {
		return false
	}
	if e.port > 0 {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return false
		}
		if e.port != port {
			return false
		}
	}
	return true
}
