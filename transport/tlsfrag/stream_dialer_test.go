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
	"io"
	"net"
	"testing"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/require"
)

// Make sure only the first Client Hello is splitted in half.
func TestStreamDialerFuncSplitsClientHello(t *testing.T) {
	hello := constructTLSRecord(t, layers.TLSHandshake, 0x0301, []byte{0x01, 0x00, 0x00, 0x03, 0xaa, 0xbb, 0xcc})
	cipher := constructTLSRecord(t, layers.TLSChangeCipherSpec, 0x0303, []byte{0x01})
	req1 := constructTLSRecord(t, layers.TLSApplicationData, 0x0303, []byte{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88})

	inner := &collectStreamDialer{}
	conn := assertCanDialFragFunc(t, inner, "ipinfo.io:443", func(payload []byte) int { return len(payload) / 2 })
	defer conn.Close()

	assertCanWriteAll(t, conn, net.Buffers{hello, cipher, req1, hello, cipher, req1})

	frag1 := constructTLSRecord(t, layers.TLSHandshake, 0x0301, []byte{0x01, 0x00, 0x00})
	frag2 := constructTLSRecord(t, layers.TLSHandshake, 0x0301, []byte{0x03, 0xaa, 0xbb, 0xcc})
	expected := net.Buffers{
		append(frag1, frag2...),           // First two fragments will be merged in one single Write
		cipher, req1, hello, cipher, req1, // Unchanged
	}
	require.Equal(t, expected, inner.bufs)
}

// Make sure we don't split if the first packet is not a Client Hello.
func TestStreamDialerFuncDontSplitNonClientHello(t *testing.T) {
	cases := []struct {
		msg string
		pkt []byte
	}{
		{
			msg: "application data",
			pkt: constructTLSRecord(t, layers.TLSApplicationData, 0x0303, []byte{0x01, 0x00, 0x00, 0x03, 0xdd, 0xee, 0xff}),
		},
		{
			msg: "cipher",
			pkt: constructTLSRecord(t, layers.TLSChangeCipherSpec, 0x0303, []byte{0xff}),
		},
		{
			msg: "invalid version",
			pkt: constructTLSRecord(t, layers.TLSHandshake, 0x0305, []byte{0x01, 0x00, 0x00, 0x03, 0xdd, 0xee, 0xff}),
		},
		{
			msg: "invalid length",
			pkt: constructTLSRecord(t, layers.TLSHandshake, 0x0305, []byte{}),
		},
	}

	cipher := constructTLSRecord(t, layers.TLSChangeCipherSpec, 0x0303, []byte{0x01})
	req := constructTLSRecord(t, layers.TLSApplicationData, 0x0303, []byte{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88})

	for _, tc := range cases {
		inner := &collectStreamDialer{}
		conn := assertCanDialFragFunc(t, inner, "ipinfo.io:443", func(payload []byte) int { return len(payload) / 2 })
		defer conn.Close()

		assertCanWriteAll(t, conn, net.Buffers{tc.pkt, cipher, req})
		expected := net.Buffers{tc.pkt, cipher, req}
		if len(tc.pkt) > 5 {
			// Header and content of the first pkt might be issued by two Writes, but they are not fragmented
			expected = net.Buffers{tc.pkt[:5], tc.pkt[5:], cipher, req}
		}
		require.Equal(t, expected, inner.bufs, tc.msg)
	}
}

// Make sure only the first Client Hello is splitted by a fixed length.
func TestFixedLenStreamDialerSplitsClientHello(t *testing.T) {
	hello := constructTLSRecord(t, layers.TLSHandshake, 0x0301, []byte{0x01, 0x00, 0x00, 0x03, 0xaa, 0xbb, 0xcc})
	cipher := constructTLSRecord(t, layers.TLSChangeCipherSpec, 0x0303, []byte{0x01})
	req1 := constructTLSRecord(t, layers.TLSApplicationData, 0x0303, []byte{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88})

	cases := []struct {
		msg                string
		original, splitted net.Buffers
		splitLen           int
	}{
		{
			msg:      "split leading bytes",
			original: net.Buffers{hello, cipher, req1, hello, cipher, req1},
			splitLen: 2,
			splitted: net.Buffers{
				// Fragmented record header and payload are written as two packets by FixedLenWriter
				constructTLSRecord(t, layers.TLSHandshake, 0x0301, []byte{0x01, 0x00}),
				constructTLSRecord(t, layers.TLSHandshake, 0x0301, []byte{0x00, 0x03, 0xaa, 0xbb, 0xcc}),
				cipher, req1, hello, cipher, req1,
			},
		},
		{
			msg:      "split trailing bytes",
			original: net.Buffers{hello, cipher, req1, hello, cipher, req1},
			splitLen: -2,
			splitted: net.Buffers{
				// Fragmented record header and payload are written as two packets by FixedLenWriter
				constructTLSRecord(t, layers.TLSHandshake, 0x0301, []byte{0x01, 0x00, 0x00, 0x03, 0xaa}),
				constructTLSRecord(t, layers.TLSHandshake, 0x0301, []byte{0xbb, 0xcc}),
				cipher, req1, hello, cipher, req1,
			},
		},
		{
			msg:      "no split",
			original: net.Buffers{hello, cipher, req1, hello, cipher, req1},
			splitLen: 0,
			splitted: net.Buffers{hello, cipher, req1, hello, cipher, req1},
		},
	}

	for _, tc := range cases {
		inner := &collectStreamDialer{}
		conn := assertCanDialFixedLenFrag(t, inner, "ipinfo.io:443", tc.splitLen)
		defer conn.Close()

		assertCanWriteAll(t, conn, tc.original)
		require.Equal(t, tc.splitted, inner.bufs, tc.msg)
	}
}

