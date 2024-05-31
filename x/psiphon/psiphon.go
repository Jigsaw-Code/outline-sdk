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

//go:build psiphon

/*
Package psiphon provides adaptors to create StreamDialers that leverage [Psiphon] technology and
infrastructure to bypass network interference.

You will need to provide your own Psiphon config file, which you must acquire from the Psiphon team.
See the [Psiphon End-User License Agreement]. For more details, email them at sponsor@psiphon.ca.

For testing, you can [generate a Psiphon config yourself].

# License restrictions

Psiphon code is licensed as GPLv3, which you will have to take into account if you incorporate Psiphon logic into your app.
If you don't want your app to be GPL, consider acquiring an appropriate license when acquiring their services.

Note that a few of Psiphon's dependencies may impose additional restrictions. For example, github.com/hashicorp/golang-lru is MPL-2.0
and github.com/juju/ratelimit is LGPL-3.0. You can use [go-licenses] to analyze the licenses of your Go code dependencies.

To prevent accidental inclusion of unvetted licenses, you must use the "psiphon" build tag in order to use this package. Typically you do that with
"-tags psiphon".

[Psiphon]: https://psiphon.ca
[Psiphon End-User License Agreement]: https://psiphon.ca/en/license.html
[go-licenses]: https://github.com/google/go-licenses
[generate a Psiphon config yourself]: https://github.com/Psiphon-Labs/psiphon-tunnel-core/tree/master?tab=readme-ov-file#generate-configuration-data
*/
package psiphon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"strings"
	"sync"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
	psi "github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon"
	"github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon/common/parameters"
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

// ClientInfo specifies information about the client app that should be communicated
// to Psiphon for authentication and metrics.
type ClientInfo struct {
	// PropagationChannelId is a string identifier which indicates how the
	// Psiphon client was distributed. This parameter is required. This value
	// is supplied by and depends on the Psiphon Network.
	PropagationChannelId string

	// SponsorId is a string identifier which indicates who is sponsoring this
	// Psiphon client. This parameter is required. This value is supplied
	// by and depends on the Psiphon Network.
	SponsorId string

	// ClientVersion is the client version number that the client reports to
	// the server. The version number refers to the host client application,
	// not the core tunnel library.
	ClientVersion string

	// ClientPlatform is the client platform ("Windows", "Android", etc.) that
	// the client reports to the server.
	ClientPlatform string
}

// ServerListConfig specifies how Psiphon obtains server lists.
type ServerListConfig struct {
	// ObfuscatedServerListRootURLs is a list of URLs which specify root
	// locations from which to fetch obfuscated server list files. This value
	// is supplied by and depends on the Psiphon Network, and is typically
	// embedded in the client binary. All URLs must point to the same entity
	// with the same ETag. At least one DownloadURL must have
	// OnlyAfterAttempts = 0.
	ObfuscatedServerListRootURLs parameters.TransferURLs

	// RemoteServerListURLs is list of URLs which specify locations to fetch
	// out-of-band server entries. This facility is used when a tunnel cannot
	// be established to known servers. This value is supplied by and depends
	// on the Psiphon Network, and is typically embedded in the client binary.
	// All URLs must point to the same entity with the same ETag. At least one
	// TransferURL must have OnlyAfterAttempts = 0.
	RemoteServerListURLs parameters.TransferURLs

	// RemoteServerListSignaturePublicKey specifies a public key that's used
	// to authenticate the remote server list payload. This value is supplied
	// by and depends on the Psiphon Network, and is typically embedded in the
	// client binary.
	RemoteServerListSignaturePublicKey string

	// ServerEntrySignaturePublicKey is a base64-encoded, ed25519 public
	// key value used to verify individual server entry signatures. This value
	// is supplied by and depends on the Psiphon Network, and is typically
	// embedded in the client binary.
	ServerEntrySignaturePublicKey string

	// TargetServerEntry is an encoded server entry. When specified, this
	// server entry is used exclusively and all other known servers are
	// ignored; also, when set, ConnectionWorkerPoolSize is ignored and
	// the pool size is 1.
	TargetServerEntry string
}

// StorageConfig specifies where Psiphon should store its data.
type StorageConfig struct {
	// DataRootDirectory is the directory in which to store persistent files,
	// which contain information such as server entries. By default, current
	// working directory.
	//
	// Psiphon will assume full control of files under this directory. They may
	// be deleted, moved or overwritten.
	DataRootDirectory string
}

// Config specifies how the Psiphon dialer should behave.
type Config struct {
	ClientInfo
	ServerListConfig
	StorageConfig
}

// extendedConfig allows us to evaluate some settings that are present in the Psiphon config,
// but the user shouldn't be specifying.
type extendedConfig struct {
	Config

	// DisableLocalHTTPProxy disables running the local HTTP proxy.
	DisableLocalHTTPProxy *bool
	// DisableLocalSocksProxy disables running the local SOCKS proxy.
	DisableLocalSocksProxy *bool
	// TargetApiProtocol specifies whether to force use of "ssh" or "web" API
	// protocol. When blank, the default, the optimal API protocol is used.
	// Note that this capability check is not applied before the
	// "CandidateServers" count is emitted.
	//
	// This parameter is intended for testing and debugging only. Not all
	// parameters are supported in the legacy "web" API protocol, including
	// speed test samples.
	TargetApiProtocol *string
}

// ParseConfig parses the config JSON into a structure that can be further edited.
func ParseConfig(configJSON []byte) (*Config, error) {
	var extCfg extendedConfig

	// Set default values
	extCfg.ClientPlatform = strings.ReplaceAll(fmt.Sprintf("OutlineSDK_%s_%s", runtime.GOOS, runtime.GOARCH), " ", "_")

	decoder := json.NewDecoder(bytes.NewReader(configJSON))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&extCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// We ignore these fields in the config, but only if they are set to the proper values.
	if extCfg.DisableLocalHTTPProxy != nil && !*extCfg.DisableLocalHTTPProxy {
		return nil, fmt.Errorf("DisableLocalHTTPProxy must be true if set")
	}
	if extCfg.DisableLocalSocksProxy != nil && !*extCfg.DisableLocalSocksProxy {
		return nil, fmt.Errorf("DisableLocalSocksProxy must be true if set")
	}
	if extCfg.TargetApiProtocol != nil && *extCfg.TargetApiProtocol != "ssh" {
		return nil, fmt.Errorf(`TargetApiProtocol must be "ssh" if set`)
	}

	return &extCfg.Config, nil
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

// Start configures and runs the Dialer. It must be called before you can use the Dialer. It returns when the tunnel is ready.
func (d *Dialer) Start(ctx context.Context, config *Config) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to convert config to JSON")
	}
	pConfig, err := psi.LoadConfig(configJSON)
	if err != nil {
		return fmt.Errorf("config load failed: %w", err)
	}

	// Override some Psiphon defaults.
	pConfig.DisableLocalHTTPProxy = true
	pConfig.DisableLocalSocksProxy = true
	pConfig.TargetApiProtocol = "ssh"

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
