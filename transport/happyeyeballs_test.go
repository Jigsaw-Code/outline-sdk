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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHappyEyeballsStreamDialer_DialStream(t *testing.T) {
	t.Run("Works with IPv4 hosts", func(t *testing.T) {
		var dialedAddr string
		dialer := HappyEyeballsStreamDialer{
			Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
				dialedAddr = addr
				return nil, nil
			}),
		}
		_, err := dialer.DialStream(context.Background(), "8.8.8.8:53")
		require.NoError(t, err)
		require.Equal(t, "8.8.8.8:53", dialedAddr)
	})

	t.Run("Works with IPv6 hosts", func(t *testing.T) {
		var dialedAddr string
		dialer := HappyEyeballsStreamDialer{
			Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
				dialedAddr = addr
				return nil, nil
			}),
		}
		_, err := dialer.DialStream(context.Background(), "[2001:4860:4860::8888]:53")
		require.NoError(t, err)
		require.Equal(t, "[2001:4860:4860::8888]:53", dialedAddr)
	})

	t.Run("Prefer IPv6", func(t *testing.T) {
		var dialedAddr string
		dialer := HappyEyeballsStreamDialer{
			Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
				dialedAddr = addr
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
		require.Equal(t, "[2001:4860:4860::8888]:53", dialedAddr)
	})

	t.Run("Prefer IPv6 if there's a small delay", func(t *testing.T) {
		var dialedAddr string
		dialer := HappyEyeballsStreamDialer{
			Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
				dialedAddr = addr
				return nil, nil
			}),
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
		require.Equal(t, "[2001:4860:4860::8888]:53", dialedAddr)
	})

	t.Run("Use IPv4 if IPv6 hangs", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var dialedAddr string
		dialer := HappyEyeballsStreamDialer{
			Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
				dialedAddr = addr
				return nil, nil
			}),
			LookupIPv6: func(ctx context.Context, host string) ([]netip.Addr, error) {
				// Make it hang.
				<-ctx.Done()
				return []netip.Addr{netip.MustParseAddr("2001:4860:4860::8888")}, nil
			},
			LookupIPv4: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
			},
		}
		_, err := dialer.DialStream(ctx, "dns.google:53")
		require.NoError(t, err)
		require.Equal(t, "8.8.8.8:53", dialedAddr)
	})

	t.Run("Use IPv6 if IPv4 fails", func(t *testing.T) {
		var dialedAddr string
		dialer := HappyEyeballsStreamDialer{
			Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
				dialedAddr = addr
				return nil, nil
			}),
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
		require.Equal(t, "[2001:4860:4860::8888]:53", dialedAddr)
	})

	t.Run("Use IPv4 if IPv6 fails", func(t *testing.T) {
		var dialedAddr string
		dialer := HappyEyeballsStreamDialer{
			Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
				dialedAddr = addr
				return nil, nil
			}),
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
		require.Equal(t, "8.8.8.8:53", dialedAddr)
	})

	t.Run("No dial if lookup fails", func(t *testing.T) {
		var dialedAddr string
		dialer := HappyEyeballsStreamDialer{
			Dialer: FuncStreamDialer(func(ctx context.Context, addr string) (StreamConn, error) {
				dialedAddr = addr
				return nil, nil
			}),
			LookupIPv6: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return nil, errors.New("lookup failed")
			},
			LookupIPv4: func(ctx context.Context, host string) ([]netip.Addr, error) {
				return nil, errors.New("lookup failed")
			},
		}
		_, err := dialer.DialStream(context.Background(), "dns.google:53")
		require.Error(t, err)
		require.Empty(t, dialedAddr)
	})
}
