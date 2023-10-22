package tlsrecordfrag

import (
	"context"
	"errors"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type tlsRecordFragDialer struct {
	dialer     transport.StreamDialer
	splitPoint uint32
}

var _ transport.StreamDialer = (*tlsRecordFragDialer)(nil)

// NewStreamDialer creates a [transport.StreamDialer] that splits the Client Hello Message
func NewStreamDialer(dialer transport.StreamDialer, prefixBytes uint32) (transport.StreamDialer, error) {
	if dialer == nil {
		return nil, errors.New("argument dialer must not be nil")
	}
	return &tlsRecordFragDialer{dialer: dialer, splitPoint: prefixBytes}, nil
}

// Dial implements [transport.StreamDialer].Dial.
func (d *tlsRecordFragDialer) Dial(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	innerConn, err := d.dialer.Dial(ctx, remoteAddr)
	if err != nil {
		return nil, err
	}
	return transport.WrapConn(innerConn, innerConn, NewWriter(innerConn, d.splitPoint)), nil
}