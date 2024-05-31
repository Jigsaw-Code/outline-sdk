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

package psiphon

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"testing"
	"time"

	psi "github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon"

	"github.com/stretchr/testify/require"
)

func TestParseConfig_ParseCorrectly(t *testing.T) {
	config, err := ParseConfig([]byte(`{
		"PropagationChannelId": "ID1",
		"SponsorId": "ID2"
	}`))
	require.NoError(t, err)
	require.Equal(t, "ID1", config.PropagationChannelId)
	require.Equal(t, "ID2", config.SponsorId)
}

func TestParseConfig_DefaultClientPlatform(t *testing.T) {
	config, err := ParseConfig([]byte(`{}`))
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("OutlineSDK_%v_%v", runtime.GOOS, runtime.GOARCH), config.ClientPlatform)
}

func TestParseConfig_OverrideClientPlatform(t *testing.T) {
	config, err := ParseConfig([]byte(`{"ClientPlatform": "win"}`))
	require.NoError(t, err)
	require.Equal(t, "win", config.ClientPlatform)
}

func TestParseConfig_AcceptOkOptions(t *testing.T) {
	_, err := ParseConfig([]byte(`{
		"DisableLocalHTTPProxy": true,
		"DisableLocalSocksProxy": true,
		"TargetApiProtocol": "ssh"
	}`))
	require.NoError(t, err)
}

func TestParseConfig_RejectBadOptions(t *testing.T) {
	_, err := ParseConfig([]byte(`{"DisableLocalHTTPProxy": false}`))
	require.Error(t, err)

	_, err = ParseConfig([]byte(`{"DisableLocalSocksProxy": false}`))
	require.Error(t, err)

	_, err = ParseConfig([]byte(`{"TargetApiProtocol": "web"}`))
	require.Error(t, err)
}

func TestParseConfig_RejectUnknownFields(t *testing.T) {
	_, err := ParseConfig([]byte(`{
		"PropagationChannelId": "ID",
		"UknownField": false
	}`))
	require.Error(t, err)
}

func TestDialer_StartSuccessful(t *testing.T) {
	// Create minimal config.
	cfg := &Config{}
	cfg.PropagationChannelId = "test"
	cfg.SponsorId = "test"

	// Intercept notice writer.
	dialer := GetSingletonDialer()
	wCh := make(chan io.Writer)
	dialer.setNoticeWriter = func(w io.Writer) {
		wCh <- w
	}
	defer func() {
		dialer.setNoticeWriter = psi.SetNoticeWriter
	}()

	errCh := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		errCh <- dialer.Start(ctx, cfg)
	}()

	// Notify fake tunnel establishment.
	w := <-wCh
	psi.SetNoticeWriter(w)
	psi.NoticeTunnels(1)

	err := <-errCh
	require.NoError(t, err)
	require.NoError(t, dialer.Stop())
}

func TestDialerStart_Cancelled(t *testing.T) {
	cfg := &Config{}
	cfg.PropagationChannelId = "test"
	cfg.SponsorId = "test"
	errCh := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		errCh <- GetSingletonDialer().Start(ctx, cfg)
	}()
	cancel()
	err := <-errCh
	require.ErrorIs(t, err, context.Canceled)
}

func TestDialerStart_Timeout(t *testing.T) {
	cfg := &Config{}
	cfg.PropagationChannelId = "test"
	cfg.SponsorId = "test"
	errCh := make(chan error)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()
	go func() {
		errCh <- GetSingletonDialer().Start(ctx, cfg)
	}()
	err := <-errCh
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestDialerDialStream_NotStarted(t *testing.T) {
	_, err := GetSingletonDialer().DialStream(context.Background(), "")
	require.ErrorIs(t, err, errNotStartedDial)
}

func TestDialerStop_NotStarted(t *testing.T) {
	err := GetSingletonDialer().Stop()
	require.ErrorIs(t, err, errNotStartedStop)
}
