// Copyright 2019 Jigsaw Operations LLC
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

package transport

import (
	"context"
)

// Endpoint represents an endpoint that can be used to established connections to a fixed destination.
type Endpoint[Conn any] interface {
	// Connect creates a connection bound to an endpoint, returning the connection.
	Connect(ctx context.Context) (Conn, error)
}

// DialerEndpoint is an [Endpoint] that connects to the given address using the given [Dialer].
// Useful if you talk repeatedly to an endpoint, such as a proxy or DNS resolver, or
// if you find yourself passing a (dialer, address) pair around.
func NewDialerEndpoint[C any](dialer Dialer[C], address string) Endpoint[C] {
	return &dialerEndpoint[C]{dialer, address}
}

type dialerEndpoint[Conn any] struct {
	Dialer  Dialer[Conn]
	Address string
}

var _ Endpoint[any] = (*dialerEndpoint[any])(nil)

// Connect implements [Endpoint].Connect.
func (e *dialerEndpoint[Conn]) Connect(ctx context.Context) (Conn, error) {
	return e.Dialer.Dial(ctx, e.Address)
}

// Dialer provides a way to dial a destination and establish connections.
type Dialer[Conn any] interface {
	// Dial connects to `raddr`.
	// `raddr` has the form `host:port`, where `host` can be a domain name or IP address.
	Dial(ctx context.Context, raddr string) (Conn, error)
}
