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

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/network"
	"github.com/Jigsaw-Code/outline-sdk/network/lwip2transport"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
)

const (
	connectivityTestDomain   = "www.google.com"
	connectivityTestResolver = "1.1.1.1:53"
)

type OutlineDevice struct {
	network.IPDevice
	sd transport.StreamDialer
	pp *outlinePacketProxy
}

var configParser = config.NewDefaultConfigParser()

func supportsHappyEyeballs(dialer transport.StreamDialer) bool {
	// Some proxy protocols, most notably Shadowsocks, can't communicate connection success.
	// Our shadowsocks.StreamDialer will return a connection successfully as long as it can
	// connect to the proxy server, regardless of whether it can connect to the target.
	// This breaks HappyEyeballs.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	conn, err := dialer.DialStream(ctx, "invalid:0")
	cancel()
	if conn != nil {
		conn.Close()
	}
	// If the dialer returns success on an invalid address, it doesn't support Happy Eyeballs.
	return err != nil
}

func getProxyHost(transportConfig string) (string, error) {
	var proxyAddress string
	innerDialer := transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
		proxyAddress = addr
		return nil, errors.New("not implemented")
	})
	dialer, err := configParser.WrapStreamDialer(innerDialer, transportConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create TCP dialer: %w", err)
	}
	dialer.DialStream(context.Background(), "")
	if proxyAddress == "" {
		return "", errors.New("Could not determine proxy address")
	}
	proxyHost, _, err := net.SplitHostPort(proxyAddress)
	if err != nil {
		return "", errors.New("Could not extract proxy host")
	}
	return proxyHost, nil
}

func newBypassStreamDialer(proxyHost string) {
	addr4, err := net.Dial("udp4", proxyHost)
	addr6, err := net.Dial("udp6", proxyHost)

	return transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
		ip, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		netip.ParseAddr(ip)

	})
}

type Dialer interface {
	transport.StreamDialer
	transport.PacketDialer
}

func NewOutlineDevice(baseDialer Dialer, transportConfig string) (od *OutlineDevice, err error) {
	proxyAddress, err := getFirstHop(transportConfig)
	if err != nil {
		return nil, err
	}
	od = &OutlineDevice{}

	streamDialer, err := configParser.WrapStreamDialer(&transport.TCPDialer{}, transportConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP dialer: %w", err)
	}
	if !supportsHappyEyeballs(streamDialer) {
		innerDialer := streamDialer
		// Disable IPv6 if the dialer doesn't support HappyEyballs.
		streamDialer = transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
				return nil, fmt.Errorf("IPv6 not supported")
			}
			return innerDialer.DialStream(ctx, addr)
		})
	}
	od.sd = streamDialer

	if od.pp, err = newOutlinePacketProxy(transportConfig); err != nil {
		return nil, fmt.Errorf("failed to create delegate UDP proxy: %w", err)
	}

	if od.IPDevice, err = lwip2transport.ConfigureDevice(od.sd, od.pp); err != nil {
		return nil, fmt.Errorf("failed to configure lwIP: %w", err)
	}

	return od, nil
}

func (d *OutlineDevice) Close() error {
	return d.IPDevice.Close()
}

func (d *OutlineDevice) Refresh() error {
	return d.pp.testConnectivityAndRefresh(connectivityTestResolver, connectivityTestDomain)
}

func resolveShadowsocksServerIPFromConfig(transportConfig string) (net.IP, error) {
	if strings.Contains(transportConfig, "|") {
		return nil, errors.New("multi-part config is not supported")
	}
	if transportConfig = strings.TrimSpace(transportConfig); transportConfig == "" {
		return nil, errors.New("config is required")
	}
	url, err := url.Parse(transportConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if url.Scheme != "ss" {
		return nil, errors.New("config must start with 'ss://'")
	}
	ipList, err := net.LookupIP(url.Hostname())
	if err != nil {
		return nil, fmt.Errorf("invalid server hostname: %w", err)
	}

	// todo: we only tested IPv4 routing table, need to test IPv6 in the future
	for _, ip := range ipList {
		if ip = ip.To4(); ip != nil {
			return ip, nil
		}
	}
	return nil, errors.New("IPv6 only Shadowsocks server is not supported yet")
}
