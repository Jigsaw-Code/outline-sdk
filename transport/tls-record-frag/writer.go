package tlsrecordfrag

import (
	"errors"
	"io"
)

type tlsRecordFragWriter struct {
	writer      io.Writer
	prefixBytes uint32
}

const maxRecordLength = 16384

func NewWriter(writer io.Writer, prefixBytes uint32) *tlsRecordFragWriter {
	return &tlsRecordFragWriter{writer, prefixBytes}
}

func (w *tlsRecordFragWriter) dontFrag(first []byte, source io.Reader) (written int64, err error) {
	tmp, err := w.writer.Write(first)
	written = int64(tmp)
	w.prefixBytes = 0
	if err != nil {
		return written, err
	}
	n, err := io.Copy(w.writer, source)
	written += n
	return written, err
}

func (w *tlsRecordFragWriter) ReadFrom(source io.Reader) (written int64, err error) {
	if 0 < w.prefixBytes {
		var first [5]byte
		_, err := io.ReadFull(source, first[:])
		if err != nil {
			return 0, err
		}
		recordLength := uint32(first[3]) << 8 | uint32(first[4])
		if w.prefixBytes >= recordLength {
			return w.dontFrag(first[:], source)
		}
		if recordLength > maxRecordLength {
			return 0, errors.New("Broken handshake message")
		}
		buf := make([]byte, recordLength+10)
		n2, err := io.ReadFull(source, buf[5:5+w.prefixBytes])
		if err != nil {
			w.prefixBytes = 0
			return 0, err
		}
		n3, err := io.ReadFull(source, buf[10+w.prefixBytes:])
		if err != nil {
			w.prefixBytes = 0
			return 0, err
		}

		header := first[:3]

		copy(buf, header)
		buf[3] = byte(uint32(n2) >> 8)
		buf[4] = byte(uint32(n2) & 0xff)

		copy(buf[5+n2:], header)
		buf[5+n2+3] = byte(uint32(n3) >> 8)
		buf[5+n2+4] = byte(uint32(n3) & 0xff)

		tmp, err := w.writer.Write(buf)
		w.prefixBytes = 0
		written = int64(tmp)
		if err != nil {
			return written, err
		}
	}
	n, err := io.Copy(w.writer, source)
	written += n
	return written, err
}

func (w *tlsRecordFragWriter) Write(data []byte) (written int, err error) {
	length := len(data)
	if length > 5+maxRecordLength {
		return 0, errors.New("Broken handshake message")
	}
	if 0 < w.prefixBytes && w.prefixBytes < uint32(length -5) {
		buf := make([]byte, length+5)
		header := data[:3]
		record1 := data[5 : 5+w.prefixBytes]
		record2 := data[5+w.prefixBytes:]

		copy(buf, header)
		buf[3] = byte(w.prefixBytes >> 8)
		buf[4] = byte(w.prefixBytes & 0xff)
		copy(buf[5:], record1)

		copy(buf[5+w.prefixBytes:], header)
		buf[5+3+w.prefixBytes] = byte(len(record2) >> 8)
		buf[5+4+w.prefixBytes] = byte(len(record2) & 0xff)
		copy(buf[5+5+w.prefixBytes:], record2)

		w.prefixBytes = 0
		return w.writer.Write(buf)
	}
	w.prefixBytes = 0
	return w.writer.Write(data)
}
