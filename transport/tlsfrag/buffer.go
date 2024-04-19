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
	"bytes"
	"errors"
	"io"
)

var (
	// errTLSClientHelloFullyReceived is returned when a full TLS Client Hello has been received and no
	// more data can be pushed to the buffer.
	errTLSClientHelloFullyReceived = errors.New("already received a complete TLS Client Hello packet")
)

// clientHelloBuffer is a byte buffer used to receive and buffer a TLS Client Hello packet.
type clientHelloBuffer struct {
	// The buffer that hosts both header and content, cap: 5 -> 5+len(content)+padding
	data []byte
	// Indicates whether the content in data is a valid TLS Client Hello record
	validationErr error
	// A reader used to read from the slice passed to Write
	bufrd *bytes.Reader
}

var _ io.Writer = (*clientHelloBuffer)(nil)
var _ io.ReaderFrom = (*clientHelloBuffer)(nil)

// newClientHelloBuffer creates and initializes a new buffer to receive a TLS Client Hello packet.
func newClientHelloBuffer() *clientHelloBuffer {
	// Allocate the 5 bytes header first, and then reallocate it to contain the entire packet later
	return &clientHelloBuffer{
		data:          make([]byte, 0, recordHeaderLen),
		validationErr: nil,
		bufrd:         bytes.NewReader(nil), // It will be Reset in Write
	}
}

// Bytes returns the full Client Hello packet including both the 5 bytes header and the content.
func (b *clientHelloBuffer) Bytes() []byte {
	return b.data
}

// Write appends p to the buffer and returns the number of bytes actually used.
// If this data completes a valid TLS Client Hello, it returns errTLSClientHelloFullyReceived.
// If an invalid TLS Client Hello message is detected, it returns the error errInvalidTLSClientHello.
// If all bytes in p have been used and the buffer still requires more data to build a complete TLS Client Hello
// message, it returns (len(p), nil).
func (b *clientHelloBuffer) Write(p []byte) (int, error) {
	b.bufrd.Reset(p)
	n, err := b.ReadFrom(b.bufrd)
	if err == nil && int(n) != len(p) {
		err = io.ErrShortWrite
	}
	return int(n), err
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
//
// ReadFrom will hang indefinitely if r provides fewer than 5 bytes and doesn't return the io.EOF error (e.g., "PING").
func (b *clientHelloBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	// Waiting to finish the header of 5 bytes
	if len(b.data) < recordHeaderLen {
		m, e := io.ReadFull(r, b.data[len(b.data):recordHeaderLen])
		b.data = b.data[:len(b.data)+m]
		n += int64(m)
		if err = e; err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				err = nil
			}
			return
		}

		hdr, e := newTLSHandshakeRecordHeader(b.data)
		if err = e; err != nil {
			b.validationErr = err
			return
		}
		if err = hdr.Validate(); err != nil {
			b.validationErr = err
			return
		}
		buf := make([]byte, 0, recordHeaderLen*2+hdr.PayloadLen())
		b.data = append(buf, b.data...)
	}

	// If the buffer is already invalid
	if b.validationErr != nil {
		err = b.validationErr
		return
	}

	// Waiting to finish the payload of cap(b.data)-5 bytes
	for len(b.data) < cap(b.data)-recordHeaderLen {
		m, e := r.Read(b.data[len(b.data) : cap(b.data)-recordHeaderLen])
		b.data = b.data[:len(b.data)+m]
		n += int64(m)

		if len(b.data) == cap(b.data)-recordHeaderLen {
			err = errTLSClientHelloFullyReceived
			return
		}
		if err = e; err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}
	}

	err = errTLSClientHelloFullyReceived
	return
}
