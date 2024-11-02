package oob

import (
	"context"
	"errors"
	"fmt"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"net"
	"syscall"
)

// oobDialer is a dialer that applies the OOB and disOOB strategies.
type oobDialer struct {
	dialer      transport.StreamDialer
	oobByte     byte
	oobPosition int64
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
	// this strategy only works when we set TCP as a strategy
	tcpConn, ok := innerConn.(*net.TCPConn)
	if !ok {
		return nil, fmt.Errorf("split strategy only works with direct TCP connections")
	}

	file, err := tcpConn.File()
	if err != nil {
		return nil, fmt.Errorf("split strategy was unable to get conn fd: %w", err)
	}

	fd := int(file.Fd())

	if d.disOOB {
		err = tcpConn.SetNoDelay(true)
		if err != nil {
			return nil, fmt.Errorf("setting tcp NO_DELAY failed: %w", err)
		}

		err = syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_TTL, 1)
		if err != nil {
			return nil, fmt.Errorf("setsockopt IPPROTO_IP/IP_TTL error: %w", err)
		}
	}

	dw := NewOOBWriter(tcpConn, fd, d.oobPosition, d.oobByte, d.disOOB)

	return transport.WrapConn(innerConn, innerConn, dw), nil
}
