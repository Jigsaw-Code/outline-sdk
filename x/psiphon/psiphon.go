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

package psiphon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
	psi "github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon"
)

// The single [Dialer] we can have.
var singletonDialer = Dialer{
	setNoticeWriter: psi.SetNoticeWriter,
}

var (
	errNotStartedDial = errors.New("dialer has not been started yet")
	errNotStartedStop = errors.New("tried to stop dialer that is not running")
	errTunnelTimeout  = errors.New("tunnel establishment timed out")
)

// DialerConfig specifies the parameters for [Dialer].
type DialerConfig struct {
	// Used as the directory for the datastore, remote server list, and obfuscasted
	// server list.
	// Empty string means the default will be used (current working directory).
	// Strongly recommended.
	DataRootDirectory string

	// Raw JSON config provided by Psiphon.
	ProviderConfig json.RawMessage
}

// Dialer is a [transport.StreamDialer] that uses Psiphon to connect to a destination.
// There's only one possible Psiphon Dialer available at any time, which is accessible via [GetSingletonDialer].
// The zero value of this type is invalid.
//
// The Dialer must be configured first with [Dialer.Start] before it can be used, and [Dialer.Stop] must be
// called before you can start it again with a new configuration. Dialer.Stop should be called
// when you no longer need the Dialer in order to release resources.
type Dialer struct {
	// Controls the Dialer state and Psiphon's global state.
	mu sync.Mutex
	// Used by DialStream.
	controller *psi.Controller
	// Used by Stop.
	stop func()
	// Allows for overriding the global notice writer for testing.
	setNoticeWriter func(io.Writer)
}

var _ transport.StreamDialer = (*Dialer)(nil)

// DialStream implements [transport.StreamDialer].
// The context is not used because Psiphon's implementation doesn't support it. If you need cancellation,
// you will need to add it independently.
func (d *Dialer) DialStream(unusedContext context.Context, addr string) (transport.StreamConn, error) {
	d.mu.Lock()
	controller := d.controller
	d.mu.Unlock()
	if controller == nil {
		return nil, errNotStartedDial
	}
	netConn, err := controller.Dial(addr, nil)
	if err != nil {
		return nil, err
	}
	return streamConn{netConn}, nil
}

func newPsiphonConfig(config *DialerConfig) (*psi.Config, error) {
	if config == nil {
		return nil, errors.New("config must not be nil")
	}
	// Validate keys. We parse as a map first because we need to check for the existence
	// of certain keys.
	var configMap map[string]interface{}
	if err := json.Unmarshal(config.ProviderConfig, &configMap); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	for key, value := range configMap {
		switch key {
		case "DisableLocalHTTPProxy", "DisableLocalSocksProxy":
			b, ok := value.(bool)
			if !ok {
				return nil, fmt.Errorf("field %v must be a boolean", key)
			}
			if b != true {
				return nil, fmt.Errorf("field %v must be true if set", key)
			}
		case "DataRootDirectory":
			return nil, errors.New("field DataRootDirectory must not be set in the provider config. Specify it in the DialerConfig instead.")
		}
	}

	// Parse provider config.
	pConfig, err := psi.LoadConfig(config.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("config load failed: %w", err)
	}

	// Force some Psiphon config defaults for the Outline SDK case.
	pConfig.DisableLocalHTTPProxy = true
	pConfig.DisableLocalSocksProxy = true
	pConfig.DataRootDirectory = config.DataRootDirectory

	return pConfig, nil
}

// Start configures and runs the Dialer. It must be called before you can use the Dialer. It returns when the tunnel is ready.
func (d *Dialer) Start(ctx context.Context, config *DialerConfig) error {
	pConfig, err := newPsiphonConfig(config)
	if err != nil {
		return err
	}

	// Will receive a value if an error occurs during the connection sequence.
	// It will be closed on succesful connection.
	errCh := make(chan error)

	// Start returns either when a tunnel is ready, or an error happens, whichever comes first.
	// When emitting the errors, we use a select statement to ensure the channel is being listened
	// on, to avoid a deadlock after the initial error.
	go func() {
		onTunnel := func() {
			select {
			case errCh <- nil:
			default:
			}
		}
		err := d.runController(ctx, pConfig, onTunnel)
		select {
		case errCh <- err:
		default:
		}
	}()

	// Wait for an active tunnel or error
	return <-errCh
}

func (d *Dialer) runController(ctx context.Context, pConfig *psi.Config, onTunnel func()) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stop != nil {
		return errors.New("tried to start dialer that is alread running")
	}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(context.Canceled)
	controllerDone := make(chan struct{})
	defer close(controllerDone)
	d.stop = func() {
		// Tell controller to stop.
		cancel(context.Canceled)
		// Wait for controller to return.
		<-controllerDone
	}

	// Set up NoticeWriter to receive events.
	d.setNoticeWriter(psi.NewNoticeReceiver(
		func(notice []byte) {
			var event clientlib.NoticeEvent
			err := json.Unmarshal(notice, &event)
			if err != nil {
				// This is unexpected and probably indicates something fatal has occurred.
				// We'll interpret it as a connection error and abort.
				cancel(fmt.Errorf("failed to unmarshal notice JSON: %w", err))
				return
			}
			switch event.Type {
			case "EstablishTunnelTimeout":
				cancel(errTunnelTimeout)
			case "Tunnels":
				count := event.Data["count"].(float64)
				if count > 0 {
					onTunnel()
				}
			}
		}))
	defer psi.SetNoticeWriter(io.Discard)

	err := pConfig.Commit(true)
	if err != nil {
		return fmt.Errorf("failed to commit config: %w", err)
	}

	err = psi.OpenDataStore(&psi.Config{DataRootDirectory: pConfig.DataRootDirectory})
	if err != nil {
		return fmt.Errorf("failed to open data store: %w", err)
	}
	defer psi.CloseDataStore()

	controller, err := psi.NewController(pConfig)
	if err != nil {
		return fmt.Errorf("failed to create Controller: %w", err)
	}
	d.controller = controller
	d.mu.Unlock()

	controller.Run(ctx)

	d.mu.Lock()
	d.controller = nil
	d.stop = nil
	return context.Cause(ctx)
}

// Stop stops the Dialer background processes, releasing resources and allowing it to be reconfigured.
// It returns when the Dialer is completely stopped.
func (d *Dialer) Stop() error {
	d.mu.Lock()
	stop := d.stop
	d.stop = nil
	d.mu.Unlock()
	if stop == nil {
		return errNotStartedStop
	}
	stop()
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
