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
	"sort"
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
	LookupIPv6 func(ctx context.Context, host string) ([]net.IP, error)
	// Function to map a host name to IPv4 addresses. If nil, net.DefaultResolver is used.
	LookupIPv4 func(ctx context.Context, host string) ([]net.IP, error)
}

var _ StreamDialer = (*HappyEyeballsStreamDialer)(nil)

func (d *HappyEyeballsStreamDialer) dial(ctx context.Context, addr string) (StreamConn, error) {
	if d.Dialer != nil {
		return d.Dialer.DialStream(ctx, addr)
	}
	return (&TCPDialer{}).DialStream(ctx, addr)
}

func (d *HappyEyeballsStreamDialer) lookupIPv4(ctx context.Context, host string) ([]netip.Addr, error) {
	var netIPs []net.IP
	var err error
	if d.LookupIPv4 != nil {
		netIPs, err = d.LookupIPv4(ctx, host)
	} else {
		netIPs, err = net.DefaultResolver.LookupIP(ctx, "ip4", host)
	}
	ips := make([]netip.Addr, 0, len(netIPs))
	for _, netIP := range netIPs {
		// Make sure it's a 4-byte IP, not IPv6-mapped IPv4.
		netIP = netIP.To4()
		if len(netIP) == 4 {
			ips = append(ips, netip.AddrFrom4([4]byte(netIP)))
		}
	}
	return ips, err
}

func (d *HappyEyeballsStreamDialer) lookupIPv6(ctx context.Context, host string) ([]netip.Addr, error) {
	var netIPs []net.IP
	var err error
	if d.LookupIPv6 != nil {
		netIPs, err = d.LookupIPv6(ctx, host)
	} else {
		netIPs, err = net.DefaultResolver.LookupIP(ctx, "ip6", host)
	}
	ips := make([]netip.Addr, 0, len(netIPs))
	for _, netIP := range netIPs {
		// Make sure it's an IPv6.
		netIP = netIP.To16()
		if len(netIP) == 16 {
			ips = append(ips, netip.AddrFrom16([16]byte(netIP)))
		}
	}
	return ips, err
}

func newClosedChan() <-chan struct{} {
	closedCh := make(chan struct{})
	close(closedCh)
	return closedCh
}

func mergeIPs(ipList []netip.Addr, newIPs ...netip.Addr) []netip.Addr {
	ipList = append(ipList, newIPs...)
	// TODO: sort IPs as per https://datatracker.ietf.org/doc/html/rfc8305#section-4
	sort.SliceStable(ipList, func(i, j int) bool {
		// Place IPv6 before IPv4.
		return ipList[i].Is6() && ipList[j].Is4()
	})
	return ipList
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
	searchCtx, searchDone := context.WithCancel(context.Background())
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
	// All the IPs to still attempt.
	ips := make([]netip.Addr, 0, 2)
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
		if len(ips) > 0 {
			if lookup6Ch != nil {
				fmt.Println("wait for Resolution delay")
				// IPv6 lookup not done yet. Set up Resolution Delay, as per
				// https://datatracker.ietf.org/doc/html/rfc8305#section-8
				resolutionDelayCtx, cancelResolutionDelay := context.WithTimeout(searchCtx, 50*time.Millisecond)
				defer cancelResolutionDelay()
				readyToDialCh = resolutionDelayCtx.Done()
			} else {
				fmt.Println("Dial wait")
				readyToDialCh = dialWaitCh
			}
		} else {
			fmt.Println("No IP to wait")
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
			ips = mergeIPs(ips, lookupRes.IPs...)

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
			ips = mergeIPs(ips, lookupRes.IPs...)

		// Wait for new attempt done. Dial new IP address.
		case <-readyToDialCh:
			// The len(ips) > 0 condition before the select protects this.
			ip := ips[0]
			ips = ips[1:]
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
	if dialErr != nil {
		return nil, dialErr
	}
	if lookupErr != nil {
		return nil, lookupErr
	}
	return nil, fmt.Errorf("address lookup returned no IPs")
}
