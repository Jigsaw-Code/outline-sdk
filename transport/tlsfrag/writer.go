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

// clientHelloFragWriter intercepts the initial TLS Client Hello record and splits it into two TLS records based on the
// return value of frag function. These fragmented records are then written to the base [io.Writer]. Subsequent packets
// are not modified and are directly transmitted through the base [io.Writer].
type clientHelloFragWriter struct {
	base io.Writer
	done bool
	frag FragFunc
	buf  *clientHelloBuffer
}

// clientHelloFragReaderFrom serves as an optimized version of clientHelloFragWriter when the base [io.Writer] also
// implements the [io.ReaderFrom] interface.
type clientHelloFragReaderFrom struct {
	*clientHelloFragWriter
	baseRF io.ReaderFrom
}

// Compilation guard against interface implementation
var _ io.Writer = (*clientHelloFragWriter)(nil)
var _ io.Writer = (*clientHelloFragReaderFrom)(nil)
var _ io.ReaderFrom = (*clientHelloFragReaderFrom)(nil)

// newClientHelloFragWriter creates a [io.Writer] that splits the first TLS Client Hello record into two records based
// on the provided frag function. It then writes these records and all subsequent messages to the base [io.Writer].
// If the first message isn't a Client Hello, no splitting occurs and all messages are written directly to base.
//
// The returned [io.Writer] will implement the [io.ReaderFrom] interface for optimized performance if the base
// [io.Writer] implements [io.ReaderFrom].
func newClientHelloFragWriter(base io.Writer, frag FragFunc) (io.Writer, error) {
	if base == nil {
		return nil, errors.New("base writer must not be nil")
	}
	if frag == nil {
		return nil, errors.New("frag callback function must not be nil")
	}
	fw := &clientHelloFragWriter{
		base: base,
		frag: frag,
		buf:  newClientHelloBuffer(),
	}
	if rf, ok := base.(io.ReaderFrom); ok {
		return &clientHelloFragReaderFrom{fw, rf}, nil
	}
	return fw, nil
}

// Write implements io.Writer.Write. It attempts to split the data received in the first one or more Write call(s)
// into two TLS records if the data corresponds to a TLS Client Hello record.
//
// Internally, this function maintains a state machine with the following states:
//   - S: reading the first client hello record and appending the data to w.buf
//   - F: the first client hello record has been read, fragmenting and writing to w.base
//   - T: forwarding all remaining packets without modification
//
// Here is the transition graph:
//
//	S ----(full handshake read)----> F -----> T
//	|                                         ^
//	|                                         |
//	+-----(invalid TLS handshake)-------------+
func (w *clientHelloFragWriter) Write(p []byte) (written int, err error) {
	// T: optimize to have fewer comparisons for the most common case.
	if w.done {
		return w.base.Write(p)
	}

	// S
	nr, e := w.buf.ReadFrom(bytes.NewBuffer(p))

	// S -> T
	if errors.Is(e, errInvalidTLSClientHello) {
		goto FlushBufAndDone
	}

	// S < x < F, wait for the next write
	if e != nil || !w.buf.HasFullyReceived() {
		return int(nr), e
	}

	// F
	if err = w.buf.Split(w.frag(w.buf.Content())); err != nil {
		return int(nr), err
	}

	// * -> T (err must be nil)
FlushBufAndDone:
	w.done = true
	nw, e := w.buf.WriteTo(w.base)
	written += w.buf.BytesOverlapped(nr, nw)
	w.buf = nil // allows the GC to recycle the memory

	// If WriteTo failed, no need to write more data
	if err = e; err != nil {
		return
	}

	m, e := w.base.Write(p[nr:])
	written += m
	err = e
	return
}

// ReadFrom implements io.ReaderFrom.ReadFrom. It attempts to split the first packet into two TLS records if the data
// corresponds to a TLS Client Hello record. And then copies the remaining data from r to the base io.Writer until EOF
// or error.
//
// If the first packet is not a valid TLS Client Hello, everything from r gets copied to the base io.Writer as is.
//
// It returns the number of bytes read. Any error except EOF encountered during the read is also returned.
//
// Internally, it uses a similar state machine to the one mentioned in w.Write. But the transition is simplier because
// we expect r containing all the data (while the first packet might be consisted of multiple Writes in Write).
func (w *clientHelloFragReaderFrom) ReadFrom(r io.Reader) (n int64, err error) {
	// T
	if w.done {
		return w.baseRF.ReadFrom(r)
	}

	// S & F
	nr, e := w.buf.ReadFrom(r)
	if err = e; err == nil && w.buf.HasFullyReceived() {
		err = w.buf.Split(w.frag(w.buf.Content()))
	} else if errors.Is(err, errInvalidTLSClientHello) {
		err = nil
	}

	// * -> T (err might be non-nil, but we still need to flush data to w.base)
	w.done = true
	nw, e := w.buf.WriteTo(w.base)
	n += int64(w.buf.BytesOverlapped(nr, nw))
	w.buf = nil // allows the GC to recycle the memory

	// If WriteTo failed, no need to write more data
	if err = errors.Join(err, e); e != nil {
		return
	}

	m, e := w.baseRF.ReadFrom(r)
	n += m
	err = e
	return
}
