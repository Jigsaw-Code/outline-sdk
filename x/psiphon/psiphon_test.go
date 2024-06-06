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
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	psi "github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
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

func TestDialer_Start_Success(t *testing.T) {
	// Create minimal config.
	cfg, delete := newTestConfig(t)
	defer delete()

	dialer := GetSingletonDialer()

	// Set a no-op startTunnel
	dialer.startTunnel = func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
		return &psi.PsiphonTunnel{}, nil
	}
	defer func() {
		dialer.startTunnel = startTunnel
	}()

	errCh := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		errCh <- dialer.Start(ctx, cfg)
	}()

	err := <-errCh
	require.NoError(t, err)
	require.NoError(t, dialer.Stop())
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
	GetSingletonDialer().Stop()
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
	GetSingletonDialer().Stop()
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

func TestDialer_Start_AlreadyStarted(t *testing.T) {
	// Create minimal config.
	cfg, delete := newTestConfig(t)
	defer delete()

	dialer := GetSingletonDialer()

	// Set a no-op startTunnel
	dialer.startTunnel = func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
		return &psi.PsiphonTunnel{}, nil
	}
	defer func() {
		dialer.startTunnel = startTunnel
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := dialer.Start(ctx, cfg)
	require.NoError(t, err)

	err = dialer.Start(ctx, cfg)
	require.ErrorIs(t, err, errTunnelAlreadyStarted)

	require.NoError(t, dialer.Stop())

	err = dialer.Start(ctx, cfg)
	require.NoError(t, err)

	require.NoError(t, dialer.Stop())
}

func TestDialer_Stop_AlreadyStopped(t *testing.T) {
	// Create minimal config.
	cfg, delete := newTestConfig(t)
	defer delete()

	dialer := GetSingletonDialer()

	// Set a no-op startTunnel
	dialer.startTunnel = func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
		return &psi.PsiphonTunnel{}, nil
	}
	defer func() {
		dialer.startTunnel = startTunnel
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := dialer.Start(ctx, cfg)
	require.NoError(t, err)

	require.NoError(t, dialer.Stop())

	require.ErrorIs(t, dialer.Stop(), errNotStartedStop)
}

func TestDialer_DialStream_Stopped(t *testing.T) {
	// Create minimal config.
	cfg, delete := newTestConfig(t)
	defer delete()

	dialer := GetSingletonDialer()

	// Regardless of what we do below, restore the real startTunnel.
	defer func() {
		dialer.startTunnel = startTunnel
	}()

	// Set a no-op startTunnel
	dialer.startTunnel = func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
		return &psi.PsiphonTunnel{}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	err := dialer.Start(ctx, cfg)

	require.NoError(t, err)

	require.NoError(t, dialer.Stop())

	_, err = dialer.DialStream(context.Background(), "")
	require.ErrorIs(t, err, errNotStartedDial)

	cancel()
}

func TestDialer_DialStream_Success(t *testing.T) {
	// Create minimal config.
	cfg, delete := newTestConfig(t)
	defer delete()

	dialer := GetSingletonDialer()

	// Regardless of what we do below, restore the real startTunnel.
	defer func() {
		dialer.startTunnel = startTunnel
		dialer.tunnelDial = tunnelDial
	}()

	// Set a no-op startTunnel
	dialer.startTunnel = func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
		return &psi.PsiphonTunnel{}, nil
	}

	// Set a no-op tunnelDial
	dialer.tunnelDial = func(tunnel *psi.PsiphonTunnel, addr string) (net.Conn, error) {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := dialer.Start(ctx, cfg)
	require.NoError(t, err)

	_, err = dialer.DialStream(context.Background(), "")
	require.NoError(t, err)

	require.NoError(t, dialer.Stop())
}

func TestDialer_DialStream_Error(t *testing.T) {
	// Create minimal config.
	cfg, delete := newTestConfig(t)
	defer delete()

	dialer := GetSingletonDialer()

	// Regardless of what we do below, restore the real startTunnel.
	defer func() {
		dialer.startTunnel = startTunnel
		dialer.tunnelDial = tunnelDial
	}()

	// Set a no-op startTunnel
	dialer.startTunnel = func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
		return &psi.PsiphonTunnel{}, nil
	}

	// Force tunnelDial to always error
	dialer.tunnelDial = func(tunnel *psi.PsiphonTunnel, addr string) (net.Conn, error) {
		return nil, fmt.Errorf("error")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := dialer.Start(ctx, cfg)
	require.NoError(t, err)

	_, err = dialer.DialStream(context.Background(), "")
	require.Error(t, err)

	require.NoError(t, dialer.Stop())
}

