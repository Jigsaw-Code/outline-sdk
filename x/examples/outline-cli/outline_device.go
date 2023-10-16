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
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"

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
	t2s   network.IPDevice
	sd    transport.StreamDialer
	pp    *outlinePacketProxy
	svrIP net.IP
}

func NewOutlineDevice(transportConfig string) (od *OutlineDevice, err error) {
	ip, err := resolveShadowsocksServerIPFromConfig(transportConfig)
	if err != nil {
		return nil, err
	}
	od = &OutlineDevice{
		svrIP: ip,
	}

	if od.sd, err = config.NewStreamDialer(transportConfig); err != nil {
		return nil, fmt.Errorf("failed to create TCP dialer: %w", err)
	}
	if od.pp, err = newOutlinePacketProxy(transportConfig); err != nil {
		return nil, fmt.Errorf("failed to create delegate UDP proxy: %w", err)
	}
	if od.t2s, err = lwip2transport.ConfigureDevice(od.sd, od.pp); err != nil {
		return nil, fmt.Errorf("failed to configure lwIP: %w", err)
	}

	return
}

func (d *OutlineDevice) Close() error {
	return d.t2s.Close()
}

func (d *OutlineDevice) Refresh() error {
	return d.pp.testConnectivityAndRefresh(connectivityTestResolver, connectivityTestDomain)
}

func (d *OutlineDevice) GetServerIP() net.IP {
	return d.svrIP
}

func (d *OutlineDevice) RelayTraffic(netDev io.ReadWriter) error {
	var err1, err2 error

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		fmt.Println("debug: OutlineDevice start receiving data from tun")
		if _, err2 = io.Copy(d.t2s, netDev); err2 != nil {
			fmt.Printf("warning: failed to write data to OutlineDevice: %v\n", err2)
		} else {
			fmt.Println("debug: tun -> OutlineDevice eof")
		}
	}()

	fmt.Println("debug: start forwarding OutlineDevice data to tun")
	if _, err1 = io.Copy(netDev, d.t2s); err1 != nil {
		fmt.Printf("warning: failed to forward OutlineDevice data to tun: %v\n", err1)
	} else {
		fmt.Println("debug: OutlineDevice -> tun eof")
	}

	wg.Wait()

	return errors.Join(err1, err2)
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
