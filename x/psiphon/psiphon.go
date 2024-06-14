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
	"errors"
	"net"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	psi "github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
)

// The single [Dialer] we can have.
var singletonDialer = Dialer{
	startTunnel: startTunnel,
	tunnelDial:  tunnelDial,
}

var (
	errNotStartedDial       = errors.New("dialer has not been started yet")
	errNotStartedStop       = errors.New("tried to stop dialer that is not running")
	errTunnelTimeout        = errors.New("tunnel establishment timed out")
	errTunnelAlreadyStarted = errors.New("tunnel already started")
)

// DialerConfig specifies the parameters for [Dialer].
type DialerConfig struct {
	// Used as the directory for the datastore, remote server list, and obfuscasted
	// server list.
	// Empty string means the default will be used (current working directory).
	// Strongly recommended.
	DataRootDirectory string

	// Raw JSON config provided by Psiphon.
	ProviderConfig []byte
}

// Dialer is a [transport.StreamDialer] that uses Psiphon to connect to a destination.
// There's only one possible Psiphon Dialer available at any time, which is accessible via [GetSingletonDialer].
// The zero value of this type is invalid.
//
// The Dialer must be configured first with [Dialer.Start] before it can be used, and [Dialer.Stop] must be
// called before you can start it again with a new configuration. Dialer.Stop should be called
// when you no longer need the Dialer in order to release resources.
type Dialer struct {
	// The Psiphon tunnel. It is nil until the tunnel is started.
	tunnel atomic.Pointer[psi.PsiphonTunnel]
	// It is (and must be) okay for this function to be called multiple times concurrently
	// or in series.
	stop atomic.Pointer[func() error]

	// Controls the Dialer state and Psiphon's global state.
	mu sync.Mutex
	// Allows tests to override the tunnel creation.
	startTunnel func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error)
	// Allows tests to override the tunnel dialing.
	tunnelDial func(tunnel *psi.PsiphonTunnel, addr string) (net.Conn, error)
}

// Ensure that [Dialer] implements [transport.StreamDialer].
var _ transport.StreamDialer = (*Dialer)(nil)

func tunnelDial(tunnel *psi.PsiphonTunnel, addr string) (net.Conn, error) {
	return tunnel.Dial(addr)
}

// DialStream implements [transport.StreamDialer].
// The context is not used because Psiphon's implementation doesn't support it. If you need cancellation,
// you will need to add it independently.
func (d *Dialer) DialStream(unusedContext context.Context, addr string) (transport.StreamConn, error) {
	tunnel := d.tunnel.Load()
	if tunnel == nil {
		return nil, errNotStartedDial
	}
	netConn, err := d.tunnelDial(tunnel, addr)
	if err != nil {
		return nil, err
	}
	return streamConn{netConn}, nil
}

func startTunnel(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
	// Note that these parameters override anything in the provider config.
	clientPlatformAllowChars := func(r rune) bool {
		return !unicode.IsSpace(r) && r != '_'
	}
	goos := strings.Join(strings.FieldsFunc(runtime.GOOS, clientPlatformAllowChars), "-")
	goarch := strings.Join(strings.FieldsFunc(runtime.GOARCH, clientPlatformAllowChars), "-")
	clientPlatform := "outline-sdk_" + goos + "_" + goarch
	trueValue := true

	params := psi.Parameters{
		DataRootDirectory: &config.DataRootDirectory,
		ClientPlatform:    &clientPlatform,
		// Disable Psiphon's local proxy servers, which we don't use.
		DisableLocalSocksProxy: &trueValue,
		DisableLocalHTTPProxy:  &trueValue,
	}

	return psi.StartTunnel(ctx, config.ProviderConfig, "", params, nil, nil)
}

// Start configures and runs the Dialer. It must be called before you can use the Dialer. It returns when the tunnel is ready for use. It is safe to call concurrently with itself and other methods.
func (d *Dialer) Start(ctx context.Context, config *DialerConfig) error {
	if config == nil {
		return errors.New("config must not be nil")
	}

	// The mutex is locked only in this method, and is used to prevent concurrent calls to
	// Start from returning before the tunnel is started (or failed).
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.tunnel.Load() != nil {
		return errTunnelAlreadyStarted
	}
	// This function is the only place where d.tunnel gets set to a non-nil value, and we're inside a
	// locked mutex, so we can be sure that it will remain nil until we set it below.

	startDoneSignal := make(chan struct{})
	defer close(startDoneSignal)
	cancelCtx, cancel := context.WithCancel(ctx)

	// The stop function is not called from within a mutex.
	// It must be safe to call concurrently and multiple times.
	stop := func() error {
		// The fact that this stop function exists means that we are either in the process of
		// starting the tunnel or have already started it.

		// Cancelling the context will cause the tunnel to stop or stop connecting.
		cancel()

		// Wait for Start to return (and note that it may return success or error at this point).
		<-startDoneSignal

		// Swap the stop function to nil to indicate that the tunnel is stopped or certainly will be.
		tunnel := d.tunnel.Swap(nil)
		if tunnel == nil {
			// We were connecting, but not yet connected; we interrupted the connection
			// sequence by canceling the context. There is no further cleanup to do.
			return nil
		}

		// This will block until the tunnel is stopped.
		tunnel.Stop()
		return nil
	}

	// Note that if Stop is called between the beginning of Start and this line, it won't actually stop the tunnel.
	// This is an acceptable limitation and is very unlikely to occur, as there is no significant processing
	// before this point. (And if you're calling Stop that quickly and concurrently, then you're rolling the dice with whether Start has even begun.)
	d.stop.Store(&stop)

	// StartTunnel returns when a tunnel is established or an error occurs.
	tunnel, err := d.startTunnel(cancelCtx, config)

	if err != nil {
		// There are some specific error values that we want to return.
		if err == psi.ErrTimeout {
			// This can occur either because there was a timeout set in the tunnel config
			// or because the context deadline was exceeded.
			err = errTunnelTimeout
			if ctx.Err() == context.DeadlineExceeded {
				err = context.DeadlineExceeded
			}
		} else if ctx.Err() == context.Canceled {
			err = context.Canceled
		}

		// Ensure this is below the ctx.Err() checks above.
		cancel()
		// Canceling the context is the only cleanup to be done on error (which implies no tunnel), so clear the stop function.
		d.stop.Store(nil)

		return err
	}

	// We have a good tunnel and it's time to make it available.
	d.tunnel.Store(tunnel)

	return nil
}

// Stop stops the Dialer background processes, releasing resources and allowing it to be reconfigured.
// It returns when the Dialer is completely stopped.
func (d *Dialer) Stop() error {
	// The stop function should only be called once, so swap it to nil as we get it.
	stop := d.stop.Swap(nil)
	if stop == nil {
		return errNotStartedStop
	}
	// Note that stop is not being called within a mutex. It must not be, so that it can execute
	// during the Start method, which is entirely mutexed.
	return (*stop)()
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
