// Copyright 2024 Jigsaw Operations LLC
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
	"net"
)

// FixedLenFragFunc takes the length of the first [handshake record]'s content (without the 5-byte header),
// and returns an integer that determines where the record should be fragmented.
//
// The returned splitLen should be in range 1 to recordLen-1.
// The record content will then be fragmented into two parts: record[:splitLen] and record[splitLen:].
// If splitLen is either ≤ 0 or ≥ recordLen, no fragmentation will occur.
//
// [handshake record]: https://datatracker.ietf.org/doc/html/rfc8446#section-5.1
type FixedLenFragFunc func(recordLen int) (splitLen int)

// fixedLenWriter splits the initial TLS Client Hello record into two TLS records based on a fixed length returned by
// a [FixedLenFragFunc] callback.
// These fragmented records are then written to the base [io.Writer]. Subsequent packets are not modified and are
// directly transmitted through the base [io.Writer].
type fixedLenWriter struct {
	base io.Writer
	frag FixedLenFragFunc
	done bool   // the first fragmented header and payload are flushed (or don't split)
	hdr  []byte // the 5 bytes header

	// the records' sizes and written bytes (including 5 bytes header)
	r1Size, r1Written, r2Size int
}

var _ io.Writer = (*fixedLenWriter)(nil)

// NewFixedLenWriter creates a [io.Writer] that splits the first TLS Client Hello record into two records based
// on the provided [FixedLenFragFunc] callback.
// It then writes these records and all subsequent messages to the base [io.Writer].
// If the first message isn't a Client Hello, no splitting occurs and all messages are written directly to base.
//
// The returned [io.Writer] will implement the [io.ReaderFrom] interface for optimized performance if the base
// [io.Writer] implements [io.ReaderFrom].
func NewFixedLenWriter(base io.Writer, frag FixedLenFragFunc) (io.Writer, error) {
	if base == nil {
		return nil, errors.New("base writer must not be nil")
	}
	if frag == nil {
		return nil, errors.New("fixed length frag callback function must not be nil")
	}
	wr := &fixedLenWriter{
		base: base,
		frag: frag,
		hdr:  make([]byte, 0, recordHeaderLen),
	}
	if rf, ok := base.(io.ReaderFrom); ok {
		return &fixedLenReaderFrom{wr, rf}, nil
	}
	return wr, nil
}

// Write implements io.Writer.Write. It attempts to split the data received in the first one or more Write call(s)
// into two TLS records if the data corresponds to a TLS Client Hello record without using any additional buffers.
func (w *fixedLenWriter) Write(p []byte) (n int, err error) {
	if !w.done {
		if len(w.hdr) < recordHeaderLen {
			// read the 5-bytes header and calculate the split position
			r := bytes.NewBuffer(p)
			if n, err = w.readRecordSizeFrom(r); err == io.EOF || err == io.ErrUnexpectedEOF {
				return n, nil
			} else if err != nil {
				w.done = true
			} else {
				// update header to the first record's header
				h, _ := newTLSHandshakeRecordHeader(w.hdr)
				h.SetPayloadLen(uint16(w.r1Size - recordHeaderLen))
			}
			p = r.Bytes()
		}

		if !w.done && w.r1Written < w.r1Size {
			var m int
			if w.r1Written < recordHeaderLen {
				// write the first record's header and payload
				var hn int
				hn, m, err = writeBothN(w.base, w.hdr[w.r1Written:], p, w.r1Size-w.r1Written)
				w.r1Written += hn + m
			} else {
				// write the first record's remaining payload
				m, err = writeN(w.base, p, w.r1Size-w.r1Written)
				w.r1Written += m
			}
			n += m
			p = p[m:]

			if w.r1Written == w.r1Size {
				// update header to the second record's header
				h, _ := newTLSHandshakeRecordHeader(w.hdr)
				h.SetPayloadLen(uint16(w.r2Size - recordHeaderLen))
				w.done = true
			}
			if err != nil || w.r1Written < w.r1Size {
				return
			}
		}
	}

	if len(w.hdr) > 0 {
		hn, pn, e := writeBoth(w.base, w.hdr, p)
		w.hdr = w.hdr[hn:]
		return n + pn, e
	} else {
		m, e := w.base.Write(p)
		return n + m, e
	}
}

// fixedLenReaderFrom optimizes for fixedLenWriter when the base [io.Writer] implements [io.ReaderFrom].
type fixedLenReaderFrom struct {
	*fixedLenWriter
	baseRF io.ReaderFrom
}

var _ io.ReaderFrom = (*fixedLenReaderFrom)(nil)

// fixedLenFirstRecordReader reads 5 bytes from r into w.hdr, and calculate the split length.
// It then copies up to (w.r1Size - w.r1Written) bytes from (w.hdr + r) into Read's buffer.
// It will update w.r1Written and w.done accordingly.
// After the first record is fully copied, it will set w.hdr to be the second fragmented record header.
type fixedLenFirstRecordReader struct {
	w        *fixedLenWriter
	r        io.Reader
	rReadLen int64
}

// fixedLenRemainingRecordReader flushes the content of w.hdr and all remaining r into Read's buffer.
type fixedLenRemainingRecordReader struct {
	w        *fixedLenWriter
	r        io.Reader
	rReadLen int64
}

var _ io.Reader = (*fixedLenFirstRecordReader)(nil)
var _ io.Reader = (*fixedLenRemainingRecordReader)(nil)

