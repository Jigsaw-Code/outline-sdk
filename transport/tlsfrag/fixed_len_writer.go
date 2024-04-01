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
	"errors"
	"io"
)

// RecordLenFragFunc takes the length of the first [handshake record]'s content (without the 5-byte header),
// and returns an integer that determines where the record should be fragmented.
//
// The returned splitLen should be in range 1 to recordLen-1.
// The record content will then be fragmented into two parts: record[:splitLen] and record[splitLen:].
// If splitLen is either ≤ 0 or ≥ recordLen, no fragmentation will occur.
//
// [handshake record]: https://datatracker.ietf.org/doc/html/rfc8446#section-5.1
type RecordLenFragFunc func(recordLen int) (splitLen int)

// recordLenFragWriter splits the initial TLS Client Hello record into two TLS records based on a fixed length
// returned by a [RecordLenFragFunc] callback.
// These fragmented records are then written to the base [io.Writer]. Subsequent packets are not modified and are
// directly transmitted through the base [io.Writer].
type recordLenFragWriter struct {
	base   io.Writer
	frag   RecordLenFragFunc
	done   bool                     // the first fragmented header and payload are flushed (or don't split)
	hdr    []byte                   // the raw 5 bytes header, use tlsHdr to update PayloadLen
	tlsHdr tlsHandshakeRecordHeader // non-nil if hdr is a valid TLS Handshake record header

	// the records' sizes and written bytes (including 5 bytes header)
	r1Size, r1Written, r2Size int
}

var _ io.Writer = (*recordLenFragWriter)(nil)

