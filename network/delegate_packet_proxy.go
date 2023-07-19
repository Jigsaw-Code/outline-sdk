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

package network

import (
	"errors"
	"sync/atomic"
)

// DelegatePacketProxy is a PacketProxy that forwards calls (like NewSession) to another PacketProxy. To create a
// DelegatePacketProxy with the default PacketProxy, use NewDelegatePacketProxy. To change the underlying PacketProxy,
// use SetProxy.
//
// Note: After changing the underlying PacketProxy, only new NewSession calls will be routed to the new PacketProxy.
// Existing sessions will not be affected.
//
// Multiple goroutines may invoke methods on a DelegatePacketProxy simultaneously.
type DelegatePacketProxy interface {
	PacketProxy

	// SetProxy updates the underlying PacketProxy to `proxy`. And `proxy` must not be nil. After this function
	// returns, all new PacketProxy calls will be forwarded to the `proxy`. Existing sessions will not be affected.
	SetProxy(proxy PacketProxy) error
}

var errInvalidProxy = errors.New("the underlying proxy must not be nil")

// Compilation guard against interface implementation
var _ DelegatePacketProxy = (*delegatePacketProxy)(nil)

type delegatePacketProxy struct {
	proxy atomic.Value
}

// NewDelegatePacketProxy creates a new [DelegatePacketProxy] that forwards calls to the `proxy` [PacketProxy].
// The `proxy` must not be nil.
func NewDelegatePacketProxy(proxy PacketProxy) (DelegatePacketProxy, error) {
	if proxy == nil {
		return nil, errInvalidProxy
	}
	dp := delegatePacketProxy{}
	dp.proxy.Store(proxy)
	return &dp, nil
}

// NewSession implements PacketProxy.NewSession, and it will forward the call to the underlying PacketProxy.
func (p *delegatePacketProxy) NewSession(respWriter PacketResponseReceiver) (PacketRequestSender, error) {
	return p.proxy.Load().(PacketProxy).NewSession(respWriter)
}

// SetProxy implements DelegatePacketProxy.SetProxy.
func (p *delegatePacketProxy) SetProxy(proxy PacketProxy) error {
	if proxy == nil {
		return errInvalidProxy
	}
	p.proxy.Store(proxy)
	return nil
}
