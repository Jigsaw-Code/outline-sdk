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
)

// See https://pkg.go.dev/github.com/Psiphon-Labs/psiphon-tunnel-core.

type Config struct{}

type Controller struct{}

func LoadConfig(configJSON []byte) (*Config, error) {
	return nil, errors.New("not available")
}

func (config *Config) Commit(migrateFromLegacyFields bool) error {
	return errors.New("not available")
}

func NewController(config *Config) (controller *Controller, err error) {
	return nil, errors.New("not available")
}

func (controller *Controller) Run(ctx context.Context) {}

func (controller *Controller) Dial(remoteAddr string, downstreamConn net.Conn) (conn net.Conn, err error) {
	return nil, errors.New("not available")
}

func GetNotice(notice []byte) (noticeType string, payload map[string]interface{}, err error) {
	return "", nil, errors.New("not available")
}

type NoticeReceiver struct{}

func NewNoticeReceiver(callback func([]byte)) *NoticeReceiver {
	return nil
}
