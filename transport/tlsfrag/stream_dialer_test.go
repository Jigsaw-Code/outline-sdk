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
	"strconv"
	"testing"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/require"
)

// this is the local conn that can be shared across tests
var theLocalConn = &localConn{}

// Make sure NewStreamDialer returns error on invalid WithTLSHostPortList calls.
func TestNewStreamDialerWithInvalidTLSAddr(t *testing.T) {
	cases := []struct {
		addr    string
		errType error // nil indicates general error
	}{
		{"1.2.3.4", nil},
		{":::::", nil},
		{"1.2.3.4:654-321", strconv.ErrSyntax},
		{"1.2.3.4:--8080", strconv.ErrSyntax},
		{"[::]:10000000000", strconv.ErrRange},
		{"1.2.3.4:-1234", strconv.ErrRange},
		{":654321", strconv.ErrRange},
	}
	for _, tc := range cases {
		d, err := NewStreamDialerFunc(localConnDialer{}, func([]byte) int { return 0 }, WithTLSHostPortList([]string{tc.addr}))
		require.Error(t, err, tc.addr)
		if tc.errType != nil {
			require.ErrorIs(t, err, tc.errType, tc.addr)
		}
		require.Nil(t, d)
	}
}

// Make sure no fragmentation connection is created if raddr is not in the allowed list.
func TestDialFragmentOnTLSAddrOnly(t *testing.T) {
	tlsAddrs := []string{
		":443",              // default entry
		":990",              // additional FTPS port
		":853",              // additional DNS-over-TLS port
		"pop.gmail.com:995", // Gmail pop3
	}
	cases := []struct {
		msg                string
		raddrs             []string
		shouldFrag         bool
		shouldFragWithList bool
	}{
		{
			msg:                "*:443 should be fragmented, raddr = %s",
			raddrs:             []string{"example.com:443", "66.77.88.99:443", "[2001:db8::1]:443"},
			shouldFrag:         true,
			shouldFragWithList: true,
		},
		{
			msg:                "*:990 should be fragmented by allowlist, raddr = %s",
			raddrs:             []string{"my-test.org:990", "192.168.1.10:990", "[2001:db8:3333:4444:5555:6666:7777:8888]:990"},
			shouldFrag:         false,
			shouldFragWithList: true,
		},
		{
			msg:                "*:8080 should not be fragmented, raddr = %s",
			raddrs:             []string{"google.com:8080", "64.233.191.255:8080", "[2001:db8:3333:4444:5555:6666:7777:8888]:8080"},
			shouldFrag:         false,
			shouldFragWithList: false,
		},
		{
			msg:                "DNS ports should not be fragmented, raddr = %s",
			raddrs:             []string{"8.8.8.8:53", "8.8.4.4:53", "2001:4860:4860::8888", "2001:4860:4860::8844"},
			shouldFrag:         false,
			shouldFragWithList: false,
		},
		{
			msg:                "DNS over TLS ports should be fragmented by allowlist, raddr = %s",
			raddrs:             []string{"9.9.9.9:853", "8.8.4.4:853", "[2001:4860:4860::8844]:853", "[2620:fe::fe]:853"},
			shouldFrag:         false,
			shouldFragWithList: true,
		},
		{
			msg:                "only gmail POP3 should be fragmented by allowlist, raddr = %s",
			raddrs:             []string{"pop.GMail.com:995"},
			shouldFrag:         false,
			shouldFragWithList: true,
		},
		{
			msg:                "non-gmail POP3 should not be fragmented, raddr = %s",
			raddrs:             []string{"8.8.8.8:995", "outlook.office365.com:995", "outlook.office365.com:993", "pop.gmail.com:993"},
			shouldFrag:         false,
			shouldFragWithList: false,
		},
	}

	base := localConnDialer{}
	assertShouldFrag := func(conn transport.StreamConn, msg, addr string) {
		prevWrCnt := theLocalConn.writeCount
		// this Write should not be pushed to theLocalConn yet because it's a valid TLS handshake
		conn.Write([]byte{22})

		nonFragConn, ok := conn.(*localConn)
		require.False(t, ok, msg, addr)
		require.Nil(t, nonFragConn, msg)
		require.Equal(t, prevWrCnt, theLocalConn.writeCount, msg, addr)
	}
	assertShouldNotFrag := func(conn transport.StreamConn, msg, addr string) {
		prevWrCnt := theLocalConn.writeCount
		// this Write should be pushed to theLocalConn because it's a direct Write call
		conn.Write([]byte{22})

		nonFragConn, ok := conn.(*localConn)
		require.True(t, ok, msg, addr)
		require.NotNil(t, nonFragConn, msg, addr)
		require.Equal(t, theLocalConn, nonFragConn)
		require.Equal(t, prevWrCnt+1, theLocalConn.writeCount, msg, addr)
	}

	// default dialer
	d1, err := NewStreamDialerFunc(base, func([]byte) int { return 0 })
	require.NoError(t, err)
	require.NotNil(t, d1)

	// with additional tls addrs
	d2, err := NewStreamDialerFunc(base, func([]byte) int { return 0 }, WithTLSHostPortList(tlsAddrs))
	require.NoError(t, err)
	require.NotNil(t, d2)

	// with no tls addrs
	d3, err := NewStreamDialerFunc(base, func([]byte) int { return 0 }, WithTLSHostPortList([]string{}))
	require.NoError(t, err)
	require.NotNil(t, d3)

	// all traffic
	d4, err := NewStreamDialerFunc(base, func([]byte) int { return 0 }, WithTLSHostPortList([]string{":0"}))
	require.NoError(t, err)
	require.NotNil(t, d4)

	for _, tc := range cases {
		for _, addr := range tc.raddrs {
			conn, err := d1.Dial(context.Background(), addr)
			require.NoError(t, err, tc.msg, addr)
			require.NotNil(t, conn, tc.msg, addr)
			if tc.shouldFrag {
				assertShouldFrag(conn, tc.msg, addr)
			} else {
				assertShouldNotFrag(conn, tc.msg, addr)
			}

			conn, err = d2.Dial(context.Background(), addr)
			require.NoError(t, err, tc.msg, addr)
			require.NotNil(t, conn, tc.msg, addr)
			if tc.shouldFragWithList {
				assertShouldFrag(conn, tc.msg, addr)
			} else {
				assertShouldNotFrag(conn, tc.msg, addr)
			}

			conn, err = d3.Dial(context.Background(), addr)
			require.NoError(t, err, tc.msg, addr)
			require.NotNil(t, conn, tc.msg, addr)
			assertShouldNotFrag(conn, tc.msg, addr)

			conn, err = d4.Dial(context.Background(), addr)
			require.NoError(t, err, tc.msg, addr)
			require.NotNil(t, conn, tc.msg, addr)
			assertShouldFrag(conn, tc.msg, addr)
		}
	}
}

// testing utilitites

type localConnDialer struct{}
type localConn struct {
	transport.StreamConn
	writeCount int
}

func (localConnDialer) Dial(ctx context.Context, raddr string) (transport.StreamConn, error) {
	return theLocalConn, nil
}

func (lc *localConn) Write(b []byte) (n int, err error) {
	lc.writeCount++
	return len(b), nil
}
