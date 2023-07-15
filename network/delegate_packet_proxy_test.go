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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// Make sure multiple goroutines can create NewSession and SetProxy concurrently
func TestSetProxyRaceCondition(t *testing.T) {
	const proxiesCnt = 10
	const sessionCntPerProxy = 5

	var proxies [proxiesCnt]*sessionCountPacketProxy
	for i := 0; i < proxiesCnt; i++ {
		proxies[i] = &sessionCountPacketProxy{}
	}

	dp := NewDelegatePacketProxy(proxies[0])

	setProxyTask := &sync.WaitGroup{}
	cancelSetProxy := &atomic.Bool{}
	setProxyTask.Add(1)
	go func() {
		for i := 0; !cancelSetProxy.Load(); i = (i + 1) % proxiesCnt {
			dp.SetProxy(proxies[i])
		}
		setProxyTask.Done()
	}()

	newSessionTask := &sync.WaitGroup{}
	newSessionTask.Add(1)
	go func() {
		for i := 0; i < proxiesCnt*sessionCntPerProxy; i++ {
			dp.NewSession(nil)
		}
		newSessionTask.Done()
	}()

	newSessionTask.Wait()
	cancelSetProxy.Store(true)
	setProxyTask.Wait()

	expectedTotal := proxiesCnt * sessionCntPerProxy
	actualTotal := 0
	for i := 0; i < proxiesCnt; i++ {
		require.GreaterOrEqual(t, proxies[i].Count(), 0)
		// it's very unlikely that all sessions are created in one single proxy
		require.Less(t, proxies[i].Count(), expectedTotal)
		actualTotal += proxies[i].Count()
	}
	require.Equal(t, expectedTotal, actualTotal)
}

// sessionCountPacketProxy logs the count of the NewSession calls, and returns a nil PacketRequestSender
type sessionCountPacketProxy struct {
	cnt atomic.Int32
}

func (sp *sessionCountPacketProxy) NewSession(respWriter PacketResponseReceiver) (PacketRequestSender, error) {
	sp.cnt.Add(1)
	return nil, nil
}

func (sp *sessionCountPacketProxy) Count() int {
	return int(sp.cnt.Load())
}
