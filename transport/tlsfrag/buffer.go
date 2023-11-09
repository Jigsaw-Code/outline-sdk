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
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// TLS record layout from [RFC 8446]:
//
//	+-------------+ 0
//	| ContentType |
//	+-------------+ 1
//	|  Protocol   |
//	|  Version    |
//	+-------------+ 3
//	|   Record    |
//	|   Length    |
//	+-------------+ 5
//	|    Data     |
//	|     ...     |
//	+-------------+ Record Length + 5
//
//	ContentType := invalid(0) | handshake(22) | application_data(23) | ...
//	Protocol Version (deprecated) := 0x0301 ("TLS 1.0") | 0x0303 ("TLS 1.2" & "TLS 1.3") | 0x0302 ("TLS 1.1")
//	0 < Record Length (of handshake)        ≤ 2^14
//	0 ≤ Record Length (of application_data) ≤ 2^14
//
// [RFC 8446]: https://datatracker.ietf.org/doc/html/rfc8446#section-5.1
const (
	tlsRecordWithTypeSize          = 1 // the minimum size that contains record type
	tlsRecordWithVersionHeaderSize = 3 // the minimum size that contains protocol version
	tlsRecordHeaderSize            = 5 // the minimum size that contains the entire header
	tlsTypeHandshake               = 22
	tlsMaxRecordLen                = 1 << 14
)

// errInvalidTLSClientHello is the error used when the data received is not a valid TLS Client Hello.
// Please use [errors.Is] to compare the returned err object with this instance.
var errInvalidTLSClientHello = errors.New("not a valid TLS Client Hello packet")

func isTLSRecordTypeHandshake(hdr []byte) bool {
	return hdr[0] == tlsTypeHandshake
}

// isValidTLSProtocolVersion determines whether hdr[1:3] is a valid TLS version according to RFC:
//
//	"""
//	legacy_record_version:
//	MUST be set to 0x0303 for all records generated by a TLS 1.3 implementation other than an initial ClientHello,
//	where it MAY also be 0x0301 for compatibility purposes. This field is deprecated and MUST be ignored for all
//	purposes. Previous versions of TLS would use other values in this field under some circumstances.
//	"""
func isValidTLSProtocolVersion(hdr []byte) bool {
	return hdr[1] == 0x03 && (0x01 <= hdr[2] && hdr[2] <= 0x03)
}

func recordLen(hdr []byte) uint16 {
	return binary.BigEndian.Uint16(hdr[3:])
}

func isValidRecordLenForHandshake(len uint16) bool {
	return 0 < len && len <= tlsMaxRecordLen
}

func putTLSClientHelloHeader(hdr []byte, recordLen uint16) {
	_ = hdr[4] // bounds check to guarantee safety of writes below
	hdr[0] = tlsTypeHandshake
	hdr[1] = 0x03
	hdr[2] = 0x01
	binary.BigEndian.PutUint16(hdr[3:], recordLen)
}

// clientHelloBuffer is a byte buffer used to receive and send the TLS Client Hello packet.
// This packet can be splitted into two records if needed.
type clientHelloBuffer struct {
	data      []byte // the buffer that hosts both header and content, len(data) should be either 5 or recordLen+10
	valid     bool   // indicate whether the content in data is a valid TLS Client Hello record
	len       int    // the number of the bytes that has been read into data
	recordLen int    // the length of the original (unsplitted) record content (without header)
	split     int    // the 0-based index to split the packet into [:split] and [split:]
}

// newClientHelloBuffer creates and initializes a new buffer to receive TLS Client Hello packet.
func newClientHelloBuffer() *clientHelloBuffer {
	// Allocate the 5 bytes header first, and reallocate it to contain the entire packet later
	return &clientHelloBuffer{
		data:  make([]byte, tlsRecordHeaderSize),
		valid: true,
	}
}

