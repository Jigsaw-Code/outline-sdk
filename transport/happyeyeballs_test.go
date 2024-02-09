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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type collectStreamDialer struct {
	Dialer StreamDialer
	Addrs  []string
}

func (d *collectStreamDialer) DialStream(ctx context.Context, addr string) (StreamConn, error) {
	d.Addrs = append(d.Addrs, addr)
	return d.Dialer.DialStream(ctx, addr)
}

var nilDialer = FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
	return nil, nil
})

func newErrorStreamDialer(err error) StreamDialer {
	return FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
		return nil, err
	})
}

func TestHappyEyeballsStreamDialer_DialStream(t *testing.T) {
	t.Run("Works with IPv4 hosts", func(t *testing.T) {
		baseDialer := collectStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{Dialer: &baseDialer}
		_, err := dialer.DialStream(context.Background(), "8.8.8.8:53")
		require.NoError(t, err)
		require.Equal(t, []string{"8.8.8.8:53"}, baseDialer.Addrs)
	})

	t.Run("Works with IPv6 hosts", func(t *testing.T) {
		baseDialer := collectStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{Dialer: &baseDialer}
		_, err := dialer.DialStream(context.Background(), "[2001:4860:4860::8888]:53")
		require.NoError(t, err)
		require.Equal(t, []string{"[2001:4860:4860::8888]:53"}, baseDialer.Addrs)
	})

	t.Run("Prefer IPv6", func(t *testing.T) {
		baseDialer := collectStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			Resolve: func(ctx context.Context, hostname string) <-chan HappyEyeballsResolution {
				resultsCh := make(chan HappyEyeballsResolution, 2)
				resultsCh <- HappyEyeballsResolution{[]netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil}
				resultsCh <- HappyEyeballsResolution{[]netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil}
				close(resultsCh)
				return resultsCh
			},
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.NoError(t, err)
		require.Equal(t, []string{"[2001:4860:4860::8888]:53"}, baseDialer.Addrs)
	})

	t.Run("Prefer IPv6 if there's a small delay", func(t *testing.T) {
		baseDialer := collectStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			Resolve: NewDualStackHappyEyeballsResolver(
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					time.Sleep(10 * time.Millisecond)
					return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil
				},
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
				},
			),
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.NoError(t, err)
		require.Equal(t, []string{"[2001:4860:4860::8888]:53"}, baseDialer.Addrs)
	})

	t.Run("Use IPv4 if IPv6 hangs, with fallback", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		baseDialer := collectStreamDialer{Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
			if addr == "8.8.8.8:53" {
				return nil, errors.New("failed to dial")
			}
			return nil, nil
		})}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			Resolve: NewDualStackHappyEyeballsResolver(
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					// Make it hang.
					<-ctx.Done()
					return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil
				},
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("8.8.4.4")}, nil
				},
			),
		}
		_, err := dialer.DialStream(ctx, "dns.google:53")
		require.NoError(t, err)
		require.Equal(t, []string{"8.8.8.8:53", "8.8.4.4:53"}, baseDialer.Addrs)
	})

	t.Run("Use IPv6 if IPv4 fails", func(t *testing.T) {
		baseDialer := collectStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			Resolve: NewDualStackHappyEyeballsResolver(
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					time.Sleep(10 * time.Millisecond)
					return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil

				},
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return nil, errors.New("lookup failed")
				},
			),
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.NoError(t, err)
		require.Equal(t, []string{"[2001:4860:4860::8888]:53"}, baseDialer.Addrs)
	})

	t.Run("Use IPv4 if IPv6 fails", func(t *testing.T) {
		baseDialer := collectStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			Resolve: NewDualStackHappyEyeballsResolver(
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return nil, errors.New("lookup failed")
				},
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					time.Sleep(10 * time.Millisecond)
					return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
				},
			),
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.NoError(t, err)
		require.Equal(t, []string{"8.8.8.8:53"}, baseDialer.Addrs)
	})

	t.Run("No dial if lookup fails", func(t *testing.T) {
		baseDialer := collectStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			Resolve: NewDualStackHappyEyeballsResolver(
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return nil, errors.New("lookup failed")
				},
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return nil, errors.New("lookup failed")
				},
			),
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.Error(t, err)
		require.Empty(t, baseDialer.Addrs)
	})

	t.Run("No IPs returned", func(t *testing.T) {
		baseDialer := collectStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			Resolve: NewDualStackHappyEyeballsResolver(
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return []netip.Addr{}, nil
				},
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return []netip.Addr{}, nil
				},
			),
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.Error(t, err)
		require.Empty(t, baseDialer.Addrs)
	})

	t.Run("Fallback to second address", func(t *testing.T) {
		var dialedAddrs []string
		dialer := HappyEyeballsStreamDialer{
			Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
				dialedAddrs = append(dialedAddrs, addr)
				if addr == "[2001:4860:4860::8888]:53" {
					return nil, errors.New("dial failed")
				}
				return nil, nil
			}),
			Resolve: NewDualStackHappyEyeballsResolver(
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil
				},
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
				},
			),
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.NoError(t, err)
		require.Equal(t, []string{"[2001:4860:4860::8888]:53", "8.8.8.8:53"}, dialedAddrs)
	})

	t.Run("IP order", func(t *testing.T) {
		dialFailErr := errors.New("failed to dial")
		baseDialer := collectStreamDialer{Dialer: newErrorStreamDialer(dialFailErr)}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			Resolve: NewDualStackHappyEyeballsResolver(
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return []netip.Addr{
						netip.MustParseAddr("::1"),
						netip.MustParseAddr("::2"),
						netip.MustParseAddr("::3"),
					}, nil
				},
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return []netip.Addr{
						netip.MustParseAddr("1.1.1.1"),
						netip.MustParseAddr("2.2.2.2"),
						netip.MustParseAddr("3.3.3.3"),
					}, nil
				},
			),
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.ErrorIs(t, err, dialFailErr)
		require.Equal(t, []string{"[::1]:53", "1.1.1.1:53", "[::2]:53", "2.2.2.2:53", "[::3]:53", "3.3.3.3:53"}, baseDialer.Addrs)
	})

	t.Run("Cancelled lookups", func(t *testing.T) {
		var hold sync.WaitGroup
		hold.Add(1)
		defer hold.Done()
		ctx, cancel := context.WithCancel(context.Background())
		baseDialer := collectStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			Resolve: NewDualStackHappyEyeballsResolver(
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					hold.Wait()
					return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil
				},
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					cancel()
					hold.Wait()
					return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
				},
			),
		}
		_, err := dialer.DialStream(ctx, "dns.google:53")
		require.ErrorIs(t, err, context.Canceled)
		require.Empty(t, baseDialer.Addrs)
	})

	t.Run("Cancelled dial", func(t *testing.T) {
		var hold sync.WaitGroup
		hold.Add(1)
		defer hold.Done()
		ctx, cancel := context.WithCancel(context.Background())
		baseDialer := collectStreamDialer{Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
			go cancel()
			hold.Wait()
			return nil, nil
		})}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			Resolve: NewDualStackHappyEyeballsResolver(
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil
				},
				func(ctx context.Context, host string) ([]netip.Addr, error) {
					return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
				},
			),
		}
		_, err := dialer.DialStream(ctx, "dns.google:53")
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, []string{"[2001:4860:4860::8888]:53"}, baseDialer.Addrs)
	})
}