func (r *fixedLenFirstRecordReader) Read(p []byte) (n int, err error) {
	if !r.w.done {
		if len(r.w.hdr) < recordHeaderLen {
			// still constructing the header
			m, e := r.w.readRecordSizeFrom(r.r)
			r.rReadLen += int64(m)
			if e != nil {
				// r.w.done = invalid TLS = (e != EOF)
				r.w.done = (e != io.EOF && e != io.ErrUnexpectedEOF)
				return 0, io.EOF
			}
			// update hdr to the first record's header
			h, _ := newTLSHandshakeRecordHeader(r.w.hdr)
			h.SetPayloadLen(uint16(r.w.r1Size - recordHeaderLen))
		}

		if r.w.r1Written < r.w.r1Size {
			if r.w.r1Written < recordHeaderLen {
				n = copy(p, r.w.hdr[r.w.r1Written:])
				r.w.r1Written += n
				if p = p[n:]; len(p) == 0 {
					return
				}
			}
			var m int
			m, err = io.LimitReader(r.r, int64(r.w.r1Size-r.w.r1Written)).Read(p)
			n += m
			r.rReadLen += int64(m)
			if r.w.r1Written += m; r.w.r1Written == r.w.r1Size {
				// update header to the second record's header
				h, _ := newTLSHandshakeRecordHeader(r.w.hdr)
				h.SetPayloadLen(uint16(r.w.r2Size - recordHeaderLen))
				r.w.done = true
			}
		}
		if err == nil && r.w.r1Written == r.w.r1Size {
			err = io.EOF
		}
		return
	}
	return 0, io.EOF
}

func (r *fixedLenRemainingRecordReader) Read(p []byte) (n int, err error) {
	if r.w.done {
		if len(r.w.hdr) > 0 {
			n = copy(p, r.w.hdr)
			r.w.hdr = r.w.hdr[n:]
			if p = p[n:]; len(p) == 0 {
				return
			}
		}
		m, e := r.r.Read(p)
		r.rReadLen += int64(m)
		return n + m, e
	}
	return 0, io.EOF
}

// ReadFrom implements io.ReaderFrom.ReadFrom. It attempts to split the first packet into two TLS records if the data
// corresponds to a TLS Client Hello record without using any additional buffers.
// And then copies the remaining data from r to the base io.Writer until EOF or error.
//
// If the first packet is not a valid TLS Client Hello, everything from r gets copied to the base io.Writer as is.
//
// It returns the number of bytes read. Any error except EOF encountered during the read is also returned.
func (w *fixedLenReaderFrom) ReadFrom(r io.Reader) (n int64, err error) {
	if !w.done || len(w.hdr) > 0 {
		r1Reader := &fixedLenFirstRecordReader{w: w.fixedLenWriter, r: r}
		r2Reader := &fixedLenRemainingRecordReader{w: w.fixedLenWriter, r: r}
		_, err = w.baseRF.ReadFrom(io.MultiReader(r1Reader, r2Reader))
		// We should return the actual bytes read from r, not the bytes passed to base or baseRF
		n = r1Reader.rReadLen + r2Reader.rReadLen
		return
	}
	return w.baseRF.ReadFrom(r)
}

// readRecordSizeFrom reads the 5 bytes header from r and updates the fragmented record sizes stored in w.
// It returns io.EOF or io.ErrUnexpectedEOF if there are not enough bytes to construct a full header.
func (w *fixedLenWriter) readRecordSizeFrom(r io.Reader) (n int, err error) {
	n, err = io.ReadFull(r, w.hdr[len(w.hdr):recordHeaderLen])
	w.hdr = w.hdr[:len(w.hdr)+n]
	if err != nil {
		return
	}

	h, err := newTLSHandshakeRecordHeader(w.hdr)
	if err != nil {
		return
	}
	if err = h.Validate(); err != nil {
		return
	}

	totalPayloadLen := int(h.GetPayloadLen())
	splitPayloadLen := w.frag(totalPayloadLen)
	if splitPayloadLen <= 0 || splitPayloadLen >= totalPayloadLen {
		err = errors.New("invalid split position")
		return
	}

	w.r1Size = splitPayloadLen + recordHeaderLen
	w.r2Size = totalPayloadLen - splitPayloadLen + recordHeaderLen
	return
}

// writeN writes at most limit bytes from p to dst.
func writeN(dst io.Writer, p []byte, limit int) (int, error) {
	if len(p) > limit {
		p = p[:limit]
	}
	return dst.Write(p)
}

// writeBoth writes both p1 and p2 to dst.
// It leverages writev system call when possible.
func writeBoth(dst io.Writer, p1 []byte, p2 []byte) (p1n int, p2n int, err error) {
	var buf io.WriterTo
	if len(p2) == 0 {
		buf = bytes.NewBuffer(p1)
	} else if len(p1) == 0 {
		buf = bytes.NewBuffer(p2)
	} else {
		buf = &net.Buffers{p1, p2}
	}
	n, e := buf.WriteTo(dst)
	if p1n = int(n); p1n > len(p1) {
		p1n = len(p1)
		p2n = int(n) - p1n
	}
	return p1n, p2n, e
}

// writeBothN writes at most limit bytes from p1 and p2 to dst.
// It leverages writev system call when possible.
func writeBothN(dst io.Writer, p1 []byte, p2 []byte, limit int) (p1n int, p2n int, err error) {
	if len(p1)+len(p2) > limit {
		p2 = p2[:limit-len(p1)]
	}
	return writeBoth(dst, p1, p2)
}