// ReadFrom reads all the data from r and appends it to this buffer until a complete Client Hello packet has been
// received, or r returns EOF or error. It returns the number of bytes read. Any error except EOF encountered during
// the read is also returned.
//
// You can call ReadFrom multiple times if r doesn't provide enough data to build a complete Client Hello packet.
// Call HasFullyReceived to check whether a complete Client Hello packet has been constructed.
func (b *clientHelloBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	if !b.valid {
		return 0, errInvalidTLSClientHello
	}

	if b.len < tlsRecordHeaderSize {
		m, e := b.readHeaderFrom(r)
		n += int64(m)
		if err = e; err == io.EOF {
			return n, nil
		}
		if err != nil {
			return
		}
	}

	if b.len < b.recordLen+tlsRecordHeaderSize {
		m, e := b.readContentFrom(r)
		n += int64(m)
		if err = e; err == io.EOF {
			return n, nil
		}
	}
	return
}

// WriteTo writes all data from this buffer to w until there's no more data or when an error occurs.
// It returns the number of bytes written. Any error encountered during the read is also returned.
//
// Note that the number of bytes written includes both the data read by ReadFrom and any additional headers.
// If you only want to know how many bytes from the last ReadFrom were written, check BytesOverlapped.
func (b *clientHelloBuffer) WriteTo(w io.Writer) (n int64, err error) {
	if b.len > 0 {
		m, e := w.Write(b.data[:b.len])
		n = int64(m)
		if err = e; err != nil {
			return
		}
		// all bytes should have been written, by definition of Write method in io.Writer
		if m != b.len {
			err = io.ErrShortWrite
		}
	}
	return
}

// HasFullyReceived returns whether a complete TLS Client Hello packet has been assembled.
func (b *clientHelloBuffer) HasFullyReceived() bool {
	return b.valid && b.recordLen > 0 && b.len >= b.recordLen+tlsRecordHeaderSize
}

// BytesOverlapped returns the number of bytes actually copied from the io.Reader in ReadFrom(r)
// to io.Writer in WriteTo, ignoring any extra headers added by Split.
//
// Here's an example explaining it further:
//
//	_, _ := buf.ReadFrom([]byte{1,2})      // {1,2} are appended to buf
//	rn, _ := buf.ReadFrom([]byte{3,4,5,6}) // rn == 3, {3,4,5} are appended to buf
//	buf.Split(2)                           // will add some additional header bytes
//	// now assume buf contains {1,2,h,h,h,h,h,3,4,5}
//	wn, _ := buf.WriteTo(w)                // wn == 8, {1,2,h,h,h,h,h,3} are written to w
//	n := buf.BytesOverlapped(rn, wn)       // n == 1, because only byte {3} comes from the last ReadFrom
func (b *clientHelloBuffer) BytesOverlapped(rn, wn int64) int {
	//   ndata = 12:  1 2 3 4 h h h h h 5 6 7
	//       rn = 5:      | |           | | |
	//       wn = 6:  | | | | | |
	// overlap == 2:      ^ ^
	//       wn & h:  x x x x | | N N N

	if wn < int64(b.split) {
		// add all 5 header bytes to wn when splitted and wn doesn't overlap with h
		// if no splitting, this condition will never be satifsfied because wn always >= 0
		wn += tlsRecordHeaderSize
	} else if b.split > 0 && wn < int64(b.split+tlsRecordHeaderSize) {
		// fill all non-overlapped h bytes to wn (bytes marked as N above) when wn partially overlaps with h
		wn = int64(b.split + tlsRecordHeaderSize)
	}

	// now both wn and n contain either a 5-byte header or no header at all
	// the header bytes get cancelled out in the subtraction (wn - ndata) below
	// rn + wn = (left+overlap) + (overlap+right) = (left+overlap+right) + overlap = ndata + overlap
	if overlap := int(rn) + int(wn) - b.len; overlap >= 0 {
		return overlap
	}
	return 0
}

// Content returns the Client Hello packet content (without the 5 bytes header).
// It might return an incomplete content, the caller needs to make sure HasFullyReceived before calling this function.
func (b *clientHelloBuffer) Content() []byte {
	if b.len <= tlsRecordHeaderSize {
		return []byte{}
	}
	return b.data[tlsRecordHeaderSize:b.len]
}

