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

package outline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/Jigsaw-Code/outline-sdk/network"
	"github.com/Jigsaw-Code/outline-sdk/network/dnstruncate"
	"github.com/Jigsaw-Code/outline-sdk/network/lwip2transport"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/shadowsocks"
	"github.com/Jigsaw-Code/outline-sdk/x/connectivity"
)

const (
	connectivityTestDomain   = "www.google.com"
	connectivityTestResolver = "1.1.1.1:53"
)

type OutlineConfig struct {
	Hostname string
	Port     uint16
	Password string
	Cipher   string
}

type OutlineDevice struct {
	t2s              network.IPDevice
	pktProxy         network.DelegatePacketProxy
	fallbackPktProxy network.PacketProxy
	ssStreamDialer   transport.StreamDialer
	ssPktListener    transport.PacketListener
	ssPktProxy       network.PacketProxy
}

func NewOutlineDevice(accessKey string) (*OutlineDevice, error) {
	var d OutlineDevice

	config, err := ParseAccessKey(accessKey)
	if err != nil {
		return nil, fmt.Errorf("invalid outline access key: %w", err)
	}

	cipher, err := shadowsocks.NewEncryptionKey(config.Cipher, config.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher `%v`: %w", config.Cipher, err)
	}

	ssAddress := net.JoinHostPort(config.Hostname, strconv.Itoa(config.Port))

	// Create Shadowsocks TCP StreamDialer
	d.ssStreamDialer, err = shadowsocks.NewStreamDialer(&transport.TCPEndpoint{Address: ssAddress}, cipher)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP dialer: %w", err)
	}

	// Create DNS Truncated PacketProxy
	d.fallbackPktProxy, err = dnstruncate.NewPacketProxy()
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS truncate proxy: %w", err)
	}

	// Create Shadowsocks UDP PacketProxy
	d.ssPktListener, err = shadowsocks.NewPacketListener(&transport.UDPEndpoint{Address: ssAddress}, cipher)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP listener: %w", err)
	}

	d.ssPktProxy, err = network.NewPacketProxyFromPacketListener(d.ssPktListener)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP proxy: %w", err)
	}

	// Create DelegatePacketProxy
	d.pktProxy, err = network.NewDelegatePacketProxy(d.fallbackPktProxy)
	if err != nil {
		return nil, fmt.Errorf("failed to create delegate UDP proxy: %w", err)
	}

	// Configure lwIP Device
	d.t2s, err = lwip2transport.ConfigureDevice(d.ssStreamDialer, d.pktProxy)
	if err != nil {
		return nil, fmt.Errorf("failed to configure lwIP: %w", err)
	}

	return &d, nil
}

func (d *OutlineDevice) Close() error {
	return d.t2s.Close()
}

func (d *OutlineDevice) Refresh() error {
	streamResolver := &transport.StreamDialerEndpoint{Dialer: d.ssStreamDialer, Address: connectivityTestResolver}
	_, err := connectivity.TestResolverStreamConnectivity(context.Background(), streamResolver, connectivityTestDomain)
	if err != nil {
		return fmt.Errorf("failed to connect to the remote Shadowsocks server: %w", err)
	}

	dialer := transport.PacketListenerDialer{Listener: d.ssPktListener}
	packetResolver := &transport.PacketDialerEndpoint{Dialer: dialer, Address: connectivityTestResolver}
	_, err = connectivity.TestResolverPacketConnectivity(context.Background(), packetResolver, connectivityTestDomain)

	if err != nil {
		return d.pktProxy.SetProxy(d.fallbackPktProxy)
	} else {
		return d.pktProxy.SetProxy(d.ssPktProxy)
	}
}

func (d *OutlineDevice) RelayTraffic(netDev io.ReadWriter) error {
	var err1, err2 error

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		_, err2 = io.Copy(d.t2s, netDev)
	}()

	_, err1 = io.Copy(netDev, d.t2s)

	wg.Wait()
	return errors.Join(err1, err2)
}
