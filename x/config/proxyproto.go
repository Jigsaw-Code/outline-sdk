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
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	proxyproto "github.com/pires/go-proxyproto"
)

const defaultProxyProtoVersion = 2

type proxyProtoConfig struct {
	version byte
}

func parseProxyProtoConfig(configURL *url.URL) (*proxyProtoConfig, error) {
	query := configURL.Opaque
	values, err := url.ParseQuery(query)
	if err != nil {
		return nil, err
	}
	cfg := proxyProtoConfig{version: defaultProxyProtoVersion}
	for key, values := range values {
		switch strings.ToLower(key) {
		case "version":
			if len(values) != 1 {
				return nil, fmt.Errorf("version option must has one value, found %v", len(values))
			}
			version, err := strconv.ParseInt(values[0], 10, 8)
			if err != nil {
				return nil, fmt.Errorf("version must be a number")
			}
			cfg.version = byte(version)
		default:
			return nil, fmt.Errorf("unsupported option %v", key)
		}
	}
	return &cfg, nil
}

func wrapStreamDialerWithProxyProto(innerSD func() (transport.StreamDialer, error), _ func() (transport.PacketDialer, error), configURL *url.URL) (transport.StreamDialer, error) {
	sd, err := innerSD()
	if err != nil {
		return nil, err
	}
	config, err := parseProxyProtoConfig(configURL)
	if err != nil {
		return nil, err
	}
	return transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
		conn, err := sd.DialStream(ctx, addr)
		if err != nil {
			return nil, err
		}
		header := proxyproto.HeaderProxyFromAddrs(config.version, conn.LocalAddr(), conn.RemoteAddr())
		if _, err = header.WriteTo(conn); err != nil {
			return nil, fmt.Errorf("failed to write PROXY protocol header: %w", err)
		}
		return conn, nil
	}), nil
}

func wrapPacketDialerWithProxyProto(_ func() (transport.StreamDialer, error), innerPD func() (transport.PacketDialer, error), configURL *url.URL) (transport.PacketDialer, error) {
	pd, err := innerPD()
	if err != nil {
		return nil, err
	}
	config, err := parseProxyProtoConfig(configURL)
	if err != nil {
		return nil, err
	}
	return transport.FuncPacketDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		conn, err := pd.DialPacket(ctx, addr)
		if err != nil {
			return nil, err
		}
		header := proxyproto.HeaderProxyFromAddrs(config.version, conn.LocalAddr(), conn.RemoteAddr())
		if _, err = header.WriteTo(conn); err != nil {
			return nil, fmt.Errorf("failed to write PROXY protocol header: %w", err)
		}
		return conn, nil
	}), nil
}
