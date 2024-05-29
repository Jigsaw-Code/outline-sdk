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
	"io"
	"net"
)

var errNotAvailable = errors.New("not available in the Psiphon stub. Use the actual implementation instead.")

// See https://pkg.go.dev/github.com/Psiphon-Labs/psiphon-tunnel-core.

type Config struct {
	// ClientPlatform is the client platform ("Windows", "Android", etc.) that
	// the client reports to the server.
	ClientPlatform string

	// DataRootDirectory is the directory in which to store persistent files,
	// which contain information such as server entries. By default, current
	// working directory.
	//
	// Psiphon will assume full control of files under this directory. They may
	// be deleted, moved or overwritten.
	DataRootDirectory string

	// DisableLocalHTTPProxy disables running the local HTTP proxy.
	DisableLocalHTTPProxy bool
	// DisableLocalSocksProxy disables running the local SOCKS proxy.
	DisableLocalSocksProxy bool

	// TargetApiProtocol specifies whether to force use of "ssh" or "web" API
	// protocol. When blank, the default, the optimal API protocol is used.
	// Note that this capability check is not applied before the
	// "CandidateServers" count is emitted.
	//
	// This parameter is intended for testing and debugging only. Not all
	// parameters are supported in the legacy "web" API protocol, including
	// speed test samples.
	TargetApiProtocol string
}

type Controller struct{}

func LoadConfig(configJSON []byte) (*Config, error) {
	return nil, errNotAvailable
}

func (config *Config) Commit(migrateFromLegacyFields bool) error {
	return errNotAvailable
}

func (config *Config) GetDataStoreDirectory() string {
	return ""
}

func NewController(config *Config) (controller *Controller, err error) {
	return nil, errNotAvailable
}

func (controller *Controller) Run(ctx context.Context) {}

func (controller *Controller) Dial(remoteAddr string, downstreamConn net.Conn) (conn net.Conn, err error) {
	return nil, errNotAvailable
}

func SetNoticeWriter(writer io.Writer) {}

type NoticeReceiver struct{}

func NewNoticeReceiver(callback func([]byte)) *NoticeReceiver {
	return nil
}

func (receiver *NoticeReceiver) Write(p []byte) (n int, err error) {
	return 0, errNotAvailable
}

func OpenDataStore(config *Config) error {
	return errNotAvailable
}

func CloseDataStore() {}

func ImportEmbeddedServerEntries(
	ctx context.Context,
	config *Config,
	embeddedServerEntryListFilename string,
	embeddedServerEntryList string) error {
	return errNotAvailable
}
