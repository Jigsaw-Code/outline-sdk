// Copyright 2024 Jigsaw Operations LLC
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

// Package psiphon provides adaptors to use Psiphon as a StreamDialer.
//
// You will need to provide a [Psiphon config file].
//
// [Psiphon config file]: https://github.com/Psiphon-Labs/psiphon-tunnel-core/tree/master?tab=readme-ov-file#generate-configuration-data
package psiphon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
	psi "github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon"
)

type PsiphonDialer struct {
	cancel     context.CancelFunc
	controller *psi.Controller
}

func (d *PsiphonDialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	netConn, err := d.controller.Dial(addr, nil)
	if err != nil {
		return nil, err
	}
	return streamConn{netConn}, nil
}

func (d *PsiphonDialer) Close() {
	d.cancel()
}

func NewStreamDialer(configJSON []byte) (*PsiphonDialer, error) {
	config, err := psi.LoadConfig(configJSON)
	if err != nil {
		return nil, fmt.Errorf("config load failed: %w", err)
	}
	// Don't let Psiphon run its local proxies.
	config.DisableLocalHTTPProxy = true
	config.DisableLocalSocksProxy = true
	err = config.Commit(false)
	if err != nil {
		return nil, fmt.Errorf("config commit failed: %w", err)
	}

	// Will receive a value when the tunnel has successfully connected.
	connected := make(chan struct{}, 1)
	// Will receive a value if an error occurs during the connection sequence.
	errored := make(chan error, 1)

	// Set up NoticeWriter to receive events.
	psi.SetNoticeWriter(psi.NewNoticeReceiver(
		func(notice []byte) {
			var event clientlib.NoticeEvent
			err := json.Unmarshal(notice, &event)
			if err != nil {
				// This is unexpected and probably indicates something fatal has occurred.
				// We'll interpret it as a connection error and abort.
				err = fmt.Errorf("failed to unmarshal notice JSON: %w", err)
				select {
				case errored <- err:
				default:
				}
				return
			}
			switch event.Type {
			case "EstablishTunnelTimeout":
				select {
				case errored <- clientlib.ErrTimeout:
				default:
				}
			case "Tunnels":
				count := event.Data["count"].(float64)
				if count > 0 {
					select {
					case connected <- struct{}{}:
					default:
					}
				}
			}
		}))

	controller, err := psi.NewController(config)
	if err != nil {
		return nil, fmt.Errorf("controller creation failed: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go controller.Run(ctx)

	// Wait for an active tunnel or error
	select {
	case <-connected:
		return &PsiphonDialer{cancel, controller}, nil
	case err := <-errored:
		cancel()
		if err != clientlib.ErrTimeout {
			err = fmt.Errorf("tunnel start produced error: %w", err)
		}
		return nil, err
	}
}

var _ transport.StreamDialer = (*PsiphonDialer)(nil)

// streamConn wraps a [net.Conn] to provide a [transport.StreamConn] interface.
type streamConn struct {
	net.Conn
}

var _ transport.StreamConn = (*streamConn)(nil)

func (c streamConn) CloseWrite() error {
	return nil
}

func (c streamConn) CloseRead() error {
	return nil
}
