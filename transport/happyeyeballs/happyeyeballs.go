// Copyright 2024 Jigsaw Operations LLC
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

/*
Package happyeyeballs provides a simplified implementation of Happy Eyeballs v2.

The Happy Eyeballs v2 algorithm ([RFC 8305]) reduces the delay in establishing connectivity
in dual-stack (IPv4/IPv6) networks.

[RFC 8305]: https://datatracker.ietf.org/doc/html/rfc8305
*/
package happyeyeballs

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

// StreamDialer is a [transport.StreamDialer] that uses HappyEyeballs v2 to establish a connection
// to the destination address.
type StreamDialer struct {
	// The base dialer to establish connections. If nil, a direct TCP connection is established.
	Dialer transport.StreamDialer
	// Function to map a host name to IPv6 addresses. If nil, the system resolver is used.
	LookupIPv6 func(ctx context.Context, host string) ([]net.IP, error)
	// Function to map a host name to IPv4 addresses. If nil, the system resolver is used.
	LookupIPv4 func(ctx context.Context, host string) ([]net.IP, error)
}

var _ transport.StreamDialer = (*StreamDialer)(nil)

func (d *StreamDialer) dial(ctx context.Context, addr string) (transport.StreamConn, error) {
	if d.Dialer != nil {
		return d.Dialer.DialStream(ctx, addr)
	}
	return (&transport.TCPDialer{}).DialStream(ctx, addr)
}

func (d *StreamDialer) lookupIPv4(ctx context.Context, host string) ([]net.IP, error) {
	if d.LookupIPv4 != nil {
		return d.LookupIPv4(ctx, host)
	}
	return net.DefaultResolver.LookupIP(ctx, "ip4", host)
}

func (d *StreamDialer) lookupIPv6(ctx context.Context, host string) ([]net.IP, error) {
	if d.LookupIPv6 != nil {
		return d.LookupIPv6(ctx, host)
	}
	return net.DefaultResolver.LookupIP(ctx, "ip6", host)
}

func newClosedChan() <-chan struct{} {
	closedCh := make(chan struct{})
	close(closedCh)
	return closedCh
}

// DialStream implements [transport.StreamDialer].
func (d *StreamDialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %w", err)
	}
	if net.ParseIP(host) != nil {
		// Host is already an IP address, just dial the address.
		return d.dial(ctx, addr)
	}

	// Indicates to attempts that the search is done, so they don't get stuck.
	searchCtx, searchDone := context.WithCancel(context.Background())
	defer searchDone()

	// DOMAIN NAME LOOKUP SECTION
	// We start the IPv4 and IPv6 lookups in parallel, writing to lookup4Ch
	// and lookup6Ch when they are done.
	type LookupResult struct {
		IPs []net.IP
		Err error
	}
	lookup6Ch := make(chan LookupResult)
	lookup4Ch := make(chan LookupResult)
	go func(lookup6Ch chan<- LookupResult, host string) {
		ips, err := d.lookupIPv6(searchCtx, host)
		if err != nil {
			err = fmt.Errorf("failed to lookup IPv6 addresses: %w", err)
		}
		lookup6Ch <- LookupResult{ips, err}
		close(lookup6Ch)
	}(lookup6Ch, host)
	go func(lookup4Ch chan<- LookupResult, host string) {
		time.Sleep(100 * time.Millisecond)
		ips, err := d.lookupIPv4(searchCtx, host)
		if err != nil {
			err = fmt.Errorf("failed to lookup IPv4 addresses: %w", err)
		}
		lookup4Ch <- LookupResult{ips, err}
		close(lookup4Ch)
	}(lookup4Ch, host)

	// DIAL ATTEMPTS SECTION
	ips := []net.IP{}
	var dialErr error
	type DialResult struct {
		Conn transport.StreamConn
		Err  error
	}
	// Channel to wait for before a new dial attempt. It starts
	// with a closed channel that doesn't block because there's no
	// wait initially.
	var dialWaitCh <-chan struct{} = newClosedChan()
	var dialCh = make(chan DialResult)

	// We keep track of pending operations (lookups and IPs to dial) so we can stop when
	// there's no more work to wait for.
	for opsPending := 2; opsPending > 0; {
		var readyToDialCh <-chan struct{} = nil
		// Enable dial if there are IPs available.
		if len(ips) > 0 {
			readyToDialCh = dialWaitCh
		} else {
			readyToDialCh = nil
		}
		select {
		// Receive IPv4 results.
		case lookupRes := <-lookup4Ch:
			opsPending--
			// Set to nil to make the read on lookup4Ch block.
			lookup4Ch = nil
			if lookupRes.Err != nil {
				dialErr = errors.Join(lookupRes.Err)
				continue
			}
			opsPending += len(lookupRes.IPs)
			ips = append(ips, lookupRes.IPs...)
			// TODO: sort IPs as per https://datatracker.ietf.org/doc/html/rfc8305#section-4

		// Receive IPv6 results.
		case lookupRes := <-lookup6Ch:
			opsPending--
			// Set to nil to make the read on lookup6Ch block.
			lookup6Ch = nil
			if lookupRes.Err != nil {
				dialErr = errors.Join(lookupRes.Err)
				continue
			}
			opsPending += len(lookupRes.IPs)
			ips = append(ips, lookupRes.IPs...)
			// TODO: sort IPs as per https://datatracker.ietf.org/doc/html/rfc8305#section-4

		// Wait for new attempt done. Dial new IP address.
		case <-readyToDialCh:
			// The len(ips) > 0 condition before the select protects this.
			ip := ips[0]
			ips = ips[1:]
			// As per https://datatracker.ietf.org/doc/html/rfc8305#section-8
			waitCtx, waitDone := context.WithTimeout(searchCtx, 250*time.Millisecond)
			dialWaitCh = waitCtx.Done()
			go func(addr string, waitDone context.CancelFunc) {
				// Cancel the wait if the dial return early.
				defer waitDone()
				conn, err := d.dial(searchCtx, addr)
				select {
				case <-searchCtx.Done():
					if conn != nil {
						conn.Close()
					}
				case dialCh <- DialResult{conn, err}:
				}
			}(net.JoinHostPort(ip.String(), port), waitDone)

		// Receive dial result.
		case dialRes := <-dialCh:
			opsPending--
			if dialRes.Err != nil {
				dialErr = errors.Join(dialRes.Err)
				continue
			}
			return dialRes.Conn, nil

		// Dial has been canceled. Return.
		case <-searchCtx.Done():
			return nil, searchCtx.Err()
		}
	}
	return nil, dialErr
}
