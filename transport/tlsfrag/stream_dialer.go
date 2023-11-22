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

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

// tlsFragDialer is a [transport.StreamDialer] that uses clientHelloFragWriter to fragment the first Client Hello
// record in a TLS session.
type tlsFragDialer struct {
	dialer transport.StreamDialer
	frag   FragFunc
}

// Compilation guard against interface implementation
var _ transport.StreamDialer = (*tlsFragDialer)(nil)

// FragFunc takes the content of the first [handshake record] in a TLS session as input, and returns an integer that
// represents the fragmentation point index. The input content excludes the 5-byte record header. The returned integer
// should be in range 0 to len(record)-1. The record will then be fragmented into two parts: record[:n] and record[n:].
// If the returned index is either ≤ 0 or ≥ len(record), no fragmentation will occur.
//
// [handshake record]: https://datatracker.ietf.org/doc/html/rfc8446#section-5.1
type FragFunc func(record []byte) (n int)

// NewStreamDialerFunc creates a [transport.StreamDialer] that intercepts the initial [TLS Client Hello]
// [handshake record] and splits it into two separate records before sending them. The split point is determined by the
// callback function frag. The dialer then adds appropriate headers to each record and transmits them sequentially
// using the base dialer. Following the fragmented Client Hello, all subsequent data is passed through directly without
// modification.
//
// [TLS Client Hello]: https://datatracker.ietf.org/doc/html/rfc8446#section-4.1.2
// [handshake record]: https://datatracker.ietf.org/doc/html/rfc8446#section-5.1
func NewStreamDialerFunc(base transport.StreamDialer, frag FragFunc) (transport.StreamDialer, error) {
	if base == nil {
		return nil, errors.New("base dialer must not be nil")
	}
	if frag == nil {
		return nil, errors.New("frag function must not be nil")
	}
	return &tlsFragDialer{base, frag}, nil
}

// Dial implements [transport.StreamConn].Dial. It establishes a connection to raddr in the format "host-or-ip:port".
// The initial TLS Client Hello record sent through the connection will be fragmented.
func (d *tlsFragDialer) Dial(ctx context.Context, raddr string) (conn transport.StreamConn, err error) {
	conn, err = d.dialer.Dial(ctx, raddr)
	if err != nil {
		return
	}
	return WrapConnFunc(conn, d.frag)
}

// WrapConnFunc wraps the base [transport.StreamConn] and splits the first TLS Client Hello packet into two records
// according to the frag function. Subsequent data is forwarded without modification. The Write to the base
// [transport.StreamConn] will be buffered until we have the full initial Client Hello record. If the first packet
// isn't a valid Client Hello, WrapConnFunc simply forwards all data through transparently.
func WrapConnFunc(base transport.StreamConn, frag FragFunc) (transport.StreamConn, error) {
	w, err := newClientHelloFragWriter(base, frag)
	if err != nil {
		return nil, err
	}
	return transport.WrapConn(base, base, w), nil
}
