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
	"testing"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/require"
)

func TestWithWriteTimeoutOptionWorks(t *testing.T) {
	pl := &transport.UDPListener{}

	defProxy, err := NewPacketProxyFromPacketListener(pl)
	require.NoError(t, err)
	require.NotNil(t, defProxy)
	require.Equal(t, 30*time.Second, defProxy.writeIdleTimeout) // default timeout is 30s

	altProxy, err := NewPacketProxyFromPacketListener(pl, WithPacketListenerWriteIdleTimeout(5*time.Minute))
	require.NoError(t, err)
	require.NotNil(t, altProxy)
	require.Equal(t, 5*time.Minute, altProxy.writeIdleTimeout)
}
