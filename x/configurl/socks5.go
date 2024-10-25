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

package configurl

import (
	"context"
	"net"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/socks5"
)

func newSOCKS5StreamDialerFactory(newSD NewStreamDialerFunc) NewStreamDialerFunc {
	return func(ctx context.Context, config *Config) (transport.StreamDialer, error) {
		return newSOCKS5Client(ctx, *config, newSD)
	}
}

func newSOCKS5PacketDialerFactory(newSD NewStreamDialerFunc, newPD NewPacketDialerFunc) NewPacketDialerFunc {
	return func(ctx context.Context, config *Config) (transport.PacketDialer, error) {
		client, err := newSOCKS5Client(ctx, *config, newSD)
		if err != nil {
			return nil, err
		}
		pd, err := newPD(ctx, config.BaseConfig)
		if err != nil {
			return nil, err
		}
		client.EnablePacket(pd)
		return transport.PacketListenerDialer{Listener: client}, nil
	}
}

func newSOCKS5PacketConnFactory(newSD NewStreamDialerFunc, newPD NewPacketDialerFunc) NewPacketConnFunc {
	return func(ctx context.Context, config *Config) (net.PacketConn, error) {
		client, err := newSOCKS5Client(ctx, *config, newSD)
		if err != nil {
			return nil, err
		}
		pd, err := newPD(ctx, config.BaseConfig)
		if err != nil {
			return nil, err
		}
		client.EnablePacket(pd)
		return client.ListenPacket(ctx)
	}
}

func newSOCKS5Client(ctx context.Context, config Config, newSD NewStreamDialerFunc) (*socks5.Client, error) {
	sd, err := newSD(ctx, config.BaseConfig)
	if err != nil {
		return nil, err
	}
	endpoint := transport.StreamDialerEndpoint{Dialer: sd, Address: config.URL.Host}
	client, err := socks5.NewClient(&endpoint)
	if err != nil {
		return nil, err
	}
	userInfo := config.URL.User
	if userInfo != nil {
		username := userInfo.Username()
		password, _ := userInfo.Password()
		err := client.SetCredentials([]byte(username), []byte(password))
		if err != nil {
			return nil, err
		}
	}
	return client, nil
}
