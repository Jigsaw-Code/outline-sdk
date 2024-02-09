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
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type colletcStreamDialer struct {
	Dialer StreamDialer
	Addrs  []string
}

func (d *colletcStreamDialer) DialStream(ctx context.Context, addr string) (StreamConn, error) {
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
		baseDialer := colletcStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{Dialer: &baseDialer}
		_, err := dialer.DialStream(context.Background(), "8.8.8.8:53")
		require.NoError(t, err)
		require.Equal(t, []string{"8.8.8.8:53"}, baseDialer.Addrs)
	})

	t.Run("Works with IPv6 hosts", func(t *testing.T) {
		baseDialer := colletcStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{Dialer: &baseDialer}
		_, err := dialer.DialStream(context.Background(), "[2001:4860:4860::8888]:53")
		require.NoError(t, err)
		require.Equal(t, []string{"[2001:4860:4860::8888]:53"}, baseDialer.Addrs)
	})

	t.Run("Prefer IPv6", func(t *testing.T) {
		baseDialer := colletcStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			LookupIPv6: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil
			},
			LookupIPv4: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
			},
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.NoError(t, err)
		require.Equal(t, []string{"[2001:4860:4860::8888]:53"}, baseDialer.Addrs)
	})

	t.Run("Prefer IPv6 if there's a small delay", func(t *testing.T) {
		baseDialer := colletcStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			LookupIPv6: func(ctx context.Context, host string) ([]netip.Addr, error) {
				time.Sleep(10 * time.Millisecond)
				return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil
			},
			LookupIPv4: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
			},
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.NoError(t, err)
		require.Equal(t, []string{"[2001:4860:4860::8888]:53"}, baseDialer.Addrs)
	})

	t.Run("Use IPv4 if IPv6 hangs, with fallback", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		baseDialer := colletcStreamDialer{Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
			if addr == "8.8.8.8:53" {
				return nil, errors.New("failed to dial")
			}
			return nil, nil
		})}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			LookupIPv6: func(ctx context.Context, host string) ([]netip.Addr, error) {
				// Make it hang.
				<-ctx.Done()
				return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil
			},
			LookupIPv4: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("8.8.4.4")}, nil
			},
		}
		_, err := dialer.DialStream(ctx, "dns.google:53")
		require.NoError(t, err)
		require.Equal(t, []string{"8.8.8.8:53", "8.8.4.4:53"}, baseDialer.Addrs)
	})

	t.Run("Use IPv6 if IPv4 fails", func(t *testing.T) {
		baseDialer := colletcStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			LookupIPv6: func(ctx context.Context, host string) ([]netip.Addr, error) {
				time.Sleep(10 * time.Millisecond)
				return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil

			},
			LookupIPv4: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return nil, errors.New("lookup failed")
			},
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.NoError(t, err)
		require.Equal(t, []string{"[2001:4860:4860::8888]:53"}, baseDialer.Addrs)
	})

	t.Run("Use IPv4 if IPv6 fails", func(t *testing.T) {
		baseDialer := colletcStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			LookupIPv6: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return nil, errors.New("lookup failed")
			},
			LookupIPv4: func(ctx context.Context, host string) ([]netip.Addr, error) {
				time.Sleep(10 * time.Millisecond)
				return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
			},
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.NoError(t, err)
		require.Equal(t, []string{"8.8.8.8:53"}, baseDialer.Addrs)
	})

	t.Run("No dial if lookup fails", func(t *testing.T) {
		baseDialer := colletcStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			LookupIPv6: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return nil, errors.New("lookup failed")
			},
			LookupIPv4: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return nil, errors.New("lookup failed")
			},
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
			LookupIPv6: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil
			},
			LookupIPv4: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
			},
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.NoError(t, err)
		require.Equal(t, []string{"[2001:4860:4860::8888]:53", "8.8.8.8:53"}, dialedAddrs)
	})

	t.Run("IP order", func(t *testing.T) {
		dialFailErr := errors.New("failed to dial")
		baseDialer := colletcStreamDialer{Dialer: newErrorStreamDialer(dialFailErr)}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			LookupIPv6: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return []netip.Addr{
					netip.MustParseAddr("::1"),
					netip.MustParseAddr("::2"),
					netip.MustParseAddr("::3"),
				}, nil
			},
			LookupIPv4: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return []netip.Addr{
					netip.MustParseAddr("1.1.1.1"),
					netip.MustParseAddr("2.2.2.2"),
					netip.MustParseAddr("3.3.3.3"),
				}, nil
			},
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.ErrorIs(t, err, dialFailErr)
		require.Equal(t, []string{"[::1]:53", "1.1.1.1:53", "[::2]:53", "2.2.2.2:53", "[::3]:53", "3.3.3.3:53"}, baseDialer.Addrs)
	})

	t.Run("Cancelled lookups", func(t *testing.T) {
		var hold sync.WaitGroup
		hold.Add(1)
		defer hold.Done()
		baseDialer := colletcStreamDialer{Dialer: nilDialer}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			LookupIPv6: func(ctx context.Context, host string) ([]netip.Addr, error) {
				hold.Wait()
				return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil
			},
			LookupIPv4: func(ctx context.Context, host string) ([]netip.Addr, error) {
				hold.Wait()
				return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
			},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := dialer.DialStream(ctx, "dns.google:53")
		require.ErrorIs(t, err, context.Canceled)
		require.Empty(t, baseDialer.Addrs)
	})

	t.Run("Cancelled dial", func(t *testing.T) {
		var hold sync.WaitGroup
		hold.Add(1)
		defer hold.Done()
		ctx, cancel := context.WithCancel(context.Background())
		baseDialer := colletcStreamDialer{Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
			cancel()
			hold.Wait()
			return nil, nil
		})}
		dialer := HappyEyeballsStreamDialer{
			Dialer: &baseDialer,
			LookupIPv6: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil
			},
			LookupIPv4: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
			},
		}
		_, err := dialer.DialStream(ctx, "dns.google:53")
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, []string{"[2001:4860:4860::8888]:53"}, baseDialer.Addrs)
	})
}
