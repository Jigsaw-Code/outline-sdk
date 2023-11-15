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
	"errors"
	"fmt"
	"io"
)

var (
	// errTLSClientHelloFullyReceived is returned when a full TLS Client Hello has been received and no
	// more data can be pushed to the buffer.
	errTLSClientHelloFullyReceived = errors.New("already received a complete TLS Client Hello packet")

	// errInvalidTLSClientHello is the error used when the data received is not a valid TLS Client Hello.
	// Please use [errors.Is] to compare the returned err object with this instance.
	errInvalidTLSClientHello = errors.New("not a valid TLS Client Hello packet")
)

// clientHelloBuffer is a byte buffer used to receive and buffer a TLS Client Hello packet.
type clientHelloBuffer struct {
	data   []byte // the buffer that hosts both header and content, cap: 5 -> 5+len(content)
	len    int    // the number of bytes that have already been read into data
	valid  bool   // indicate whether the content in data is a valid TLS Client Hello record
	toRead int    // the number of bytes to read next, e.g. 1 -> 2 -> 2 -> len(content)
}

var _ io.Writer = (*clientHelloBuffer)(nil)
var _ io.ReaderFrom = (*clientHelloBuffer)(nil)

// newClientHelloBuffer creates and initializes a new buffer to receive a TLS Client Hello packet.
func newClientHelloBuffer() *clientHelloBuffer {
	// Allocate the 5 bytes header first, and then reallocate it to contain the entire packet later
	return &clientHelloBuffer{
		data:   make([]byte, recordHeaderLen),
		len:    0,
		valid:  true,
		toRead: tlsRecordWithTypeSize,
	}
}

// Len returns the length of this buffer including both the 5 bytes header and the content.
func (b *clientHelloBuffer) Len() int {
	return b.len
}

// Bytes returns the full Client Hello packet including both the 5 bytes header and the content.
func (b *clientHelloBuffer) Bytes() []byte {
	return b.data[:b.len]
}

func (b *clientHelloBuffer) growBy(size int) {
	buf := make([]byte, b.len+size)
	copy(buf, b.data[:b.len])
	b.data = buf
}

// Write appends p to the buffer and returns the number of bytes actually used.
// If this data completes a valid TLS Client Hello, it returns errTLSClientHelloFullyReceived.
// If an invalid TLS Client Hello message is detected, it returns the error errInvalidTLSClientHello.
// If all bytes in p have been used and the buffer still requires more data to build a complete TLS Client Hello
// message, it returns (len(p), nil).
func (b *clientHelloBuffer) Write(p []byte) (n int, err error) {
	if !b.valid {
		return 0, errInvalidTLSClientHello
	}

	defer func() { b.valid = !errors.Is(err, errInvalidTLSClientHello) }()

	for b.toRead > 0 {
		if len(p) == 0 {
			return
		}

		sz := b.toRead
		if len(p) < sz {
			sz = len(p)
		}
		copy(b.data[b.len:], p[:sz])
		n += sz
		b.len += sz
		b.toRead -= sz
		p = p[sz:]

		// check whether message is valid according to the bytes just read
		switch b.len {
		case tlsRecordWithTypeSize: // 1
			if typ := getRecordType(b.data); typ != recordTypeHandshake {
				return n, fmt.Errorf("record type %d is not handshake: %w", typ, errInvalidTLSClientHello)
			}
			b.toRead = tlsRecordWithVersionHeaderSize - tlsRecordWithTypeSize // +2

		case tlsRecordWithVersionHeaderSize: // 3
			if ver := getTLSVersion(b.data); !isValidTLSVersion(ver) {
				return n, fmt.Errorf("%#04x is not a valid TLS version: %w", ver, errInvalidTLSClientHello)
			}
			b.toRead = recordHeaderLen - tlsRecordWithVersionHeaderSize // +2

		case recordHeaderLen: // 5
			if b.toRead = int(getMsgLen(b.data)); !isValidMsgLenForHandshake(uint16(b.toRead)) {
				return n, fmt.Errorf("message length %v out of range: %w", b.toRead, errInvalidTLSClientHello)
			}
			b.growBy(b.toRead)
		}
	}

	return n, errTLSClientHelloFullyReceived
}

// ReadFrom reads all the data from r and appends it to this buffer until a complete Client Hello packet has been
// received, or r returns EOF or error. It returns the number of bytes read. Any error except EOF encountered during
// the read is also returned.
//
// If this buffer completes a valid TLS Client Hello, it returns errTLSClientHelloFullyReceived.
// If an invalid TLS Client Hello message is detected, it returns the error errInvalidTLSClientHello.
// If this buffer still requires more data to build a complete TLS Client Hello message, it returns nil error.
//
// You can call ReadFrom multiple times if r doesn't provide enough data to build a complete Client Hello packet.
func (b *clientHelloBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	return 0, errors.New("to be implemented")
}
