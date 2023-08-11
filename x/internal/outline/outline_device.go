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
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/Jigsaw-Code/outline-sdk/network"
	"github.com/Jigsaw-Code/outline-sdk/network/lwip2transport"
	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type OutlineDevice struct {
	t2s network.IPDevice
	pp  *outlinePacketProxy
	sd  transport.StreamDialer
}

func NewOutlineClientDevice(accessKey string) (d *OutlineDevice, err error) {
	d = &OutlineDevice{}

	d.sd, err = NewOutlineStreamDialer(accessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP dialer: %w", err)
	}

	d.pp, err = newOutlinePacketProxy(accessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP proxy: %w", err)
	}

	d.t2s, err = lwip2transport.ConfigureDevice(d.sd, d.pp)
	if err != nil {
		return nil, fmt.Errorf("failed to configure lwIP: %w", err)
	}

	return
}

func (d *OutlineDevice) Close() error {
	return d.t2s.Close()
}

func (d *OutlineDevice) Refresh() error {
	return d.pp.testConnectivityAndRefresh("1.1.1.1:53", "www.google.com")
}

// RelayTraffic copies all traffic between an IPDevice (`netDev`) and the OutlineDevice (`d`) in both directions.
// It will not return until both devices have been closed or any error occur. Therefore, the caller must call this
// function in a goroutine and make sure to close both devices (`netDev` and `d`) asynchronously.
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
