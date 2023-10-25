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

package tlsrecordfrag

import (
	"context"
	"errors"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type tlsRecordFragDialer struct {
	dialer      transport.StreamDialer
	prefixBytes int32
}

var _ transport.StreamDialer = (*tlsRecordFragDialer)(nil)

// NewStreamDialer creates a [transport.StreamDialer] that splits the Client Hello Message
func NewStreamDialer(dialer transport.StreamDialer, prefixBytes int32) (transport.StreamDialer, error) {
	if dialer == nil {
		return nil, errors.New("argument dialer must not be nil")
	}
	return &tlsRecordFragDialer{dialer: dialer, prefixBytes: prefixBytes}, nil
}

// Dial implements [transport.StreamDialer].Dial.
func (d *tlsRecordFragDialer) Dial(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	innerConn, err := d.dialer.Dial(ctx, remoteAddr)
	if err != nil {
		return nil, err
	}
	return transport.WrapConn(innerConn, innerConn, NewWriter(innerConn, d.prefixBytes)), nil
}
