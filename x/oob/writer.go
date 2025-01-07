package oob

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/x/sockopt"
)

type oobWriter struct {
	conn        *net.TCPConn
	opts        sockopt.TCPOptions
	resetTTL    sync.Once
	setTTL      sync.Once
	oobPosition int64
	sd          SocketDescriptor
	oobByte     byte // Byte to send as OOB
	disOOB      bool // Flag to enable disOOB mode
	delay       time.Duration
}

var _ io.Writer = (*oobWriter)(nil)

type oobWriterReaderFrom struct {
	*oobWriter
	rf io.ReaderFrom
}

// NewWriter creates an [io.Writer] that sends an OOB byte at the specified "oobPosition".
// If disOOB is enabled, it will apply the --disOOB strategy.
// "oobByte" specifies the value of the byte to send out-of-band.
func NewWriter(
	conn *net.TCPConn,
	opts sockopt.TCPOptions,
	oobPosition int64,
	oobByte byte,
	disOOB bool,
	delay time.Duration,
) io.Writer {
	return &oobWriter{conn: conn, opts: opts, oobPosition: oobPosition, oobByte: oobByte, disOOB: disOOB, delay: delay}
}

func (w *oobWriter) Write(data []byte) (int, error) {
	var written int
	var err error

	if w.oobPosition > 0 && w.oobPosition < int64(len(data))-1 {
		firstPart := data[:w.oobPosition+1]
		secondPart := data[w.oobPosition:]

		// Split the data into two parts
		tmp := secondPart[0]
		secondPart[0] = w.oobByte

		var oldTTL int
		if w.disOOB {
			w.setTTL.Do(func() {
				oldTTL, err = w.opts.HopLimit()
				if err != nil {
					return
				}
				err = w.opts.SetHopLimit(0)
			})
			if err != nil {
				return written, fmt.Errorf("oob: new hop limit set error: %w", err)
			}
		}

		err = w.send(firstPart, MSG_OOB)
		if err != nil {
			return written, err
		}
		written = int(w.oobPosition)
		secondPart[0] = tmp

		if w.disOOB {
			w.resetTTL.Do(func() {
				err = w.opts.SetHopLimit(oldTTL)
			})
			if err != nil {
				return written, fmt.Errorf("oob: old hop limit set error: %w", err)
			}
		}

		data = secondPart

		time.Sleep(w.delay)
	}

	n, err := w.conn.Write(data)
	written += n

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
