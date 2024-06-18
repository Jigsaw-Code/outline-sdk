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
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
	"github.com/stretchr/testify/require"
)

func newTestConfig(tb testing.TB) (*DialerConfig, func()) {
	tempDir, err := os.MkdirTemp("", "psiphon")
	require.NoError(tb, err)
	return &DialerConfig{
		DataRootDirectory: tempDir,
		ProviderConfig: json.RawMessage(`{
			"PropagationChannelId": "ID1",
			"SponsorId": "ID2"
		}`),
	}, func() { os.RemoveAll(tempDir) }
}

func TestDialer_Start_Successful(t *testing.T) {
	cfg, delete := newTestConfig(t)
	defer delete()

	dialer := GetSingletonDialer()
	startTunnel = func(ctx context.Context, configJSON []byte, embeddedServerEntryList string, params clientlib.Parameters, paramsDelta clientlib.ParametersDelta, noticeReceiver func(clientlib.NoticeEvent)) (retTunnel *clientlib.PsiphonTunnel, retErr error) {
		return &clientlib.PsiphonTunnel{}, nil
	}
	defer func() {
		startTunnel = clientlib.StartTunnel
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, dialer.Start(ctx, cfg))
	require.ErrorIs(t, dialer.Start(ctx, cfg), errAlreadyStarted)
	require.NoError(t, dialer.Stop())
	require.ErrorIs(t, dialer.Stop(), errNotStartedStop)
}

func TestDialer_Start_Cancelled(t *testing.T) {
	cfg, delete := newTestConfig(t)
	defer delete()
	errCh := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		errCh <- GetSingletonDialer().Start(ctx, cfg)
	}()
	cancel()
	err := <-errCh
	require.ErrorIs(t, err, context.Canceled)
}

func TestDialer_Start_Timeout(t *testing.T) {
	cfg, delete := newTestConfig(t)
	defer delete()
	errCh := make(chan error)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()
	go func() {
		errCh <- GetSingletonDialer().Start(ctx, cfg)
	}()
	err := <-errCh
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestDialer_DialStream_NotStarted(t *testing.T) {
	_, err := GetSingletonDialer().DialStream(context.Background(), "")
	require.ErrorIs(t, err, errNotStartedDial)
}

func TestDialer_Stop_NotStarted(t *testing.T) {
	err := GetSingletonDialer().Stop()
	require.ErrorIs(t, err, errNotStartedStop)
}
