// Copyright 2024 The Outline Authors
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
	"errors"
	"net"
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
	dialer := GetSingletonDialer()
	startTunnel = func(ctx context.Context, config *DialerConfig) (psiphonTunnel, error) {
		return &clientlib.PsiphonTunnel{}, nil
	}
	defer func() {
		startTunnel = psiphonStartTunnel
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, dialer.Start(ctx, nil))
	require.NotNil(t, dialer.tunnel)
	require.ErrorIs(t, dialer.Start(ctx, nil), errAlreadyStarted)
	require.NoError(t, dialer.Stop())
	require.Nil(t, dialer.tunnel)
	require.ErrorIs(t, dialer.Stop(), errNotStartedStop)
	require.NoError(t, dialer.Start(ctx, nil))
	require.NoError(t, dialer.Stop())
}

func TestDialer_StopOnStart(t *testing.T) {
	dialer := GetSingletonDialer()
	startCalled := make(chan struct{})
	startTunnel = func(ctx context.Context, config *DialerConfig) (psiphonTunnel, error) {
		startCalled <- struct{}{}
		select {
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		}
	}
	defer func() {
		startTunnel = psiphonStartTunnel
	}()

	resultCh := make(chan error)
	go func() {
		resultCh <- dialer.Start(context.Background(), nil)
	}()
	<-startCalled
	require.NoError(t, dialer.Stop())
	require.Error(t, <-resultCh)
}

func TestDialer_StartOnStart(t *testing.T) {
	dialer := GetSingletonDialer()
	startCalled := make(chan struct{})
	startTunnel = func(ctx context.Context, config *DialerConfig) (psiphonTunnel, error) {
		startCalled <- struct{}{}
		select {
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		}
	}
	defer func() {
		startTunnel = psiphonStartTunnel
	}()

	resultCh := make(chan error)
	go func() {
		resultCh <- dialer.Start(context.Background(), nil)
	}()
	<-startCalled
	startTunnel = func(ctx context.Context, config *DialerConfig) (psiphonTunnel, error) {
		return nil, errors.New("failed to start")
	}
	require.ErrorIs(t, dialer.Start(context.Background(), nil), errAlreadyStarted)
	require.NoError(t, dialer.Stop())
	require.Error(t, <-resultCh)
}

