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
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"golang.org/x/net/websocket"
)

type wsConfig struct {
	tcp_path string
	udp_path string
}

func parseWSConfig(configURL *url.URL) (*wsConfig, error) {
	query := configURL.Opaque
	values, err := url.ParseQuery(query)
	if err != nil {
		return nil, err
	}
	var cfg wsConfig
	for key, values := range values {
		switch strings.ToLower(key) {
		case "tcp_path":
			if len(values) != 1 {
				return nil, fmt.Errorf("udp_path option must has one value, found %v", len(values))
			}
			cfg.tcp_path = values[0]
		case "udp_path":
			if len(values) != 1 {
				return nil, fmt.Errorf("tcp_path option must has one value, found %v", len(values))
			}
			cfg.udp_path = values[0]
		default:
			return nil, fmt.Errorf("unsupported option %v", key)
		}
	}
	return &cfg, nil
}

// wsToStreamConn converts a [websocket.Conn] to a [transport.StreamConn].
type wsToStreamConn struct {
	*websocket.Conn
}

func (c *wsToStreamConn) CloseRead() error {
	// Nothing to do.
	return nil
}

func (c *wsToStreamConn) CloseWrite() error {
	return c.Close()
}

func wrapStreamDialerWithWebsocket(innerSD func() (transport.StreamDialer, error), _ func() (transport.PacketDialer, error), configURL *url.URL) (transport.StreamDialer, error) {
	sd, err := innerSD()
	if err != nil {
		return nil, err
	}
	config, err := parseWSConfig(configURL)
	if err != nil {
		return nil, err
	}
	if config.tcp_path == "" {
		return nil, errors.New("must specify tcp_path")
	}
	return transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
		tcpConn, err := sd.DialStream(ctx, addr)
		if err != nil {
			return nil, err
		}
		wsURL := url.URL{Scheme: "ws", Host: addr, Path: config.tcp_path}
		origin := url.URL{Scheme: "http", Host: addr}
		wsCfg, err := websocket.NewConfig(wsURL.String(), origin.String())
		if err != nil {
			return nil, err
		}
		wsConn, err := websocket.NewClient(wsCfg, tcpConn)
		// TODO: wrap wsConn
		return &wsToStreamConn{wsConn}, err
	}), nil
}

func wrapPacketDialerWithWebsocket(innerSD func() (transport.StreamDialer, error), _ func() (transport.PacketDialer, error), configURL *url.URL) (transport.PacketDialer, error) {
	sd, err := innerSD()
	if err != nil {
		return nil, err
	}
	config, err := parseWSConfig(configURL)
	if err != nil {
		return nil, err
	}
	if config.udp_path == "" {
		return nil, errors.New("must specify udp_path")
	}
	return transport.FuncPacketDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		tcpConn, err := sd.DialStream(ctx, addr)
		if err != nil {
			return nil, err
		}
		url := url.URL{Scheme: "http", Host: addr, Path: config.udp_path}
		wsCfg, err := websocket.NewConfig(url.String(), "")
		if err != nil {
			return nil, err
		}
		wsConn, err := websocket.NewClient(wsCfg, tcpConn)
		return wsConn, err
	}), nil
}
