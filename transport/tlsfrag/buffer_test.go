// Copyright 2023 The Outline Authors
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
	"bytes"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test Write valid Client Hello to the buffer.
func TestWriteValidClientHello(t *testing.T) {
	for _, tc := range validClientHelloTestCases() {
		buf := newClientHelloBuffer()

		totalExpectedBytes := []byte{}
		for k, pkt := range tc.pkts {
			n, err := buf.Write(pkt)
			if k < tc.expectLastPkt {
				require.NoError(t, err, tc.msg+": pkt-%d", k)
			} else {
				require.ErrorIs(t, err, errTLSClientHelloFullyReceived, tc.msg+": pkt-%d", k)
			}
			require.Equal(t, len(pkt)-len(tc.expectRemaining[k]), n, tc.msg+": pkt-%d", k)
			require.Equal(t, tc.expectRemaining[k], pkt[n:], tc.msg+": pkt-%d", k)

			totalExpectedBytes = append(totalExpectedBytes, pkt[:n]...)
			require.Equal(t, totalExpectedBytes, buf.Bytes(), tc.msg+": pkt-%d", k)
		}
		require.Equal(t, tc.expectTotalPkt, buf.Bytes(), tc.msg)
		require.Equal(t, len(tc.expectTotalPkt)+5, cap(buf.Bytes()), tc.msg)
	}
}

// Test ReadFrom Reader(s) containing valid Client Hello.
func TestReadFromValidClientHello(t *testing.T) {
	for _, tc := range validClientHelloTestCases() {
		buf := newClientHelloBuffer()

		totalExpectedBytes := []byte{}
		for k, pkt := range tc.pkts {
			r := bytes.NewBuffer(pkt)
			require.Equal(t, len(pkt), r.Len(), tc.msg+": pkt-%d", k)

			n, err := buf.ReadFrom(r)
			if k < tc.expectLastPkt {
				require.NoError(t, err, tc.msg+": pkt-%d", k)
			} else {
				require.ErrorIs(t, err, errTLSClientHelloFullyReceived, tc.msg+": pkt-%d", k)
			}
			require.Equal(t, len(pkt)-len(tc.expectRemaining[k]), int(n), tc.msg+": pkt-%d", k)
			require.Equal(t, tc.expectRemaining[k], pkt[n:], tc.msg+": pkt-%d", k)

			totalExpectedBytes = append(totalExpectedBytes, pkt[:n]...)
			require.Equal(t, totalExpectedBytes, buf.Bytes(), tc.msg+": pkt-%d", k)
			require.Equal(t, len(tc.expectRemaining[k]), r.Len(), tc.msg+": pkt-%d", k)
			require.Equal(t, tc.expectRemaining[k], r.Bytes(), tc.msg+": pkt-%d", k)
		}
		require.Equal(t, tc.expectTotalPkt, buf.Bytes(), tc.msg)
		require.Equal(t, len(tc.expectTotalPkt)+5, cap(buf.Bytes()), tc.msg)
	}
}

