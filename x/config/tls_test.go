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

package config

import (
	"net/url"
	"testing"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/tls"
	"github.com/stretchr/testify/require"
)

func TestTLS(t *testing.T) {
	tlsURL, err := url.Parse("tls")
	require.NoError(t, err)
	_, err = wrapStreamDialerWithTLS(&transport.TCPDialer{}, tlsURL)
	require.NoError(t, err)
}

func TestTLS_SNI(t *testing.T) {
	tlsURL, err := url.Parse("tls:sni=www.google.com")
	require.NoError(t, err)
	options, err := parseOptions(tlsURL)
	require.NoError(t, err)
	cfg := tls.ClientConfig{ServerName: "host", CertificateName: "host"}
	for _, option := range options {
		option("host", &cfg)
	}
	require.Equal(t, "www.google.com", cfg.ServerName)
	require.Equal(t, "host", cfg.CertificateName)
}

func TestTLS_NoSNI(t *testing.T) {
	tlsURL, err := url.Parse("tls:sni=")
	require.NoError(t, err)
	options, err := parseOptions(tlsURL)
	require.NoError(t, err)
	cfg := tls.ClientConfig{ServerName: "host", CertificateName: "host"}
	for _, option := range options {
		option("host", &cfg)
	}
	require.Equal(t, "", cfg.ServerName)
	require.Equal(t, "host", cfg.CertificateName)
}

func TestTLS_MultipleSNI(t *testing.T) {
	tlsURL, err := url.Parse("tls:sni=www.google.com&sni=second")
	require.NoError(t, err)
	_, err = parseOptions(tlsURL)
	require.Error(t, err)
}

func TestTLS_CertName(t *testing.T) {
	tlsURL, err := url.Parse("tls:certname=www.google.com")
	require.NoError(t, err)
	options, err := parseOptions(tlsURL)
	require.NoError(t, err)
	cfg := tls.ClientConfig{ServerName: "host", CertificateName: "host"}
	for _, option := range options {
		option("host", &cfg)
	}
	require.Equal(t, "host", cfg.ServerName)
	require.Equal(t, "www.google.com", cfg.CertificateName)
}

func TestTLS_Combined(t *testing.T) {
	tlsURL, err := url.Parse("tls:SNI=sni.example.com&CertName=certname.example.com")
	require.NoError(t, err)
	options, err := parseOptions(tlsURL)
	require.NoError(t, err)
	cfg := tls.ClientConfig{ServerName: "host", CertificateName: "host"}
	for _, option := range options {
		option("host", &cfg)
	}
	require.Equal(t, "sni.example.com", cfg.ServerName)
	require.Equal(t, "certname.example.com", cfg.CertificateName)
}

func TestTLS_UnsupportedOption(t *testing.T) {
	tlsURL, err := url.Parse("tls:unsupported")
	require.NoError(t, err)
	_, err = parseOptions(tlsURL)
	require.Error(t, err)
}
