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

package config

import (
	"net/url"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/socks5"
)

func wrapStreamDialerWithSOCKS5(innerSD func() (transport.StreamDialer, error), _ func() (transport.PacketDialer, error), configURL *url.URL) (transport.StreamDialer, error) {
	sd, err := innerSD()
	if err != nil {
		return nil, err
	}
	endpoint := transport.StreamDialerEndpoint{Dialer: sd, Address: configURL.Host}
	dialer, err := socks5.NewStreamDialer(&endpoint)
	if err != nil {
		return nil, err
	}
	userInfo := configURL.User
	if userInfo != nil {
		username := userInfo.Username()
		password, _ := userInfo.Password()
		err := dialer.SetCredentials([]byte(username), []byte(password))
		if err != nil {
			return nil, err
		}
	}
	return dialer, nil
}