// Example TLS Client Hello packet copied from https://tls13.xargs.org/#client-hello
// Total Len = 253, ContentLen = 253 - 5 = 248
var exampleTLS13ClientHello = []byte{
	0x16, 0x03, 0x01, 0x00, 0xf8, 0x01, 0x00, 0x00, 0xf4, 0x03, 0x03, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c,
	0x1d, 0x1e, 0x1f, 0x20, 0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7, 0xe8, 0xe9, 0xea, 0xeb, 0xec, 0xed, 0xee, 0xef,
	0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7, 0xf8, 0xf9, 0xfa, 0xfb, 0xfc, 0xfd, 0xfe, 0xff, 0x00, 0x08, 0x13, 0x02,
	0x13, 0x03, 0x13, 0x01, 0x00, 0xff, 0x01, 0x00, 0x00, 0xa3, 0x00, 0x00, 0x00, 0x18, 0x00, 0x16, 0x00, 0x00, 0x13, 0x65,
	0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x75, 0x6c, 0x66, 0x68, 0x65, 0x69, 0x6d, 0x2e, 0x6e, 0x65, 0x74, 0x00, 0x0b,
	0x00, 0x04, 0x03, 0x00, 0x01, 0x02, 0x00, 0x0a, 0x00, 0x16, 0x00, 0x14, 0x00, 0x1d, 0x00, 0x17, 0x00, 0x1e, 0x00, 0x19,
	0x00, 0x18, 0x01, 0x00, 0x01, 0x01, 0x01, 0x02, 0x01, 0x03, 0x01, 0x04, 0x00, 0x23, 0x00, 0x00, 0x00, 0x16, 0x00, 0x00,
	0x00, 0x17, 0x00, 0x00, 0x00, 0x0d, 0x00, 0x1e, 0x00, 0x1c, 0x04, 0x03, 0x05, 0x03, 0x06, 0x03, 0x08, 0x07, 0x08, 0x08,
	0x08, 0x09, 0x08, 0x0a, 0x08, 0x0b, 0x08, 0x04, 0x08, 0x05, 0x08, 0x06, 0x04, 0x01, 0x05, 0x01, 0x06, 0x01, 0x00, 0x2b,
	0x00, 0x03, 0x02, 0x03, 0x04, 0x00, 0x2d, 0x00, 0x02, 0x01, 0x01, 0x00, 0x33, 0x00, 0x26, 0x00, 0x24, 0x00, 0x1d, 0x00,
	0x20, 0x35, 0x80, 0x72, 0xd6, 0x36, 0x58, 0x80, 0xd1, 0xae, 0xea, 0x32, 0x9a, 0xdf, 0x91, 0x21, 0x38, 0x38, 0x51, 0xed,
	0x21, 0xa2, 0x8e, 0x3b, 0x75, 0xe9, 0x65, 0xd0, 0xd2, 0xcd, 0x16, 0x62, 0x54,
}

type validClientHelloCase struct {
	msg             string
	pkts            net.Buffers
	expectTotalPkt  []byte
	expectLastPkt   int // the index of the expected last packet
	expectRemaining net.Buffers
}

// generating test cases that contain one valid Client Hello
func validClientHelloTestCases() []validClientHelloCase {
	return []validClientHelloCase{
		{
			msg:             "full client hello in single buffer",
			pkts:            [][]byte{exampleTLS13ClientHello},
			expectTotalPkt:  exampleTLS13ClientHello,
			expectLastPkt:   0,
			expectRemaining: [][]byte{{}},
		},
		{
			msg:             "full client hello with extra bytes",
			pkts:            [][]byte{append(exampleTLS13ClientHello, 0x88, 0x87, 0x86, 0x85, 0x84, 0x83, 0x82, 0x81)},
			expectTotalPkt:  exampleTLS13ClientHello,
			expectLastPkt:   0,
			expectRemaining: [][]byte{{0x88, 0x87, 0x86, 0x85, 0x84, 0x83, 0x82, 0x81}},
		},
		{
			msg:             "client hello in three buffers",
			pkts:            [][]byte{exampleTLS13ClientHello[:2], exampleTLS13ClientHello[2:123], exampleTLS13ClientHello[123:]},
			expectTotalPkt:  exampleTLS13ClientHello,
			expectLastPkt:   2,
			expectRemaining: [][]byte{{}, {}, {}},
		},
		{
			msg: "client hello in three buffers with extra bytes",
			pkts: [][]byte{
				exampleTLS13ClientHello[:2],
				exampleTLS13ClientHello[2:123],
				append(exampleTLS13ClientHello[123:], 0x88, 0x87, 0x86, 0x85, 0x84, 0x83, 0x82, 0x81),
			},
			expectTotalPkt:  exampleTLS13ClientHello,
			expectLastPkt:   2,
			expectRemaining: [][]byte{{}, {}, {0x88, 0x87, 0x86, 0x85, 0x84, 0x83, 0x82, 0x81}},
		},
	}
}