// Split fragments the Client Hello packet into two TLS records at the specified 0-based splitBytes:
// [:splitBytes] and [splitBytes:]. Any necessary headers will be added to this buffer.
//
// If the packet has already be splitted before, a non-nil error and returned.
// If the split index is ≤ 0 or ≥ the total length, do nothing.
func (b *clientHelloBuffer) Split(splitBytes int) error {
	if b.split > 0 {
		return errors.New("packet has already been fragmented")
	}
	if !b.HasFullyReceived() || b.len != b.recordLen+tlsRecordHeaderSize {
		return errors.New("incomplete packet cannot be fragmented")
	}
	if splitBytes <= 0 || splitBytes >= b.recordLen {
		return nil
	}
	_ = b.data[b.len+tlsRecordHeaderSize-1] // bounds check to guarantee safety of writes below

	// the 2nd record starting point (including header), and move the 2nd record content 5 bytes to the right
	sz2 := b.recordLen - splitBytes
	b.split = tlsRecordHeaderSize + splitBytes
	b.len += tlsRecordHeaderSize

	if copy(b.data[b.split+tlsRecordHeaderSize:b.len], b.data[b.split:]) != sz2 {
		return errors.New("failed to split the second record")
	}

	putTLSClientHelloHeader(b.data[0:], uint16(splitBytes))
	putTLSClientHelloHeader(b.data[b.split:], uint16(sz2))
	return nil
}

// readHeaderFrom read a 5 bytes TLS Client Hello header from r into b.data[0:5].
func (b *clientHelloBuffer) readHeaderFrom(r io.Reader) (n int, err error) {
	if b.len >= tlsRecordHeaderSize {
		return 0, errors.New("header has already been read")
	}
	if len(b.data) < tlsRecordHeaderSize {
		return 0, errors.New("insufficient buffer to hold the header")
	}

	prevLen := b.len
	for err == nil && b.len < tlsRecordHeaderSize {
		m, e := r.Read(b.data[b.len:tlsRecordHeaderSize])
		err = e
		n += m
		b.len += m
	}

	if prevLen < tlsRecordWithTypeSize && b.len >= tlsRecordWithTypeSize {
		if !isTLSRecordTypeHandshake(b.data) {
			b.valid = false
			err = errors.Join(err, fmt.Errorf("not a handshake record: %w", errInvalidTLSClientHello))
		}
	}

	if prevLen < tlsRecordWithVersionHeaderSize && b.len >= tlsRecordWithVersionHeaderSize {
		if !isValidTLSProtocolVersion(b.data) {
			b.valid = false
			err = errors.Join(err, fmt.Errorf("not a valid TLS version: %w", errInvalidTLSClientHello))
		}
	}

	if prevLen < tlsRecordHeaderSize && b.len >= tlsRecordHeaderSize {
		if rl := recordLen(b.data); !isValidRecordLenForHandshake(rl) {
			b.valid = false
			err = errors.Join(err, fmt.Errorf("record length out of range: %w", errInvalidTLSClientHello))
		} else {
			b.recordLen = int(rl)
			// allocate space for 2 headers and 1 content (might be splitted into two contents)
			buf := make([]byte, b.recordLen+tlsRecordHeaderSize*2)
			if copy(buf, b.data[:tlsRecordHeaderSize]) != tlsRecordHeaderSize {
				err = errors.Join(err, errors.New("failed to copy header data"))
			} else {
				b.data = buf
			}
		}
	}
	return
}

// readContentFrom read a recordLen bytes TLS Client Hello record content from r into b.data[5:5+recordLen].
func (b *clientHelloBuffer) readContentFrom(r io.Reader) (n int, err error) {
	fullsz := tlsRecordHeaderSize + b.recordLen
	if b.len >= fullsz {
		return 0, errors.New("content has already been read")
	}
	if len(b.data) < fullsz {
		return 0, errors.New("insufficient buffer to hold the content")
	}

	for err == nil && b.len < fullsz {
		m, e := r.Read(b.data[b.len:fullsz])
		err = e
		n += m
		b.len += m
	}
	return
}
