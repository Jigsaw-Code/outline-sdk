package oob

import (
	"io"
	"net"
	"sync"
	"syscall"
)

var defaultTTL = 64

type oobWriter struct {
	conn        *net.TCPConn
	resetTTL    sync.Once
	oobPosition int64
	fd          int
	oobByte     byte // Byte to send as OOB
	disOOB      bool // Flag to enable disOOB mode
}

var _ io.Writer = (*oobWriter)(nil)

type oobWriterReaderFrom struct {
	*oobWriter
	rf io.ReaderFrom
}

var _ io.ReaderFrom = (*oobWriterReaderFrom)(nil)

// NewOOBWriter creates an [io.Writer] that sends an OOB byte at the specified "oobPosition".
// If disOOB is enabled, it will apply the --disOOB strategy.
// "oobByte" specifies the value of the byte to send out-of-band.
func NewOOBWriter(conn *net.TCPConn, fd int, oobPosition int64, oobByte byte, disOOB bool) io.Writer {
	return &oobWriter{conn: conn, fd: fd, oobPosition: oobPosition, oobByte: oobByte, disOOB: disOOB}
}

func (w *oobWriterReaderFrom) ReadFrom(source io.Reader) (int64, error) {
	reader := io.MultiReader(io.LimitReader(source, w.oobPosition), source)
	written, err := w.rf.ReadFrom(reader)
	w.oobPosition -= written
	return written, err
}

func (w *oobWriter) Write(data []byte) (int, error) {
	var written int
	var err error

	if w.oobPosition > 0 && w.oobPosition < int64(len(data)) {
		firstPart := data[:w.oobPosition+1]
		secondPart := data[w.oobPosition:]

		// Split the data into two parts
		tmp := secondPart[0]
		secondPart[0] = w.oobByte

		err = w.send(firstPart, syscall.MSG_OOB)
		if err != nil {
			return written, err
		}
		written = int(w.oobPosition)
		secondPart[0] = tmp

		w.resetTTL.Do(func() {
			if w.disOOB {
				err = syscall.SetsockoptInt(w.fd, syscall.IPPROTO_IP, syscall.IP_TTL, defaultTTL)
			}
		})

		if err != nil {
			return written, err
		}
		data = secondPart
	}

	// Write the remaining data
	err = w.send(data, 0)
	written += len(data)
	return written, err
}

func (w *oobWriter) send(data []byte, flags int) error {
	// Use SyscallConn to access the underlying file descriptor safely
	rawConn, err := w.conn.SyscallConn()
	if err != nil {
		return err
	}

	// Use Control to execute Sendto on the file descriptor
	var sendErr error
	err = rawConn.Control(func(fd uintptr) {
		sendErr = syscall.Sendto(int(w.fd), data, flags, nil)
	})
	if err != nil {
		return err
	}
	return sendErr
}
