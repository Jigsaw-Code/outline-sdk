// Copyright 2020 The Outline Authors
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

package shadowsocks

import (
	"io"
	"net"
	"sync"
	"testing"

	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/shadowsocks/go-shadowsocks2/shadowaead"
	"github.com/stretchr/testify/require"
)

func TestCompatibility(t *testing.T) {
	cipherName := "chacha20-ietf-poly1305"
	secret := "secret"
	fromLeft := "payload1"
	fromRight := "payload2"
	left, right := net.Pipe()

	var wait sync.WaitGroup
	wait.Add(1)
	key, err := NewEncryptionKey(cipherName, secret)
	require.NoError(t, err, "NewCipher failed: %v", err)
	ssWriter := NewWriter(left, key)
	go func() {
		defer wait.Done()
		var err error
		ssWriter.Write([]byte(fromLeft))

		ssReader := NewReader(left, key)
		receivedByLeft := make([]byte, len(fromRight))
		_, err = ssReader.Read(receivedByLeft)
		require.NoError(t, err, "Read failed: %v", err)
		require.Equal(t, fromRight, string(receivedByLeft))
		left.Close()
	}()

	otherCipher, err := core.PickCipher(cipherName, []byte{}, secret)
	require.NoError(t, err)
	rightSSConn := shadowaead.NewConn(right, otherCipher.(shadowaead.Cipher))
	receivedByRight := make([]byte, len(fromLeft))
	_, err = io.ReadFull(rightSSConn, receivedByRight)
	require.NoError(t, err)
	require.Equal(t, fromLeft, string(receivedByRight))

	_, err = rightSSConn.Write([]byte(fromRight))
	require.NoError(t, err, "Write failed: %v", err)

	rightSSConn.Close()
	wait.Wait()
}