func ExampleHappyEyeballsStreamDialer() {
	ips := []netip.Addr{}
	dialer := HappyEyeballsStreamDialer{
		Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
			ips = append(ips, netip.MustParseAddrPort(addr).Addr())
			return nil, errors.New("not implemented")
		}),
		Resolve: NewDualStackHappyEyeballsResolver(
			func(ctx context.Context, hostname string) ([]netip.Addr, error) {
				return net.DefaultResolver.LookupNetIP(ctx, "ip6", hostname)
			},
			func(ctx context.Context, hostname string) ([]netip.Addr, error) {
				return net.DefaultResolver.LookupNetIP(ctx, "ip4", hostname)
			},
		),
	}
	dialer.DialStream(context.Background(), "dns.google:53")
	// Sort list so that output is consistent.
	sort.SliceStable(ips, func(i, j int) bool { return ips[i].Less(ips[j]) })
	fmt.Println("Sorted IPs:", ips)
	// Output:
	// Sorted IPs: [8.8.4.4 8.8.8.8 2001:4860:4860::8844 2001:4860:4860::8888]
}

func ExampleHappyEyeballsStreamDialer_fixedResolution() {
	ips := []netip.Addr{}
	dialer := HappyEyeballsStreamDialer{
		Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
			ips = append(ips, netip.MustParseAddrPort(addr).Addr())
			return nil, errors.New("not implemented")
		}),
		Resolve: func(ctx context.Context, hostname string) <-chan HappyEyeballsResolution {
			resultCh := make(chan HappyEyeballsResolution, 1)
			resultCh <- HappyEyeballsResolution{
				IPs: []netip.Addr{
					netip.MustParseAddr("2001:4860:4860::8844"),
					netip.MustParseAddr("2001:4860:4860::8888"),
					netip.MustParseAddr("8.8.8.8"),
					netip.MustParseAddr("8.8.4.4"),
				},
				Err: nil,
			}
			close(resultCh)
			return resultCh
		},
	}
	dialer.DialStream(context.Background(), "dns.google:53")
	fmt.Println(ips)
	// Output:
	// [2001:4860:4860::8844 8.8.8.8 2001:4860:4860::8888 8.8.4.4]
}
