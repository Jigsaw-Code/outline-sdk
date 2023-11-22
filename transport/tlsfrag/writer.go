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
	done bool // Indicates all splitted rcds have been already written to base
	frag FragFunc

	helloBuf *clientHelloBuffer // The buffer containing and parsing a TLS Client Hello record
	record   *bytes.Buffer      // The buffer containing splitted records what will be written to base
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
		base:     base,
		frag:     frag,
		helloBuf: newClientHelloBuffer(),
	}
	if rf, ok := base.(io.ReaderFrom); ok {
		return &clientHelloFragReaderFrom{fw, rf}, nil
	}
	return fw, nil
}

// Write implements io.Writer.Write. It attempts to split the data received in the first one or more Write call(s)
// into two TLS records if the data corresponds to a TLS Client Hello record.
func (w *clientHelloFragWriter) Write(p []byte) (n int, err error) {
	if !w.done {
		// not yet splitted, append to the buffer
		if w.record == nil {
			if n, err = w.helloBuf.Write(p); err == nil {
				// all written, but Client Hello is not fully received yet
				return
			}
			p = p[n:]
			if errors.Is(err, errTLSClientHelloFullyReceived) {
				w.splitHelloBufToRecord()
			} else {
				w.copyHelloBufToRecord()
			}
		}
		// already splitted (but previous Writes might fail), try to flush all remaining w.record to w.base
		if _, err = w.flushRecord(); err != nil {
			return
		}
	}

	if len(p) > 0 {
		m, e := w.base.Write(p)
		n += m
		err = e
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
	if !w.done {
		// not yet splitted, append to the buffer
		if w.record == nil {
			if n, err = w.helloBuf.ReadFrom(r); err == nil {
				// EOF, but Client Hello is not fully received yet
				return
			}
			if errors.Is(err, errTLSClientHelloFullyReceived) {
				w.splitHelloBufToRecord()
			} else {
				w.copyHelloBufToRecord()
			}
		}
		// already splitted (but previous Writes might fail), try to flush all remaining w.record to w.base
		if _, err = w.flushRecord(); err != nil {
			return
		}
	}

	m, e := w.baseRF.ReadFrom(r)
	n += m
	err = e
	return
}

// copyHelloBufToRecord copies w.helloBuf into w.record without allocations.
func (w *clientHelloFragWriter) copyHelloBufToRecord() {
	w.record = bytes.NewBuffer(w.helloBuf.Bytes())
	w.helloBuf = nil // allows the GC to recycle the memory
}

// splitHelloBufToRecord splits w.helloBuf into two records and put them into w.record without allocations.
func (w *clientHelloFragWriter) splitHelloBufToRecord() {
	originalRecord := w.helloBuf.Bytes()
	content := received[recordHeaderLen:]
	headLen := w.frag(content)
	if split <= 0 || split >= len(content) {
		w.copyHelloBufToRecord()
		return
	}

	// received: | <== header (5) ==> | <== split ==> | <== len(content)-split ==> |  ... cap with padding (5) ... |
	//                                                 \                            \
	//                                                  +-----------------+          +-----------------+
	//                                                                     \                            \
	// splitted: | <== header (5) ==> | <== split ==> | <== header2 (5) ==> | <== len(content)-split ==> |
	splitted := received[:len(received)+recordHeaderLen]
	hdr1 := tlsRecordHeaderFromRawBytes(splitted[:recordHeaderLen])
	hdr2 := tlsRecordHeaderFromRawBytes(splitted[recordHeaderLen+split : recordHeaderLen*2+split])
        // Shift tail fragment to make space for record header.
	recvContent2 := splitted[recordHeaderLen+split : len(received)]
	content2 := splitted[recordHeaderLen*2+split:]
	copy(content2, recvContent2)
        // Insert header for second fragment.
	copy(hdr2, hdr1)
	hdr2.SetPayloadLen(uint16(len(content) - split))
	w.record = bytes.NewBuffer(splitted)
	w.helloBuf = nil // allows the GC to recycle the memory
}

// flushRecord writes all bytes from w.record to base.
func (w *clientHelloFragWriter) flushRecord() (int, error) {
	n, err := io.Copy(w.base, w.record)
	if w.record.Len() == 0 {
		w.record = nil // allows the GC to recycle the memory
		w.done = true
	}
	return int(n), err
}