// Make sure the first Client Hello can be splitted multiple times.
func TestNestedFixedLenStreamDialerSplitsClientHello(t *testing.T) {
	hello := constructTLSRecord(t, layers.TLSHandshake, 0x0301, []byte{
		0x01, 0x00, 0x00, 0x03, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11,
	})

	// Fragmented record header and payload are written as two packets by FixedLenWriter
	frag1 := constructTLSRecord(t, layers.TLSHandshake, 0x0301, []byte{0x01, 0x00, 0x00})
	frag2 := constructTLSRecord(t, layers.TLSHandshake, 0x0301, []byte{0x03, 0xaa, 0xbb, 0xcc, 0xdd})
	frag3 := constructTLSRecord(t, layers.TLSHandshake, 0x0301, []byte{0xee, 0xff, 0x99, 0x88, 0x77, 0x66, 0x55, 0x44})
	frag4 := constructTLSRecord(t, layers.TLSHandshake, 0x0301, []byte{0x33, 0x22, 0x11})

	cipher := constructTLSRecord(t, layers.TLSChangeCipherSpec, 0x0303, []byte{0x01})
	req1 := constructTLSRecord(t, layers.TLSApplicationData, 0x0303, []byte{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88})

	inner := &collectStreamDialer{}
	d, err := NewFixedLenStreamDialer(inner, 3) // Further split msg[:8] mentioned below into msg[:3] + msg[3:8]
	require.NoError(t, err)
	d, err = NewFixedLenStreamDialer(d, 8) // Further split msg[:16] mentioned below into msg[:8] + msg[8:16]
	require.NoError(t, err)
	conn := assertCanDialFixedLenFrag(t, d, "ipinfo.io:443", -3) // Split msg[:19] into msg[:16] + msg[16:19]
	defer conn.Close()

	assertCanWriteAll(t, conn, net.Buffers{hello, cipher, req1, hello, cipher, req1})

	expected := net.Buffers{
		frag1, frag2, frag3, frag4,
		cipher, req1, hello, cipher, req1, // Unchanged
	}
	require.Equal(t, expected, inner.bufs)
}

// test assertions

func assertCanDialFragFunc(t *testing.T, inner transport.StreamDialer, raddr string, frag FragFunc) transport.StreamConn {
	d, err := NewStreamDialerFunc(inner, frag)
	require.NoError(t, err)
	require.NotNil(t, d)
	conn, err := d.DialStream(context.Background(), raddr)
	require.NoError(t, err)
	require.NotNil(t, conn)
	return conn
}

func assertCanDialFixedLenFrag(t *testing.T, inner transport.StreamDialer, raddr string, splitLen int) transport.StreamConn {
	d, err := NewFixedLenStreamDialer(inner, splitLen)
	require.NoError(t, err)
	require.NotNil(t, d)
	conn, err := d.DialStream(context.Background(), raddr)
	require.NoError(t, err)
	require.NotNil(t, conn)
	return conn
}

func assertCanWriteAll(t *testing.T, w io.Writer, buf net.Buffers) {
	for _, p := range buf {
		n, err := w.Write(p)
		require.NoError(t, err)
		require.Equal(t, len(p), n)
	}
}

// Private test helpers

func constructTLSRecord(t *testing.T, typ layers.TLSType, ver layers.TLSVersion, payload []byte) []byte {
	pkt := layers.TLS{
		AppData: []layers.TLSAppDataRecord{{
			TLSRecordHeader: layers.TLSRecordHeader{
				ContentType: typ,
				Version:     ver,
				Length:      uint16(len(payload)),
			},
			Payload: payload,
		}},
	}

	buf := gopacket.NewSerializeBuffer()
	err := pkt.SerializeTo(buf, gopacket.SerializeOptions{})
	require.NoError(t, err)
	return buf.Bytes()
}

// collectStreamDialer collects all writes to this stream dialer and append it to bufs
type collectStreamDialer struct {
	bufs net.Buffers
}

func (d *collectStreamDialer) DialStream(ctx context.Context, raddr string) (transport.StreamConn, error) {
	return d, nil
}

func (c *collectStreamDialer) Write(p []byte) (int, error) {
	c.bufs = append(c.bufs, append([]byte{}, p...)) // Copy p rather than retaining it according to the principle of Write
	return len(p), nil
}

func (c *collectStreamDialer) Read(p []byte) (int, error)         { return 0, errors.New("not supported") }
func (c *collectStreamDialer) Close() error                       { return nil }
func (c *collectStreamDialer) CloseRead() error                   { return nil }
func (c *collectStreamDialer) CloseWrite() error                  { return nil }
func (c *collectStreamDialer) LocalAddr() net.Addr                { return nil }
func (c *collectStreamDialer) RemoteAddr() net.Addr               { return nil }
func (c *collectStreamDialer) SetDeadline(t time.Time) error      { return errors.New("not supported") }
func (c *collectStreamDialer) SetReadDeadline(t time.Time) error  { return errors.New("not supported") }
func (c *collectStreamDialer) SetWriteDeadline(t time.Time) error { return errors.New("not supported") }
