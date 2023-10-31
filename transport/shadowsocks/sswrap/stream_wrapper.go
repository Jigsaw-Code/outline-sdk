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

package sswrap

import (
	"context"
	"fmt"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type StreamConnWrapper struct {
	key *EncryptionKey

	// SaltGenerator is used by Shadowsocks to generate the connection salts.
	// `SaltGenerator` can be `nil`, which defaults to [shadowsocks.RandomSaltGenerator].
	SaltGenerator SaltGenerator
}

func (w *StreamConnWrapper) WrapConn(ctx context.Context, proxyConn transport.StreamConn) (transport.StreamConn, error) {
	ssw, err := NewWriter(proxyConn, w.key, w.SaltGenerator)
	if err != nil {
		return nil, fmt.Errorf("failed to create Writer: %w", err)
	}
	ssr := NewReader(proxyConn, w.key)
	return transport.WrapConn(proxyConn, ssr, ssw), nil
}
