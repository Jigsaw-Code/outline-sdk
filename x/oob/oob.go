package oob

import (
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"
	"time"
)

var defaultTTL = 64

type oobWriter struct {
	conn        *net.TCPConn
	resetTTL    sync.Once
	oobPosition int64
	fd          SocketDescriptor
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
func NewOOBWriter(conn *net.TCPConn, fd SocketDescriptor, oobPosition int64, oobByte byte, disOOB bool) io.Writer {
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

		err = w.send(firstPart, 0x01)
		if err != nil {
			return written, err
		}
		written = int(w.oobPosition)
		secondPart[0] = tmp

		time.Sleep(200 * time.Millisecond)

		w.resetTTL.Do(func() {
			if w.disOOB {
				err = setsockoptInt(w.fd, syscall.IPPROTO_IP, syscall.IP_TTL, defaultTTL)
			}
		})

		if err != nil {
			return written, fmt.Errorf("setsockopt IPPROTO_IP/IP_TTL error: %w", err)
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
		return fmt.Errorf("oob strategy was unable to get raw conn: %w", err)
	}

	// Use Control to execute Sendto on the file descriptor
	var sendErr error
	err = rawConn.Control(func(fd uintptr) {
		sendErr = sendTo(SocketDescriptor(fd), data, flags)
	})
	if err != nil {
		return fmt.Errorf("oob strategy was unable to control socket: %w", err)
	}
	if sendErr != nil {
		return fmt.Errorf("oob strategy was unable to send data: %w", sendErr)
	}
	return nil
}
