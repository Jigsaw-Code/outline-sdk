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

package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"
)

/*
HappyEyeballsStreamDialer is a [StreamDialer] that uses [Happy Eyeballs v2] to establish a connection
to the destination address.

Happy Eyeballs v2 reduces the connection delay when compared to v1, with significant differences when one of the
address lookups times out. V1 will wait for both the IPv4 and IPv6 lookups to return before attempting connections,
while V2 starts connections as soon as it gets a lookup result, with a slight delay if IPv4 arrives before IPv6.

Go and most platforms provide V1 only, so you will benefit from using the HappyEyeballsStreamDialer in place of the
standard dialer, even if you are not using custom transports.

[Happy Eyeballs v2]: https://datatracker.ietf.org/doc/html/rfc8305
*/
type HappyEyeballsStreamDialer struct {
	// The base dialer to establish connections. If nil, a direct TCP connection is established.
	Dialer StreamDialer
	// Function to map a host name to IPv6 addresses. If nil, net.DefaultResolver is used.
	LookupIPv6 func(ctx context.Context, host string) ([]netip.Addr, error)
	// Function to map a host name to IPv4 addresses. If nil, net.DefaultResolver is used.
	LookupIPv4 func(ctx context.Context, host string) ([]netip.Addr, error)
}

var _ StreamDialer = (*HappyEyeballsStreamDialer)(nil)

func (d *HappyEyeballsStreamDialer) dial(ctx context.Context, addr string) (StreamConn, error) {
	if d.Dialer != nil {
		return d.Dialer.DialStream(ctx, addr)
	}
	return (&TCPDialer{}).DialStream(ctx, addr)
}

func (d *HappyEyeballsStreamDialer) lookupIPv4(ctx context.Context, host string) ([]netip.Addr, error) {
	if d.LookupIPv4 != nil {
		return d.LookupIPv4(ctx, host)
	}
	return net.DefaultResolver.LookupNetIP(ctx, "ip4", host)
}

func (d *HappyEyeballsStreamDialer) lookupIPv6(ctx context.Context, host string) ([]netip.Addr, error) {
	if d.LookupIPv6 != nil {
		return d.LookupIPv6(ctx, host)
	}
	return net.DefaultResolver.LookupNetIP(ctx, "ip6", host)
}

func newClosedChan() <-chan struct{} {
	closedCh := make(chan struct{})
	close(closedCh)
	return closedCh
}

// DialStream implements [StreamDialer].
func (d *HappyEyeballsStreamDialer) DialStream(ctx context.Context, addr string) (StreamConn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %w", err)
	}
	if net.ParseIP(host) != nil {
		// Host is already an IP address, just dial the address.
		return d.dial(ctx, addr)
	}

	// Indicates to attempts that the search is done, so they don't get stuck.
	searchCtx, searchDone := context.WithCancel(ctx)
	defer searchDone()

	// DOMAIN NAME LOOKUP SECTION
	// We start the IPv4 and IPv6 lookups in parallel, writing to lookup4Ch
	// and lookup6Ch when they are done.
	type LookupResult struct {
		IPs []netip.Addr
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
		ips, err := d.lookupIPv4(searchCtx, host)
		if err != nil {
			err = fmt.Errorf("failed to lookup IPv4 addresses: %w", err)
		}
		lookup4Ch <- LookupResult{ips, err}
		close(lookup4Ch)
	}(lookup4Ch, host)

	// DIAL ATTEMPTS SECTION
	// We keep IPv4s and IPv6 separate and track the last one attempted so we can
	// alternate the address family in the connection attempts.
	var ip4s []netip.Addr
	var ip6s []netip.Addr
	var lastDialed netip.Addr
	// Keep track of the lookup and dial errors separately. We prefer the dial errors
	// when returning.
	var lookupErr error
	var dialErr error
	type DialResult struct {
		Conn StreamConn
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
		if len(ip6s) > 0 {
			readyToDialCh = dialWaitCh
		} else if len(ip4s) > 0 {
			if lookup6Ch != nil && !lastDialed.IsValid() {
				// IPv6 lookup not done yet and we havent' waited for it. Set up Resolution Delay, as per
				// https://datatracker.ietf.org/doc/html/rfc8305#section-8
				resolutionDelayCtx, cancelResolutionDelay := context.WithTimeout(searchCtx, 50*time.Millisecond)
				defer cancelResolutionDelay()
				readyToDialCh = resolutionDelayCtx.Done()
			} else {
				readyToDialCh = dialWaitCh
			}
		} else {
			// No IPs. Keep dial disabled.
			readyToDialCh = nil
		}
		select {
		// Receive IPv6 results.
		case lookupRes := <-lookup6Ch:
			opsPending--
			// Set to nil to make the read on lookup6Ch block and to signal IPv6 lookup is done.
			lookup6Ch = nil
			if lookupRes.Err != nil {
				lookupErr = errors.Join(lookupRes.Err)
				continue
			}
			opsPending += len(lookupRes.IPs)
			// TODO: sort IPs as per https://datatracker.ietf.org/doc/html/rfc8305#section-4
			ip6s = lookupRes.IPs

		// Receive IPv4 results.
		case lookupRes := <-lookup4Ch:
			opsPending--
			// Set to nil to make the read on lookup4Ch block and to signal IPv4 lookup is done.
			lookup4Ch = nil
			if lookupRes.Err != nil {
				lookupErr = errors.Join(lookupRes.Err)
				continue
			}
			opsPending += len(lookupRes.IPs)
			// TODO: sort IPs as per https://datatracker.ietf.org/doc/html/rfc8305#section-4
			ip4s = lookupRes.IPs

		// Wait for new attempt done. Dial new IP address.
		case <-readyToDialCh:
			var toDial netip.Addr
			if len(ip6s) == 0 || (lastDialed.Is6() && len(ip4s) > 0) {
				toDial = ip4s[0]
				ip4s = ip4s[1:]
			} else {
				toDial = ip6s[0]
				ip6s = ip6s[1:]
			}
			// Connection Attempt Delay, as per https://datatracker.ietf.org/doc/html/rfc8305#section-8
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
			}(net.JoinHostPort(toDial.String(), port), waitDone)
			lastDialed = toDial

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
	if dialErr != nil {
		return nil, dialErr
	}
	if lookupErr != nil {
		return nil, lookupErr
	}
	return nil, fmt.Errorf("address lookup returned no IPs")
}
