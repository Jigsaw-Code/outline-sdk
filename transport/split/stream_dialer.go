// Copyright 2023 The Outline Authors
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

package split

import (
	"context"
	"errors"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

// splitDialer is a [transport.StreamDialer] that implements the split strategy.
// Use [NewStreamDialer] to create new instances.
type splitDialer struct {
	dialer    transport.StreamDialer
	nextSplit SplitIterator
}

var _ transport.StreamDialer = (*splitDialer)(nil)

// NewStreamDialer creates a [transport.StreamDialer] that splits the outgoing stream according to nextSplit.
func NewStreamDialer(dialer transport.StreamDialer, nextSplit SplitIterator) (transport.StreamDialer, error) {
	if dialer == nil {
		return nil, errors.New("argument dialer must not be nil")
	}
	if nextSplit == nil {
		return nil, errors.New("argument nextSplit must not be nil")
	}
	return &splitDialer{dialer: dialer, nextSplit: nextSplit}, nil
}

// DialStream implements [transport.StreamDialer].DialStream.
func (d *splitDialer) DialStream(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	innerConn, err := d.dialer.DialStream(ctx, remoteAddr)
	if err != nil {
		return nil, err
	}
	return transport.WrapConn(innerConn, innerConn, NewWriter(innerConn, d.nextSplit)), nil
}
