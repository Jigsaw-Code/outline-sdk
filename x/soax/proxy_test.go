// Copyright 2025 The Outline Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package soax

import (
	"context"
	"errors"
	"net"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/things-go/go-socks5"
)

func TestSessionConfig_newUsername_AllFields(t *testing.T) {
	config := SessionConfig{
		PackageID:     123456,
		PackageKey:    "my_package_key",
		CountryCode:   "us",
		RegionID:      "new york",
		CityID:        "new york city",
		ISPID:         "verizon",
		SessionID:     "my_session",
		SessionLength: 10 * time.Minute,
		IdleTTL:       5 * time.Minute,
	}
	userinfo := config.newUserPassword()
	require.Equal(t,
		"package-123456-country-us-region-new%20york-city-new%20york%20city-isp-verizon-sessionid-my_session-sessionlength-600-idlettl-300:my_package_key",
		userinfo.String())
	require.Equal(t, "package-123456-country-us-region-new york-city-new york city-isp-verizon-sessionid-my_session-sessionlength-600-idlettl-300", userinfo.Username())
	password, isSet := userinfo.Password()
	require.True(t, isSet)
	require.Equal(t, "my_package_key", password)
}

func TestSessionConfig_newUsername_PackageOnly(t *testing.T) {
	config := SessionConfig{
		PackageID:  123456,
		PackageKey: "my_package_key",
	}
	require.Equal(t, "package-123456:my_package_key", config.newUserPassword().String())
}

func TestSessionConfig_newSession_SetsDefaults(t *testing.T) {
	config := SessionConfig{
		PackageID:  123456,
		PackageKey: "my_package_key",
	}
	session := config.NewSession()
	require.NotNil(t, session)
	require.NotContains(t, session.config.SessionID, "-")
	require.NotEmpty(t, session.config.SessionID)
	require.Equal(t, 1*time.Hour, session.config.SessionLength)
	require.Equal(t, 1*time.Hour, session.config.IdleTTL)
	require.Equal(t, proxyAddress, session.config.Endpoint)
}

type FuncCredentialStore func(user, password, userAddr string) bool

func (f FuncCredentialStore) Valid(user, password, userAddr string) bool {
	return f(user, password, userAddr)
}

var _ socks5.CredentialStore = FuncCredentialStore(nil)

func TestSession_NewSOCKS5Client(t *testing.T) {
	config := SessionConfig{
		PackageID:  123456,
		PackageKey: "my_package_key",
	}
	var userinfo url.Userinfo

	// Create a SOCKS5 server that expects the credentials from the config.
	cator := socks5.UserPassAuthenticator{
		Credentials: FuncCredentialStore(func(userToCheck, passwordToCheck, userAddr string) bool {
			require.Equal(t, userinfo.Username(), userToCheck)
			expectedPassword, _ := userinfo.Password()
			require.Equal(t, expectedPassword, passwordToCheck)
			return true
		}),
	}
	server := socks5.NewServer(
		socks5.WithAuthMethods([]socks5.Authenticator{cator}),
	)

	var running sync.WaitGroup
	// Create SOCKS5 proxy on localhost with a random port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	running.Add(1)
	go func() {
		defer running.Done()
		err := server.Serve(listener)
		// The server will be stopped by closing the listener.
		if err != nil && !errors.Is(err, net.ErrClosed) {
			require.NoError(t, err)
		}
	}()
	defer func() {
		listener.Close()
		running.Wait()
	}()

	// The SOCKS5 server from things-go/go-socks5 will try to connect to this address.
	// We need a listener, otherwise the Dial will fail with connection refused.
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer targetListener.Close()

	// Set the endpoint to our test server.
	config.Endpoint = listener.Addr().String()
	session := config.NewSession()
	userinfo = *session.config.newUserPassword()
	client, err := session.NewSOCKS5Client()
	require.NoError(t, err)
	require.NotNil(t, client)

	_, err = client.DialStream(context.Background(), targetListener.Addr().String())
	require.NoError(t, err)
}
