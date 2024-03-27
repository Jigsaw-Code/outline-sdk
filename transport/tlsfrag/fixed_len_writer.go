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

type fixedLenWriter struct {
	base io.Writer
	frag FixedLenFragFunc
	done bool
	hdr  []byte // the 5 bytes header

	// the records' sizes and written bytes (including 5 bytes header)
	r1Size, r1Written, r2Size int
}

type fixedLenReaderFrom struct {
	*fixedLenWriter
	baseRF io.ReaderFrom
}

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

// ReadFrom implements io.ReaderFrom.ReadFrom. It attempts to split the first packet into two TLS records if the data
// corresponds to a TLS Client Hello record without using any additional buffers.
// And then copies the remaining data from r to the base io.Writer until EOF or error.
//
// If the first packet is not a valid TLS Client Hello, everything from r gets copied to the base io.Writer as is.
//
// It returns the number of bytes read. Any error except EOF encountered during the read is also returned.
func (w *fixedLenReaderFrom) ReadFrom(r io.Reader) (n int64, err error) {
	if !w.done {
		if len(w.hdr) < recordHeaderLen {
			m, e := w.readRecordSizeFrom(r)
			n = int64(m)
			if err = e; err == io.EOF || err == io.ErrUnexpectedEOF {
				return n, nil
			} else if err != nil {
				w.done = true
			} else {
				h, _ := newTLSHandshakeRecordHeader(w.hdr)
				h.SetPayloadLen(uint16(w.r1Size - recordHeaderLen))
			}
		}

		if !w.done && w.r1Written < w.r1Size {
			var m int64
			if w.r1Written < recordHeaderLen {
				var hn int64
				hn, m, err = readFromBothN(w.baseRF, bytes.NewBuffer(w.hdr[w.r1Written:]), r, w.r1Size-w.r1Written)
				w.r1Written += int(hn + m)
			} else {
				m, err = w.baseRF.ReadFrom(io.LimitReader(r, int64(w.r1Size-w.r1Written)))
				w.r1Written += int(m)
			}
			n += m

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
		hn, pn, e := readFromBoth(w.baseRF, bytes.NewBuffer(w.hdr), r)
		w.hdr = w.hdr[hn:]
		return n + pn, e
	} else {
		m, e := w.baseRF.ReadFrom(r)
		return n + m, e
	}
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

// readFromBoth lets dst to read bytes from both r1 and r2.
func readFromBoth(dst io.ReaderFrom, r1 io.Reader, r2 io.Reader) (r1n int64, r2n int64, err error) {
	r1n, err = dst.ReadFrom(r1)
	if err == nil {
		r2n, err = dst.ReadFrom(r2)
	}
	return
}

// readFromBothN lets dst to read at most limit bytes from both r1 and r2.
func readFromBothN(dst io.ReaderFrom, r1 io.Reader, r2 io.Reader, limit int) (r1n int64, r2n int64, err error) {
	r1n, err = dst.ReadFrom(io.LimitReader(r1, int64(limit)))
	if err == nil && r1n < int64(limit) {
		r2n, err = dst.ReadFrom(io.LimitReader(r2, int64(limit)-r1n))
	}
	return
}