func TestDialer_Concurrent(t *testing.T) {
	// Create minimal config.
	cfg, delete := newTestConfig(t)
	defer delete()

	dialer := GetSingletonDialer()

	// Regardless of what we do below, restore the real startTunnel.
	defer func() {
		dialer.startTunnel = startTunnel
	}()

	errCh1 := make(chan error)
	errCh2 := make(chan error)

	//
	// Two concurrent calls to Start.
	//
	dialer.startTunnel = func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
		return &psi.PsiphonTunnel{}, nil
	}
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errCh1 <- dialer.Start(ctx, cfg)
	}()
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errCh2 <- dialer.Start(ctx, cfg)
	}()
	err1 := <-errCh1
	err2 := <-errCh2
	require.NotEqual(t, (err1 == nil), (err2 == nil), "Expected one of the calls to fail: %v %v", err1, err2)

	require.NoError(t, dialer.Stop())

	//
	// Two concurrent calls to Start, ensuring that the first is still in progress when the second starts.
	// Set a no-op startTunnel that will take a while.
	//
	dialer.startTunnel = func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
		// Wait a bit before returning.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
		return &psi.PsiphonTunnel{}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		errCh1 <- dialer.Start(ctx, cfg)
	}()
	go func() {
		time.Sleep(20 * time.Millisecond) // delay the start a bit
		errCh2 <- dialer.Start(ctx, cfg)
	}()
	err1 = <-errCh1
	err2 = <-errCh2
	cancel()
	require.NoError(t, err1)
	require.ErrorIs(t, err2, errTunnelAlreadyStarted)
	require.NoError(t, dialer.Stop())

	//
	// Force Stop to occur during Start. (Stopping before and after is tested elsewhere.)
	// Set a no-op startTunnel that will take a while.
	//
	dialer.startTunnel = func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
		// Wait a bit before returning.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
		return &psi.PsiphonTunnel{}, nil
	}
	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		errCh1 <- dialer.Start(ctx, cfg)
	}()
	go func() {
		time.Sleep(20 * time.Millisecond) // delay the stop a bit
		errCh2 <- dialer.Stop()
	}()
	err1 = <-errCh1
	err2 = <-errCh2
	cancel()
	require.ErrorIs(t, err1, context.Canceled)
	require.NoError(t, err2)

	//
	// Call Stop twice concurrently.
	//
	dialer.startTunnel = func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
		return &psi.PsiphonTunnel{}, nil
	}
	ctx, cancel = context.WithCancel(context.Background())
	require.NoError(t, dialer.Start(ctx, cfg))
	go func() {
		errCh1 <- dialer.Stop()
	}()
	go func() {
		errCh2 <- dialer.Stop()
	}()
	err1 = <-errCh1
	err2 = <-errCh2
	cancel()
	require.NotEqual(t, (err1 == nil), (err2 == nil), "Expected one of the calls to fail: %v %v", err1, err2)

	//
	// Call DialStream concurrently with Start.
	//
	dialer.startTunnel = func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
		// Wait a bit before returning.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
		return &psi.PsiphonTunnel{}, nil
	}
	dialer.tunnelDial = func(tunnel *psi.PsiphonTunnel, addr string) (net.Conn, error) {
		return nil, nil
	}
	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		errCh1 <- dialer.Start(ctx, cfg)
	}()
	go func() {
		time.Sleep(20 * time.Millisecond) // delay the dial a bit, to ensure start is in progress
		_, err := dialer.DialStream(context.Background(), "")
		errCh2 <- err
	}()
	err1 = <-errCh1
	err2 = <-errCh2
	cancel()
	require.NoError(t, err1)
	require.ErrorIs(t, err2, errNotStartedDial)
	require.NoError(t, dialer.Stop())

	//
	// Call DialStream concurrently with Stop.
	//
	dialer.startTunnel = func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
		return &psi.PsiphonTunnel{}, nil
	}
	dialer.tunnelDial = func(tunnel *psi.PsiphonTunnel, addr string) (net.Conn, error) {
		return nil, nil
	}
	ctx, cancel = context.WithCancel(context.Background())
	require.NoError(t, dialer.Start(ctx, cfg))
	go func() {
		_, err := dialer.DialStream(context.Background(), "")
		errCh1 <- err
	}()
	go func() {
		errCh2 <- dialer.Stop()
	}()
	err1 = <-errCh1
	err2 = <-errCh2
	cancel()
	// The dial should either not fail (because it beat the stop) or fail with the right error.
	if err1 != nil {
		require.ErrorIs(t, err1, errNotStartedDial)
	}
	require.NoError(t, err2)

	//
	// Call DialStream twice concurrently.
	//
	dialer.startTunnel = func(ctx context.Context, config *DialerConfig) (*psi.PsiphonTunnel, error) {
		return &psi.PsiphonTunnel{}, nil
	}
	dialer.tunnelDial = func(tunnel *psi.PsiphonTunnel, addr string) (net.Conn, error) {
		return nil, nil
	}
	ctx, cancel = context.WithCancel(context.Background())
	require.NoError(t, dialer.Start(ctx, cfg))
	go func() {
		_, err := dialer.DialStream(context.Background(), "")
		errCh1 <- err
	}()
	go func() {
		_, err := dialer.DialStream(context.Background(), "")
		errCh2 <- err
	}()
	err1 = <-errCh1
	err2 = <-errCh2
	cancel()
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, dialer.Stop())

}
