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

import "errors"

type DelegatePacketProxy interface {
	PacketProxy
	SetProxy(proxy PacketProxy)
}

// Compilation guard against interface implementation
var _ DelegatePacketProxy = (*delegatePacketProxy)(nil)

type delegatePacketProxy struct {
	proxy PacketProxy
}

func NewDelegatePacketProxy(proxy PacketProxy) DelegatePacketProxy {
	return &delegatePacketProxy{proxy}
}

func (p *delegatePacketProxy) NewSession(respWriter PacketResponseReceiver) (PacketRequestSender, error) {
	realProxy := p.proxy
	if realProxy == nil {
		return nil, errors.New("not supported")
	}
	return realProxy.NewSession(respWriter)
}

func (p *delegatePacketProxy) SetProxy(proxy PacketProxy) {
	p.proxy = proxy
}
