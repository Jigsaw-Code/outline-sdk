package oob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type oobWriter struct {
	writer      io.Writer
	oobPosition int64
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
func NewOOBWriter(writer io.Writer, oobPosition int64, oobByte byte, disOOB bool) io.Writer {
	ow := &oobWriter{writer: writer, oobPosition: oobPosition, oobByte: oobByte, disOOB: disOOB}
	if rf, ok := writer.(io.ReaderFrom); ok {
		return &oobWriterReaderFrom{ow, rf}
	}
	return ow
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

	if conn, ok := w.writer.(net.Conn); ok {
		if w.oobPosition > 0 && w.oobPosition < int64(len(data)) {
			// Write the first part with the regular writer
			written, err = w.writer.Write(data[:w.oobPosition])
			if err != nil {
				return written, err
			}

			// Send the specified OOB byte using the new sendOOBByte method
			_ = conn
			err = w.sendOOBByte(conn)
			if err != nil {
				return written, err
			}
			data = data[written:] // Skip the OOB byte
		}
	}
	// Write the remaining data
	n, err := w.writer.Write(data)
	written += n
	return written, err
}

// sendOOBByte sends the specified OOB byte over the provided connection.
// It sets the appropriate flags based on whether disOOB mode is enabled.
func (w *oobWriter) sendOOBByte(conn net.Conn) error {
	// Attempt to convert to *net.TCPConn to access SyscallConn
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return errors.New("connection is not a TCP connection")
	}

	oobData := []byte{w.oobByte}
	var flags int
	if w.disOOB {
		flags = syscall.MSG_OOB | syscall.MSG_DONTROUTE // Additional flag for disOOB mode
	} else {
		flags = syscall.MSG_OOB
	}

	// Use SyscallConn to access the underlying file descriptor safely
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return err
	}

	// Use Control to execute Sendto on the file descriptor
	var sendErr error
	err = rawConn.Control(func(fd uintptr) {
		sendErr = syscall.Sendto(int(fd), oobData, flags, nil)
		if sendErr != nil {
			fmt.Errorf("failed to send OOB byte: %v", sendErr)
		}
	})
	if err != nil {
		return err
	}
	return sendErr
}

// oobDialer is a dialer that applies the OOB and disOOB strategies.
type oobDialer struct {
	dialer      transport.StreamDialer
	oobPosition int64
	oobByte     byte
	disOOB      bool
}

// NewStreamDialerWithOOB creates a [transport.StreamDialer] that applies OOB byte sending at "oobPosition" and supports disOOB.
// "oobByte" specifies the value of the byte to send out-of-band.
func NewStreamDialerWithOOB(dialer transport.StreamDialer, oobPosition int64, oobByte byte, disOOB bool) (transport.StreamDialer, error) {
	if dialer == nil {
		return nil, errors.New("argument dialer must not be nil")
	}
	return &oobDialer{dialer: dialer, oobPosition: oobPosition, oobByte: oobByte, disOOB: disOOB}, nil
}

// DialStream implements [transport.StreamDialer].DialStream with OOB and disOOB support.
func (d *oobDialer) DialStream(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	innerConn, err := d.dialer.DialStream(ctx, remoteAddr)
	if err != nil {
		return nil, err
	}
	// Wrap connection with OOB and/or disOOB writer based on configuration
	return transport.WrapConn(innerConn, innerConn, NewOOBWriter(innerConn, d.oobPosition, d.oobByte, d.disOOB)), nil
}
