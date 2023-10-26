// Copyright 2023 Jigsaw Operations LLC
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

package tls

import (
	"context"
	"testing"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/require"
)

func TestDomain(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPStreamDialer{})
	require.NoError(t, err)
	conn, err := sd.Dial(context.Background(), "dns.google:443")
	require.NoError(t, err)
	conn.Close()
}

func TestIP(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPStreamDialer{})
	require.NoError(t, err)
	conn, err := sd.Dial(context.Background(), "8.8.8.8:443")
	require.NoError(t, err)
	conn.Close()
}

func TestIPOverride(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPStreamDialer{}, WithCertificateName("8.8.8.8"))
	require.NoError(t, err)
	conn, err := sd.Dial(context.Background(), "dns.google:443")
	require.NoError(t, err)
	conn.Close()
}

func TestFakeSNI(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPStreamDialer{}, WithSNI("decoy.example.com"))
	require.NoError(t, err)
	conn, err := sd.Dial(context.Background(), "dns.google:443")
	require.NoError(t, err)
	conn.Close()
}

func TestNoSNI(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPStreamDialer{}, WithSNI(""))
	require.NoError(t, err)
	conn, err := sd.Dial(context.Background(), "dns.google:443")
	require.NoError(t, err)
	conn.Close()
}

func TestAllCustom(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPStreamDialer{}, WithSNI("decoy.android.com"), WithCertificateName("www.youtube.com"))
	require.NoError(t, err)
	conn, err := sd.Dial(context.Background(), "www.google.com:443")
	require.NoError(t, err)
	conn.Close()
}

func TestWithSNI(t *testing.T) {
	var cfg ClientConfig
	WithSNI("example.com")("", 0, &cfg)
	require.Equal(t, "example.com", cfg.ServerName)
}

func TestWithALPN(t *testing.T) {
	var cfg ClientConfig
	WithALPN([]string{"h2", "http/1.1"})("", 0, &cfg)
	require.Equal(t, []string{"h2", "http/1.1"}, cfg.NextProtos)
}
