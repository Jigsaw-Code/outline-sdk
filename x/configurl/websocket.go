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

package configurl

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
	tcpPath string
	udpPath string
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
			cfg.tcpPath = values[0]
		case "udp_path":
			if len(values) != 1 {
				return nil, fmt.Errorf("tcp_path option must has one value, found %v", len(values))
			}
			cfg.udpPath = values[0]
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

func wrapStreamDialerWithWebSocket(innerSD func() (transport.StreamDialer, error), _ func() (transport.PacketDialer, error), configURL *url.URL) (transport.StreamDialer, error) {
	sd, err := innerSD()
	if err != nil {
		return nil, err
	}
	config, err := parseWSConfig(configURL)
	if err != nil {
		return nil, err
	}
	if config.tcpPath == "" {
		return nil, errors.New("must specify tcp_path")
	}
	return transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
		wsURL := url.URL{Scheme: "ws", Host: addr, Path: config.tcpPath}
		origin := url.URL{Scheme: "http", Host: addr}
		wsCfg, err := websocket.NewConfig(wsURL.String(), origin.String())
		if err != nil {
			return nil, fmt.Errorf("failed to create websocket config: %w", err)
		}
		baseConn, err := sd.DialStream(ctx, addr)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to websocket endpoint: %w", err)
		}
		wsConn, err := websocket.NewClient(wsCfg, baseConn)
		if err != nil {
			baseConn.Close()
			return nil, fmt.Errorf("failed to create websocket client: %w", err)
		}
		return &wsToStreamConn{wsConn}, nil
	}), nil
}

func wrapPacketDialerWithWebSocket(innerSD func() (transport.StreamDialer, error), _ func() (transport.PacketDialer, error), configURL *url.URL) (transport.PacketDialer, error) {
	sd, err := innerSD()
	if err != nil {
		return nil, err
	}
	config, err := parseWSConfig(configURL)
	if err != nil {
		return nil, err
	}
	if config.udpPath == "" {
		return nil, errors.New("must specify udp_path")
	}
	return transport.FuncPacketDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		wsURL := url.URL{Scheme: "ws", Host: addr, Path: config.udpPath}
		origin := url.URL{Scheme: "http", Host: addr}
		wsCfg, err := websocket.NewConfig(wsURL.String(), origin.String())
		if err != nil {
			return nil, fmt.Errorf("failed to create websocket config: %w", err)
		}
		baseConn, err := sd.DialStream(ctx, addr)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to websocket endpoint: %w", err)
		}
		wsConn, err := websocket.NewClient(wsCfg, baseConn)
		if err != nil {
			baseConn.Close()
			return nil, fmt.Errorf("failed to create websocket client: %w", err)
		}
		return wsConn, nil
	}), nil
}
