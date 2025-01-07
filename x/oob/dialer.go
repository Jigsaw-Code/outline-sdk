package oob

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/sockopt"
)

// oobDialer is a dialer that applies the OOB and disOOB strategies.
type oobDialer struct {
	dialer      transport.StreamDialer
	opts        sockopt.TCPOptions
	oobByte     byte
	oobPosition int64
	disOOB      bool
	delay       time.Duration
}

// NewStreamDialer creates a [transport.StreamDialer] that applies OOB byte sending at "oobPosition" and supports disOOB.
// "oobByte" specifies the value of the byte to send out-of-band.
func NewStreamDialer(dialer transport.StreamDialer,
	oobPosition int64, oobByte byte, disOOB bool, delay time.Duration) (transport.StreamDialer, error) {
	if dialer == nil {
		return nil, errors.New("argument dialer must not be nil")
	}
	return &oobDialer{
		dialer:      dialer,
		oobPosition: oobPosition,
		oobByte:     oobByte,
		disOOB:      disOOB,
		delay:       delay,
	}, nil
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
		return nil, fmt.Errorf("oob: only works with direct TCP connections")
	}

	opts, err := sockopt.NewTCPOptions(tcpConn)
	if err != nil {
		return nil, fmt.Errorf("oob: unable to get TCP options: %w", err)
	}

	dw := NewWriter(tcpConn, opts, d.oobPosition, d.oobByte, d.disOOB, d.delay)

	return transport.WrapConn(innerConn, innerConn, dw), nil
}
