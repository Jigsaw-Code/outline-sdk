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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

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
	sd    transport.StreamDialer
	pp    *outlinePacketProxy
	svrIP net.IP
}

var configToDialer = config.NewDefaultConfigToDialer()

func NewOutlineDevice(transportConfig string) (od *OutlineDevice, err error) {
	if err := validateConfig(transportConfig); err != nil {
		return nil, err
	}
	parsed, err := url.Parse(transportConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config URL: %w", err)
	}
	transportConfig, err = formatConfig(parsed)
	if err != nil {
		return nil, err
	}
	ip, err := resolveShadowsocksServerIPFromHostname(parsed.Hostname())
	if err != nil {
		return nil, err
	}
	od = &OutlineDevice{
		svrIP: ip,
	}

	if od.sd, err = configToDialer.NewStreamDialer(transportConfig); err != nil {
		return nil, fmt.Errorf("failed to create TCP dialer: %w", err)
	}
	if od.pp, err = newOutlinePacketProxy(transportConfig); err != nil {
		return nil, fmt.Errorf("failed to create delegate UDP proxy: %w", err)
	}
	if od.IPDevice, err = lwip2transport.ConfigureDevice(od.sd, od.pp); err != nil {
		return nil, fmt.Errorf("failed to configure lwIP: %w", err)
	}

	return
}

func (d *OutlineDevice) Close() error {
	return d.IPDevice.Close()
}

func (d *OutlineDevice) Refresh() error {
	return d.pp.testConnectivityAndRefresh(connectivityTestResolver, connectivityTestDomain)
}

func (d *OutlineDevice) GetServerIP() net.IP {
	return d.svrIP
}

// based on ssconf spec: https://reddit.com/r/outlinevpn/w/index/dynamic_access_keys
func constructShadowsocksSessionConfig(resp []byte) (string, error) {
	var cfg struct {
		Server     string `json:"server"`
		ServerPort string `json:"server_port"`
		Password   string `json:"password"`
		Method     string `json:"method"`
		Prefix     string `json:"prefix,omitempty"` // optional
	}

	if err := json.Unmarshal([]byte(resp), &cfg); err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %w", err)
	}

	// TODO: unsure what to do with prefix field
	var cfgURL url.URL
	cfgURL.Scheme = "ss"
	cfgURL.User = url.User(base64.StdEncoding.EncodeToString([]byte(cfg.Method + ":" + cfg.Password)))
	cfgURL.Host = cfg.Server + ":" + cfg.ServerPort

	return cfgURL.String(), nil
}

func fetchShadowsocksSessionConfig(transportConfig string) ([]byte, error) {
	transportConfig = "https" + strings.TrimPrefix(transportConfig, "ssconf")

	resp, err := http.Get(transportConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config from ssconf: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to ready body from request: %w", err)
	}
	defer resp.Body.Close()

	return body, nil
}

func formatConfig(transportConfigURL *url.URL) (string, error) {
	switch transportConfigURL.Scheme {
	case "ss":
		return transportConfigURL.String(), nil
	case "ssconf":
		fetched, err := fetchShadowsocksSessionConfig(transportConfigURL.String())
		if err != nil {
			return "", err
		}
		return constructShadowsocksSessionConfig(fetched)
	default:
		return "", errors.New("config must start with 'ss://' or 'ssconf://'")
	}
}

func validateConfig(transportConfig string) error {
	if strings.Contains(transportConfig, "|") {
		return errors.New("multi-part config is not supported")
	}
	if transportConfig = strings.TrimSpace(transportConfig); transportConfig == "" {
		return errors.New("config is required")
	}
	return nil
}

func resolveShadowsocksServerIPFromHostname(hostname string) (net.IP, error) {
	ipList, err := net.LookupIP(hostname)
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
