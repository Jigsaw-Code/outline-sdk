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
	done bool // indicates all splitted rcds have been already written to base
	frag FragFunc

	buf  *clientHelloBuffer // the buffer containing and parsing a TLS Client Hello record
	rcds *bytes.Buffer      // the buffer containing splitted records what will be written to base
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
func (w *clientHelloFragWriter) Write(p []byte) (n int, err error) {
	if w.done {
		return w.base.Write(p)
	}
	if w.rcds != nil {
		if _, err = w.flushRecords(); err != nil {
			return
		}
		return w.base.Write(p)
	}

	if n, err = w.buf.Write(p); err != nil {
		if errors.Is(err, errTLSClientHelloFullyReceived) {
			w.splitBufToRecords()
		} else {
			w.copyBufToRecords()
		}
		// recursively flush w.rcds and write the remaining content
		m, e := w.Write(p[n:])
		return n + m, e
	}

	if n < len(p) {
		return n, io.ErrShortWrite
	}
	return
}

// ReadFrom implements io.ReaderFrom.ReadFrom. It attempts to split the first packet into two TLS records if the data
// corresponds to a TLS Client Hello record. And then copies the remaining data from r to the base io.Writer until EOF
// or error.
//
// If the first packet is not a valid TLS Client Hello, everything from r gets copied to the base io.Writer as is.
//
// It returns the number of bytes read. Any error except EOF encountered during the read is also returned.
func (w *clientHelloFragReaderFrom) ReadFrom(r io.Reader) (n int64, err error) {
	if w.done {
		return w.baseRF.ReadFrom(r)
	}
	if w.rcds != nil {
		if _, err = w.flushRecords(); err != nil {
			return
		}
		return w.baseRF.ReadFrom(r)
	}

	if n, err = w.buf.ReadFrom(r); err != nil {
		if errors.Is(err, errTLSClientHelloFullyReceived) {
			w.splitBufToRecords()
		} else {
			w.copyBufToRecords()
		}
		// recursively flush w.rcds and read the remaining content from r
		m, e := w.ReadFrom(r)
		return n + m, e
	}
	return
}

// copyBuf copies w.buf into w.rcds.
func (w *clientHelloFragWriter) copyBufToRecords() {
	w.rcds = bytes.NewBuffer(w.buf.Bytes())
	w.buf = nil // allows the GC to recycle the memory
}

// splitBuf splits w.buf into two records and put them into w.rcds.
func (w *clientHelloFragWriter) splitBufToRecords() {
	content := w.buf.Bytes()[recordHeaderLen:]
	split := w.frag(content)
	if split <= 0 || split >= len(content) {
		w.copyBufToRecords()
		return
	}

	header := make([]byte, recordHeaderLen)
	w.rcds = bytes.NewBuffer(make([]byte, 0, w.buf.Len()+recordHeaderLen))

	putTLSClientHelloHeader(header, uint16(split))
	w.rcds.Write(header)
	w.rcds.Write(content[:split])

	putTLSClientHelloHeader(header, uint16(len(content)-split))
	w.rcds.Write(header)
	w.rcds.Write(content[split:])

	w.buf = nil // allows the GC to recycle the memory
}

// flushRecords writes all bytes from w.rcds to base.
func (w *clientHelloFragWriter) flushRecords() (int, error) {
	n, err := io.Copy(w.base, w.rcds)
	if w.rcds.Len() == 0 {
		w.rcds = nil // allows the GC to recycle the memory
		w.done = true
	}
	return int(n), err
}
