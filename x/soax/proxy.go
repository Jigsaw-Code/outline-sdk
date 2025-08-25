// Copyright 2025 The Outline Authors
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

package soax

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/socks5"
	"github.com/Jigsaw-Code/outline-sdk/x/httpconnect"
)

const (
	proxyAddress = "proxy.soax.com:5000"
)

// SessionConfig defines how to connect to the SOAX proxy.
type SessionConfig struct {
	// Credentials
	PackageID  int
	PackageKey string

	// Restrictions
	CountryCode string
	RegionID    string
	CityID      string
	ISPID       string

	// Session management
	SessionID     string
	SessionLength time.Duration
	IdleTTL       time.Duration

	// Endpoint address. If empty, defaults to proxy.soax.com:5000.
	// Useful for testing or reverse proxies.
	Endpoint string
}

// Session represents a session with unique SessionID, created from a SessionConfig.
type Session struct {
	config *SessionConfig
}

func (c *SessionConfig) newUserPassword() *url.Userinfo {
	params := []string{"package", strconv.Itoa(c.PackageID)}
	if c.CountryCode != "" {
		params = append(params, "country", strings.ToLower(c.CountryCode))
	}
	if c.RegionID != "" {
		params = append(params, "region", c.RegionID)
	}
	if c.CityID != "" {
		params = append(params, "city", c.CityID)
	}
	if c.ISPID != "" {
		params = append(params, "isp", c.ISPID)
	}
	if c.SessionID != "" {
		params = append(params, "sessionid", c.SessionID)
	}
	if c.SessionLength > 0 {
		params = append(params, "sessionlength", strconv.Itoa(int(c.SessionLength.Seconds())))
	}
	if c.IdleTTL > 0 {
		params = append(params, "idlettl", strconv.Itoa(int(c.IdleTTL.Seconds())))
	}
	return url.UserPassword(strings.Join(params, "-"), c.PackageKey)
}

func (c *SessionConfig) NewSession() *Session {
	session := new(Session)
	// Copy the config to not modify the original one.
	session.config = c
	if session.config.SessionID == "" {
		session.config.SessionID = strconv.Itoa(int(time.Now().UnixMilli()))
	}
	if session.config.SessionLength == 0 {
		// 1 hour is the maximum session length as per
		// https://helpcenter.soax.com/en/articles/9925415-building-sticky-sessions-connection
		session.config.SessionLength = 1 * time.Hour
	}
	if session.config.IdleTTL == 0 {
		// Ensure IPs won't be released if they become idle during the session as per
		// https://helpcenter.soax.com/en/articles/9939557-understanding-session-parameters#h_ffc53ee8d5
		session.config.IdleTTL = session.config.SessionLength
	}
	if session.config.Endpoint == "" {
		session.config.Endpoint = proxyAddress
	}
	return session
}

// NewSOCKS5Client creates a [socks5.Client] that connects through the SOAX proxy.
func (c *Session) NewSOCKS5Client() (*socks5.Client, error) {
	client, err := socks5.NewClient(&transport.TCPEndpoint{Address: c.config.Endpoint})
	if err != nil {
		return nil, err
	}
	client.EnablePacket(&transport.UDPDialer{})
	userinfo := c.config.newUserPassword()
	password, _ := userinfo.Password()
	client.SetCredentials([]byte(userinfo.Username()), []byte(password))
	return client, nil
}

// NewStreamDialer creates a [transport.StreamDialer] that connects through the SOAX proxy.
// It uses HTTP CONNECT, so it only supports TCP.
func (c *Session) NewWebProxyStreamDialer() (transport.StreamDialer, error) {
	rt, err := httpconnect.NewHTTPProxyTransport(&transport.TCPDialer{}, c.config.Endpoint)
	if err != nil {
		return nil, err
	}
	b64EncodedCredentials := base64.StdEncoding.EncodeToString([]byte(c.config.newUserPassword().String()))
	return httpconnect.NewConnectClient(rt,
		httpconnect.WithHeaders(http.Header{
			"Proxy-Authorization": {fmt.Sprintf("Basic %s", b64EncodedCredentials)},
			// TODO: Add support for Respond-With, as explained in
			// https://helpcenter.soax.com/en/articles/9956908-getting-node-information-with-the-respond-with-header
		}),
	)
}
