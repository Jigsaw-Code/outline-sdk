// Copyright 2024 The Outline Authors
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

package wait_stream

import (
	"context"
	"errors"
	"net"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/sockopt"
)

type waitStreamDialer struct {
	dialer transport.StreamDialer
}

var _ transport.StreamDialer = (*waitStreamDialer)(nil)

func NewStreamDialer(dialer transport.StreamDialer) (transport.StreamDialer, error) {
	if dialer == nil {
		return nil, errors.New("argument dialer must not be nil")
	}
	return &waitStreamDialer{dialer: dialer}, nil
}

func (d *waitStreamDialer) DialStream(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	innerConn, err := d.dialer.DialStream(ctx, remoteAddr)
	if err != nil {
		return nil, err
	}

	tcpInnerConn, ok := innerConn.(*net.TCPConn)
	if !ok {
		return nil, errors.New("wait_stream strategy: expected base dialer to return TCPConn")
	}

	tcpOptions, err := sockopt.NewTCPOptions(tcpInnerConn)
	if err != nil {
		return nil, err
	}

	dw := NewWriter(innerConn, tcpOptions)

	return transport.WrapConn(innerConn, innerConn, dw), nil
}
