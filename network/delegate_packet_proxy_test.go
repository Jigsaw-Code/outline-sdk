// Copyright 2023 The Outline Authors
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

// Make sure the underlying packet proxy can be initialized and updated
func TestProxyCanBeUpdated(t *testing.T) {
	defProxy := &sessionCountPacketProxy{}
	newProxy := &sessionCountPacketProxy{}
	p, err := NewDelegatePacketProxy(defProxy)
	require.NotNil(t, p)
	require.NoError(t, err)

	// Initially no NewSession is called
	require.Exactly(t, 0, defProxy.Count())
	require.Exactly(t, 0, newProxy.Count())

	snd, err := p.NewSession(nil)
	require.Nil(t, snd)
	require.NoError(t, err)

	// defProxy.NewSession's count++
	require.Exactly(t, 1, defProxy.Count())
	require.Exactly(t, 0, newProxy.Count())

	// SetProxy should not call NewSession
	err = p.SetProxy(newProxy)
	require.NoError(t, err)
	require.Exactly(t, 1, defProxy.Count())
	require.Exactly(t, 0, newProxy.Count())

	// newProxy.NewSession's count += 2
	snd, err = p.NewSession(nil)
	require.Nil(t, snd)
	require.NoError(t, err)

	snd, err = p.NewSession(nil)
	require.Nil(t, snd)
	require.NoError(t, err)

	require.Exactly(t, 1, defProxy.Count())
	require.Exactly(t, 2, newProxy.Count())
}

// Make sure multiple goroutines can call NewSession and SetProxy concurrently
// Need to run this test with `-race` flag
func TestSetProxyRaceCondition(t *testing.T) {
	const proxiesCnt = 10
	const sessionCntPerProxy = 5

	var proxies [proxiesCnt]*sessionCountPacketProxy
	for i := 0; i < proxiesCnt; i++ {
		proxies[i] = &sessionCountPacketProxy{}
	}

	dp, err := NewDelegatePacketProxy(proxies[0])
	require.NotNil(t, dp)
	require.NoError(t, err)

	setProxyTask := &sync.WaitGroup{}
	cancelSetProxy := &atomic.Bool{}
	setProxyTask.Add(1)
	go func() {
		for i := 0; !cancelSetProxy.Load(); i = (i + 1) % proxiesCnt {
			err := dp.SetProxy(proxies[i])
			require.NoError(t, err)
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
		actualTotal += proxies[i].Count()
	}
	require.Equal(t, expectedTotal, actualTotal)
}

// Make sure we cannot SetProxy to nil
func TestSetProxyWithNilValue(t *testing.T) {
	// must not initialize with nil
	dp, err := NewDelegatePacketProxy(nil)
	require.Error(t, err)
	require.Nil(t, dp)

	dp, err = NewDelegatePacketProxy(&sessionCountPacketProxy{})
	require.NoError(t, err)
	require.NotNil(t, dp)

	// must not SetProxy to nil
	err = dp.SetProxy(nil)
	require.Error(t, err)
}

// Make sure we can SetProxy to different types
func TestSetProxyOfDifferentTypes(t *testing.T) {
	defProxy := &sessionCountPacketProxy{}
	newProxy := &noopPacketProxy{}

	p, err := NewDelegatePacketProxy(defProxy)
	require.NotNil(t, p)
	require.NoError(t, err)

	// SetProxy should not return error
	err = p.SetProxy(newProxy)
	require.NoError(t, err)

	// NewSession' should not go to defProxy
	snd, err := p.NewSession(nil)
	require.Nil(t, snd)
	require.NoError(t, err)
	require.Exactly(t, 0, defProxy.Count())
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

type noopPacketProxy struct {
}

func (noopPacketProxy) NewSession(PacketResponseReceiver) (PacketRequestSender, error) {
	return nil, nil
}
