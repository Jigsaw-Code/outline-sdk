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
	"io"
	"testing"
	"time"

	psi "github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon"

	"github.com/stretchr/testify/require"
)

func TestNewPsiphonConfig_ParseCorrectly(t *testing.T) {
	config, err := newPsiphonConfig(&DialerConfig{
		ProviderConfig: json.RawMessage(`{
			"PropagationChannelId": "ID1",
			"SponsorId": "ID2"
		}`),
	})
	require.NoError(t, err)
	require.Equal(t, "ID1", config.PropagationChannelId)
	require.Equal(t, "ID2", config.SponsorId)
}

func TestNewPsiphonConfig_AcceptOkOptions(t *testing.T) {
	_, err := newPsiphonConfig(&DialerConfig{
		ProviderConfig: json.RawMessage(`{
		"DisableLocalHTTPProxy": true,
		"DisableLocalSocksProxy": true
	}`)})
	require.NoError(t, err)
}

func TestNewPsiphonConfig_RejectBadOptions(t *testing.T) {
	_, err := newPsiphonConfig(&DialerConfig{
		ProviderConfig: json.RawMessage(`{"DisableLocalHTTPProxy": false}`)})
	require.Error(t, err)

	_, err = newPsiphonConfig(&DialerConfig{
		ProviderConfig: json.RawMessage(`{"DisableLocalSocksProxy": false}`)})
	require.Error(t, err)
	require.Error(t, err)
}

func TestDialer_StartSuccessful(t *testing.T) {
	// Create minimal config.
	cfg := &DialerConfig{ProviderConfig: json.RawMessage(`{
  	  "PropagationChannelId": "test",
	  "SponsorId": "test"
	}`)}

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

	// We use a select because the error may happen before the notice writer is set.
	select {
	case w := <-wCh:
		// Notify fake tunnel establishment once we have the notice writer.
		psi.SetNoticeWriter(w)
		psi.NoticeTunnels(1)
	case err := <-errCh:
		t.Fatalf("Got error from Start: %v", err)
	}

	err := <-errCh
	require.NoError(t, err)
	require.NoError(t, dialer.Stop())
}

func TestDialerStart_Cancelled(t *testing.T) {
	cfg := &DialerConfig{ProviderConfig: json.RawMessage(`{
  	  "PropagationChannelId": "test",
	  "SponsorId": "test"
	}`)}
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
	cfg := &DialerConfig{ProviderConfig: json.RawMessage(`{
  	  "PropagationChannelId": "test",
	  "SponsorId": "test"
	}`)}
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