func TestDialer_Start_NilConfig(t *testing.T) {
	require.Error(t, GetSingletonDialer().Start(context.Background(), nil))
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

func TestDialer_CancelledAfterStart_DoesntCloseTunnel(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "psiphon")
	cfg := &DialerConfig{
		DataRootDirectory: tempDir,
		ProviderConfig: json.RawMessage(`{
			"SponsorId" : "DB4A6B5E997A4E48",
			"PropagationChannelId" : "60F2A5F62855A295",

			"ClientPlatform" : "outline",
			"ClientVersion" : "1",

			"DisableLocalSocksProxy" : true,
			"DisableLocalHTTPProxy" : true,
			"EmitDiagnosticNotices" : true,
			"EstablishTunnelTimeoutSeconds" : 10,

			"ObfuscatedServerListRootURLs" : [{"URL": "aHR0cHM6Ly9zMy5hbWF6b25hd3MuY29tL3BzaXBob24vd2ViL29jdmwtMG9iai1zMnZnL29zbA==", "OnlyAfterAttempts": 0, "SkipVerify": false}, {"URL": "aHR0cHM6Ly93d3cucmVzdWx0c3VuaXZlcnNhbHVubGltaXRlZGpoLmNvbS93ZWIvb2N2bC0wb2JqLXMydmcvb3Ns", "OnlyAfterAttempts": 2, "SkipVerify": true}, {"URL": "aHR0cHM6Ly93d3cuYnJhbmRpbmd1c2FnYW1lcmVwLmNvbS93ZWIvb2N2bC0wb2JqLXMydmcvb3Ns", "OnlyAfterAttempts": 2, "SkipVerify": true}, {"URL": "aHR0cHM6Ly93d3cuYmxvZ3NmbWNhbmNlcmNpdGl6ZW4uY29tL3dlYi9vY3ZsLTBvYmotczJ2Zy9vc2w=", "OnlyAfterAttempts": 2, "SkipVerify": true}],
			"RemoteServerListURLs" : [{"URL": "aHR0cHM6Ly9zMy5hbWF6b25hd3MuY29tL3BzaXBob24vd2ViL29jdmwtMG9iai1zMnZnL3NlcnZlcl9saXN0X2NvbXByZXNzZWQ=", "OnlyAfterAttempts": 0, "SkipVerify": false}, {"URL": "aHR0cHM6Ly93d3cucmVzdWx0c3VuaXZlcnNhbHVubGltaXRlZGpoLmNvbS93ZWIvb2N2bC0wb2JqLXMydmcvc2VydmVyX2xpc3RfY29tcHJlc3NlZA==", "OnlyAfterAttempts": 2, "SkipVerify": true}, {"URL": "aHR0cHM6Ly93d3cuYnJhbmRpbmd1c2FnYW1lcmVwLmNvbS93ZWIvb2N2bC0wb2JqLXMydmcvc2VydmVyX2xpc3RfY29tcHJlc3NlZA==", "OnlyAfterAttempts": 2, "SkipVerify": true}, {"URL": "aHR0cHM6Ly93d3cuYmxvZ3NmbWNhbmNlcmNpdGl6ZW4uY29tL3dlYi9vY3ZsLTBvYmotczJ2Zy9zZXJ2ZXJfbGlzdF9jb21wcmVzc2Vk", "OnlyAfterAttempts": 2, "SkipVerify": true}],
			"RemoteServerListSignaturePublicKey" : "MIICIDANBgkqhkiG9w0BAQEFAAOCAg0AMIICCAKCAgEAt7Ls+/39r+T6zNW7GiVpJfzq/xvL9SBH5rIFnk0RXYEYavax3WS6HOD35eTAqn8AniOwiH+DOkvgSKF2caqk/y1dfq47Pdymtwzp9ikpB1C5OfAysXzBiwVJlCdajBKvBZDerV1cMvRzCKvKwRmvDmHgphQQ7WfXIGbRbmmk6opMBh3roE42KcotLFtqp0RRwLtcBRNtCdsrVsjiI1Lqz/lH+T61sGjSjQ3CHMuZYSQJZo/KrvzgQXpkaCTdbObxHqb6/+i1qaVOfEsvjoiyzTxJADvSytVtcTjijhPEV6XskJVHE1Zgl+7rATr/pDQkw6DPCNBS1+Y6fy7GstZALQXwEDN/qhQI9kWkHijT8ns+i1vGg00Mk/6J75arLhqcodWsdeG/M/moWgqQAnlZAGVtJI1OgeF5fsPpXu4kctOfuZlGjVZXQNW34aOzm8r8S0eVZitPlbhcPiR4gT/aSMz/wd8lZlzZYsje/Jr8u/YtlwjjreZrGRmG8KMOzukV3lLmMppXFMvl4bxv6YFEmIuTsOhbLTwFgh7KYNjodLj/LsqRVfwz31PgWQFTEPICV7GCvgVlPRxnofqKSjgTWI4mxDhBpVcATvaoBl1L/6WLbFvBsoAUBItWwctO2xalKxF5szhGm8lccoc5MZr8kfE0uxMgsxz4er68iCID+rsCAQM=",
			"ServerEntrySignaturePublicKey" : "sHuUVTWaRyh5pZwy4UguSgkwmBe0EHtJJkoF5WrxmvA=",

			"TargetApiProtocol" : "ssh"
		}`),
	}
	ctx, cancel := context.WithCancel(context.Background())
	dialer := GetSingletonDialer()
	dialer.Start(ctx, cfg)
	cancel()
	require.NotNil(t, dialer.tunnel)
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
	require.ErrorIs(t, err, context.Canceled)
}

type errorTunnel struct {
	err     error
	stopped bool
}

func (t *errorTunnel) Dial(addr string) (net.Conn, error) {
	return nil, t.err
}

func (t *errorTunnel) Stop() {
	t.stopped = true
}

func TestDialer_DialStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dialer := GetSingletonDialer()

	// Dial before Start.
	_, err := dialer.DialStream(ctx, "")
	require.ErrorIs(t, err, errNotStartedDial)

	var tunnel errorTunnel
	startTunnel = func(ctx context.Context, config *DialerConfig) (psiphonTunnel, error) {
		tunnel.stopped = false
		return &tunnel, nil
	}
	defer func() {
		startTunnel = psiphonStartTunnel
	}()
	// Make sure it works on restarts.
	for i := 0; i < 2; i++ {
		// Dial after Start.
		require.NoError(t, dialer.Start(ctx, nil))
		require.False(t, tunnel.stopped)
		conn, err := dialer.DialStream(ctx, "")
		require.NoError(t, err)
		require.NoError(t, conn.CloseRead())
		require.NoError(t, conn.CloseWrite())

		// Dial after Stop.
		require.NoError(t, dialer.Stop())
		require.True(t, tunnel.stopped)
		_, err = dialer.DialStream(nil, "")
		require.ErrorIs(t, err, errNotStartedDial)
	}
}

func TestDialer_DialStream_Error(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dialer := GetSingletonDialer()
	tunnel := errorTunnel{
		err: errors.New("failed to dial"),
	}
	startTunnel = func(ctx context.Context, config *DialerConfig) (psiphonTunnel, error) {
		tunnel.stopped = false
		return &tunnel, nil
	}
	defer func() {
		startTunnel = psiphonStartTunnel
	}()
	require.NoError(t, dialer.Start(ctx, nil))
	require.False(t, tunnel.stopped)
	_, err := dialer.DialStream(ctx, "")
	require.Equal(t, tunnel.err, err)
	require.NoError(t, dialer.Stop())
}

func TestDialer_Stop_NotStarted(t *testing.T) {
	err := GetSingletonDialer().Stop()
	require.ErrorIs(t, err, errNotStartedStop)
}
