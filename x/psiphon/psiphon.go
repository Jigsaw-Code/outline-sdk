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
	"errors"
	"fmt"
	"net"
	"runtime"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
	"github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon"
	psi "github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon"
)

type Dialer struct {
	cancel     context.CancelFunc
	controller *psi.Controller
}

func (d *Dialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	netConn, err := d.controller.Dial(addr, nil)
	if err != nil {
		return nil, err
	}
	return streamConn{netConn}, nil
}

func (d *Dialer) Close() {
	d.cancel()
}

type Config struct {
	config *psi.Config
}

func LoadConfig(configJSON []byte) (*Config, error) {
	config, err := psi.LoadConfig(configJSON)
	if err != nil {
		return nil, fmt.Errorf("config load failed: %w", err)
	}
	// Don't let Psiphon run its local proxies.
	config.DisableLocalHTTPProxy = true
	config.DisableLocalSocksProxy = true
	config.ClientPlatform = fmt.Sprintf("OutlineSDK/%s/%s", runtime.GOOS, runtime.GOARCH)
	// TODO(fortuna): Figure out a better way to do this to allow the user to override config options.
	err = config.Commit(false)
	if err != nil {
		return nil, fmt.Errorf("config commit failed: %w", err)
	}
	return &Config{config}, nil
}

func (cfg *Config) NewDialer(ctx context.Context) (*Dialer, error) {
	// Will receive a value when the tunnel has successfully connected.
	connected := make(chan struct{}, 1)
	// Will receive a value if an error occurs during the connection sequence.
	errCh := make(chan error, 1)

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
				case errCh <- err:
				default:
				}
				return
			}
			switch event.Type {
			case "EstablishTunnelTimeout":
				select {
				case errCh <- context.DeadlineExceeded:
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

	err := psiphon.OpenDataStore(&psiphon.Config{DataRootDirectory: cfg.config.DataRootDirectory})
	if err != nil {
		return nil, fmt.Errorf("failed to open Psiphon data store: %w", err)
	}
	needsCleanup := true
	defer func() {
		if needsCleanup {
			psiphon.CloseDataStore()
		}
	}()

	controller, err := psi.NewController(cfg.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Psiphon Controller: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	go controller.Run(ctx)

	// Wait for an active tunnel or error
	select {
	case <-connected:
		needsCleanup = false
		return &Dialer{cancel, controller}, nil
	case err := <-errCh:
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("failed to start Psiphon tunnel: %w", err)
		}
		return nil, err
	}
}

var _ transport.StreamDialer = (*Dialer)(nil)

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
