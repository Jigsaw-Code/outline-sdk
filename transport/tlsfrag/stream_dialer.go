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
// should be in range 1 to len(record)-1. The record will then be fragmented into two parts: record[:n] and record[n:].
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
// If you just want to split the record at a fixed position (e.g., always at the 5th byte or 2nd from the last byte),
// use [NewFixedLenStreamDialer]. It consumes less resources and is more efficient.
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

// DialStream implements [transport.StreamConn].DialStream. It establishes a connection to raddr in the format "host-or-ip:port".
// The initial TLS Client Hello record sent through the connection will be fragmented.
func (d *tlsFragDialer) DialStream(ctx context.Context, raddr string) (transport.StreamConn, error) {
	baseConn, err := d.dialer.DialStream(ctx, raddr)
	if err != nil {
		return nil, err
	}
	conn, err := WrapConnFunc(baseConn, d.frag)
	if err != nil {
		baseConn.Close()
		return nil, err
	}
	return conn, nil
}

// WrapConnFunc wraps the base [transport.StreamConn] and splits the first TLS Client Hello packet into two records
// according to the frag function. After that, all subsequent data is forwarded without modification.
//
// If the first packet isn't a valid Client Hello, WrapConnFunc doesn't modify anything.
//
// The Write to the base [transport.StreamConn] will be buffered until we have the full initial Client Hello record.
// If your goal is to simply fragment the Client Hello at a fixed position, [WrapConnFixedLen] is more efficient as it
// won't allocate any additional buffers.
func WrapConnFunc(base transport.StreamConn, frag FragFunc) (transport.StreamConn, error) {
	w, err := newClientHelloFragWriter(base, frag)
	if err != nil {
		return nil, err
	}
	return transport.WrapConn(base, base, w), nil
}

// tlsFixedLenFragDialer is a [transport.StreamDialer] that will fragment the first Client Hello record at a fixed
// position within the record in a TLS session.
type tlsFixedLenFragDialer struct {
	dialer   transport.StreamDialer
	splitLen int
}

// NewFixedLenStreamDialer is a [transport.StreamDialer] that fragments the [TLS handshake record]. It splits the
// record into two records based on the given splitLen. If splitLen is positive, the first piece will contain the
// specified number of leading bytes from the original message. If it is negative, the second piece will contain
// the specified number of trailing bytes.
//
// [TLS handshake record]: https://datatracker.ietf.org/doc/html/rfc8446#section-5.1
func NewFixedLenStreamDialer(base transport.StreamDialer, splitLen int) (transport.StreamDialer, error) {
	if base == nil {
		return nil, errors.New("base dialer must not be nil")
	}
	if splitLen == 0 {
		return base, nil
	}
	return &tlsFixedLenFragDialer{base, splitLen}, nil
}

// DialStream implements [transport.StreamConn].DialStream. It establishes a connection to raddr in the format
// "host-or-ip:port".
// The initial TLS Client Hello record sent through the connection will be fragmented at a fixed position
// indicated by d.splitLen.
func (d *tlsFixedLenFragDialer) DialStream(ctx context.Context, raddr string) (transport.StreamConn, error) {
	baseConn, err := d.dialer.DialStream(ctx, raddr)
	if err != nil {
		return nil, err
	}
	conn, err := WrapConnFixedLen(baseConn, d.splitLen)
	if err != nil {
		baseConn.Close()
		return nil, err
	}
	return conn, nil
}

// WrapConnFixedLen wraps the base [transport.StreamConn] and splits the first TLS Client Hello record into two records
// at a fixed position indicated by splitLen. Subsequent data is forwarded without modification.
//
// If splitLen is positive, the first fragmented record content will have that length (excluding the 5-byte header);
// if splitLen is negative, the second fragmented record content will be that length.
//
// If the first message isn't a valid Client Hello, WrapConnFixedLen forwards all data without modifying it.
//
// This is more efficient than [WrapConnFunc] because it doesn't allocate additional buffers.
func WrapConnFixedLen(base transport.StreamConn, splitLen int) (conn transport.StreamConn, err error) {
	var w io.Writer
	if splitLen > 0 {
		w, err = NewFixedLenWriter(base, func(recordLen int) int { return splitLen })
	} else {
		w, err = NewFixedLenWriter(base, func(recordLen int) int { return recordLen + splitLen })
	}
	if err != nil {
		return
	}
	conn = transport.WrapConn(base, base, w)
	return
}
