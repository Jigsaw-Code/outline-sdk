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
// You will need to provide your own Psiphon config file, which you can get from the Psiphon team
// or [generate one yourself].
//
// [generate one yourself]: https://github.com/Psiphon-Labs/psiphon-tunnel-core/tree/master?tab=readme-ov-file#generate-configuration-data
package psiphon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
	"github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon"
	psi "github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon"
)

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
	// TODO(fortuna): Return something the user can edit.
	return &Config{config}, nil
}

// Dialer is a [transport.StreamDialer] that uses Psiphon to connect to a destination.
// There's only one possible Psiphon Dialer available at any time, which is accessible via [GetSingletonDialer].
//
// The Dialer must be configured first with [Dialer.Start] before it can be used, and [Dialer.Stop] must be
// called before you can start it again with a new configuration. Dialer.Stop should be called
// when you no longer need the Dialer in order to release resources.
type Dialer struct {
	mu             sync.Mutex
	cancel         context.CancelFunc
	controller     *psi.Controller
	controllerDone <-chan struct{}
}

var _ transport.StreamDialer = (*Dialer)(nil)

// DialStream implements [transport.StreamDialer].
func (d *Dialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	if d.controller == nil {
		return nil, errors.New("dialer has not been started yet")
	}
	netConn, err := d.controller.Dial(addr, nil)
	if err != nil {
		return nil, err
	}
	return streamConn{netConn}, nil
}

// The single [Dialer] we can have.
var singletonDialer Dialer

// Start configures and runs the Dialer. It must be called before you can use the Dialer.
func (d *Dialer) Start(ctx context.Context, config *Config) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cancel != nil || d.controller != nil {
		return errors.New("tried to start dialer that is alread running")
	}

	err := config.config.Commit(false)
	if err != nil {
		return fmt.Errorf("failed to commit config: %w", err)
	}

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

	err = psiphon.OpenDataStore(&psiphon.Config{DataRootDirectory: config.config.DataRootDirectory})
	if err != nil {
		return fmt.Errorf("failed to open data store: %w", err)
	}
	needsCleanup := true
	defer func() {
		if needsCleanup {
			psiphon.CloseDataStore()
		}
	}()

	controller, err := psi.NewController(config.config)
	if err != nil {
		return fmt.Errorf("failed to create Controller: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		controllerDone := make(chan struct{})
		d.controllerDone = controllerDone
		controller.Run(ctx)
		close(controllerDone)
	}()

	// Wait for an active tunnel or error
	select {
	case <-connected:
		needsCleanup = false
		d.cancel = cancel
		d.controller = controller
		return nil
	case err := <-errCh:
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("failed to run Controller: %w", err)
		}
		return err
	}
}

// Stop stops the Dialer background processes, releasing resources and allowing it to be reconfigured.
func (d *Dialer) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cancel == nil || d.controller == nil {
		return errors.New("tried to stop dialer that is not running")
	}
	d.cancel()
	// Wait for Controller to finish before cleaning up.
	<-d.controllerDone
	d.cancel = nil
	d.controller = nil
	psiphon.CloseDataStore()
	return nil
}

// GetSingletonDialer returns the single Psiphon dialer instance.
func GetSingletonDialer() *Dialer {
	return &singletonDialer
}

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