// NewRecordLenFuncWriter creates a [io.Writer] that splits the first TLS Client Hello record into two records
// based on the provided [RecordLenFragFunc] callback.
// It then writes these records and all subsequent messages to the base [io.Writer].
// If the first message isn't a Client Hello, no splitting occurs and all messages are written directly to base.
//
// The returned [io.Writer] will implement the [io.ReaderFrom] interface for optimized performance if the base
// [io.Writer] implements [io.ReaderFrom].
func NewRecordLenFuncWriter(base io.Writer, frag RecordLenFragFunc) (io.Writer, error) {
	if base == nil {
		return nil, errors.New("base writer must not be nil")
	}
	if frag == nil {
		return nil, errors.New("fixed length frag callback function must not be nil")
	}
	wr := &recordLenFragWriter{
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
func (w *recordLenFragWriter) Write(p []byte) (n int, err error) {
	if !w.done {
		if w.tlsHdr == nil {
			// try to fill r.w.hdr of totally 5 bytes
			hdrLen := len(w.hdr)
			n = copy(w.hdr[hdrLen:recordHeaderLen], p)
			p = p[n:]
			if w.hdr = w.hdr[:hdrLen+n]; len(w.hdr) < recordHeaderLen {
				return
			}

			// construct the structured TLS Handshake header object
			w.tlsHdr, err = newTLSHandshakeRecordHeader(w.hdr)
			if err != nil || w.tlsHdr.Validate() != nil || w.updateSplitRecordLens() != nil {
				// invalid TLS header, or invalid split lens, stop splitting
				w.done = true
			} else {
				// update header to be the first record's header
				w.tlsHdr.SetPayloadLen(uint16(w.r1Size - recordHeaderLen))
			}
		}

		if !w.done && w.r1Written < w.r1Size {
			var m int
			if w.r1Written < recordHeaderLen {
				// write the first record's header
				m, err = w.base.Write(w.tlsHdr[w.r1Written:recordHeaderLen])
				if w.r1Written += m; w.r1Written < recordHeaderLen || err != nil || len(p) == 0 {
					return
				}
			}

			// write the first record's remaining payload
			m, err = writeN(w.base, p, w.r1Size-w.r1Written)
			w.r1Written += m
			n += m
			p = p[m:]

			if w.r1Written == w.r1Size {
				// update w.tlsHdr (aliases w.hdr) to the second record's header
				w.tlsHdr.SetPayloadLen(uint16(w.r2Size - recordHeaderLen))
				w.done = true
			}
			if w.r1Written < w.r1Size || err != nil {
				return
			}
		}
	}

	if len(w.hdr) > 0 {
		m, e := w.base.Write(w.hdr)
		w.hdr = w.hdr[m:]
		if err = e; err != nil || len(w.hdr) > 0 || len(p) == 0 {
			return
		}
	}
	m, e := w.base.Write(p)
	return n + m, e
}

// fixedLenReaderFrom optimizes for fixedLenWriter when the base [io.Writer] implements [io.ReaderFrom].
type fixedLenReaderFrom struct {
	*recordLenFragWriter
	baseRF io.ReaderFrom
}

var _ io.ReaderFrom = (*fixedLenReaderFrom)(nil)

// fixedLenFirstRecordReader reads 5 bytes from r into w.hdr (aliased by w.tlsHdr), and calculate the split length.
// It then copies up to (w.r1Size - w.r1Written) bytes from (w.tlsHdr & r) into Read's buffer.
// It will update w.r1Written and w.done accordingly.
// After the first record is fully copied, it will set w.tlsHdr (aliases w.hdr) to be the second record's header.
type fixedLenFirstRecordReader struct {
	w        *recordLenFragWriter
	r        io.Reader
	rReadLen int64
}

// fixedLenRemainingRecordReader flushes the content of w.hdr and all remaining r into Read's buffer.
type fixedLenRemainingRecordReader struct {
	w        *recordLenFragWriter
	r        io.Reader
	rReadLen int64
}

var _ io.Reader = (*fixedLenFirstRecordReader)(nil)
var _ io.Reader = (*fixedLenRemainingRecordReader)(nil)

func (r *fixedLenFirstRecordReader) Read(p []byte) (n int, err error) {
	if !r.w.done {
		if r.w.tlsHdr == nil {
			// try to fill r.w.hdr of totally 5 bytes
			hdrLen := len(r.w.hdr)
			m, e := io.ReadFull(r.r, r.w.hdr[hdrLen:recordHeaderLen])
			r.rReadLen += int64(m)
			if r.w.hdr = r.w.hdr[:hdrLen+m]; len(r.w.hdr) < recordHeaderLen {
				e = io.EOF
			}
			if err = e; err != nil {
				return
			}

			// construct the structured TLS Handshake header object
			r.w.tlsHdr, err = newTLSHandshakeRecordHeader(r.w.hdr)
			if err != nil || r.w.tlsHdr.Validate() != nil || r.w.updateSplitRecordLens() != nil {
				// invalid TLS header, or invalid split lens, stop splitting
				r.w.done = true
				err = io.EOF
				return
			}

			// update header to be the first record's header
			r.w.tlsHdr.SetPayloadLen(uint16(r.w.r1Size - recordHeaderLen))
		}

		if r.w.r1Written < r.w.r1Size {
			if r.w.r1Written < recordHeaderLen {
				n = copy(p, r.w.tlsHdr[r.w.r1Written:])
				r.w.r1Written += n
				if p = p[n:]; len(p) == 0 {
					return
				}
			}
			m, e := io.LimitReader(r.r, int64(r.w.r1Size-r.w.r1Written)).Read(p)
			n += m
			err = e
			r.rReadLen += int64(m)
			if r.w.r1Written += m; r.w.r1Written == r.w.r1Size {
				// update r.w.tlsHdr (aliases r.w.hdr) to the second record's header
				r.w.tlsHdr.SetPayloadLen(uint16(r.w.r2Size - recordHeaderLen))
				r.w.done = true
			}
		}
		return
	}
	return 0, io.EOF
}

func (r *fixedLenRemainingRecordReader) Read(p []byte) (n int, err error) {
	if !r.w.done {
		return 0, io.EOF
	}
	if len(r.w.hdr) > 0 {
		n = copy(p, r.w.hdr)
		p = p[n:]
		if r.w.hdr = r.w.hdr[n:]; len(r.w.hdr) > 0 {
			return
		}
	}
	m, e := r.r.Read(p)
	r.rReadLen += int64(m)
	return n + m, e
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
		r1Reader := &fixedLenFirstRecordReader{w: w.recordLenFragWriter, r: r}
		r2Reader := &fixedLenRemainingRecordReader{w: w.recordLenFragWriter, r: r}
		_, err = w.baseRF.ReadFrom(io.MultiReader(r1Reader, r2Reader))

		// We should return the actual bytes read from r, not the bytes passed to base or baseRF
		n = r1Reader.rReadLen + r2Reader.rReadLen
		return
	}
	return w.baseRF.ReadFrom(r)
}

// updateSplitLen determines the split length by calling w.frag with the input of w.tlsHdr.PayloadLen().
// It returns nil error if w.frag returns a valid split length, otherwise it returns non-nil error.
//
// The corresponding record lengths (including the 5 bytes header) will be set to w.r1Size and w.r2Size.
func (w *recordLenFragWriter) updateSplitRecordLens() error {
	if w.tlsHdr == nil {
		return errors.New("invalid TLS header")
	}
	totalPayloadLen := int(w.tlsHdr.PayloadLen())
	splitPayloadLen := w.frag(totalPayloadLen)
	if splitPayloadLen <= 0 || splitPayloadLen >= totalPayloadLen {
		return errors.New("invalid split position")
	}
	w.r1Size = splitPayloadLen + recordHeaderLen
	w.r2Size = totalPayloadLen - splitPayloadLen + recordHeaderLen
	return nil
}

// writeN writes at most limit bytes from p to dst.
func writeN(dst io.Writer, p []byte, limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	if len(p) > limit {
		p = p[:limit]
	}
	return dst.Write(p)
}
