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
	"fmt"

	"github.com/Jigsaw-Code/outline-sdk/network"
	"github.com/Jigsaw-Code/outline-sdk/network/dnstruncate"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/connectivity"
)

type outlinePacketProxy struct {
	network.DelegatePacketProxy

	remotePktListener transport.PacketListener // this will be used in connectivity test
	remote, fallback  network.PacketProxy
}

func newOutlinePacketProxy(accessKey string) (opp *outlinePacketProxy, err error) {
	proxy := outlinePacketProxy{}

	proxy.fallback, err = dnstruncate.NewPacketProxy()
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS truncate proxy: %w", err)
	}

	// Create Shadowsocks UDP PacketProxy
	proxy.remotePktListener, err = NewOutlinePacketListener(accessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP listener: %w", err)
	}

	proxy.remote, err = network.NewPacketProxyFromPacketListener(proxy.remotePktListener)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP proxy: %w", err)
	}

	// Create DelegatePacketProxy
	proxy.DelegatePacketProxy, err = network.NewDelegatePacketProxy(proxy.fallback)
	if err != nil {
		return nil, fmt.Errorf("failed to create delegate UDP proxy: %w", err)
	}

	return &proxy, nil
}

func (proxy *outlinePacketProxy) testConnectivityAndRefresh(resolver, domain string) error {
	dialer := transport.PacketListenerDialer{Listener: proxy.remotePktListener}
	dnsResolver := &transport.PacketDialerEndpoint{Dialer: dialer, Address: resolver}
	_, err := connectivity.TestResolverPacketConnectivity(context.Background(), dnsResolver, domain)

	if err != nil {
		return proxy.SetProxy(proxy.fallback)
	} else {
		return proxy.SetProxy(proxy.remote)
	}
}
