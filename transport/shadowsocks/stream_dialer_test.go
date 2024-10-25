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

package shadowsocks

import (
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/shadowsocks/go-shadowsocks2/socks"
	"github.com/stretchr/testify/require"
)

func TestStreamDialer_Dial(t *testing.T) {
	key := makeTestKey(t)
	proxy, running := startShadowsocksTCPEchoProxy(key, testTargetAddr, t)
	d, err := NewStreamDialer(&transport.TCPEndpoint{Address: proxy.Addr().String()}, key)
	if err != nil {
		t.Fatalf("Failed to create StreamDialer: %v", err)
	}
	conn, err := d.DialStream(context.Background(), testTargetAddr)
	if err != nil {
		t.Fatalf("StreamDialer.Dial failed: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	expectEchoPayload(conn, makeTestPayload(1024), make([]byte, 1024), t)
	conn.Close()

	proxy.Close()
	running.Wait()
}

func TestStreamDialer_DialNoPayload(t *testing.T) {
	key := makeTestKey(t)
	proxy, running := startShadowsocksTCPEchoProxy(key, testTargetAddr, t)
	d, err := NewStreamDialer(&transport.TCPEndpoint{Address: proxy.Addr().String()}, key)
	if err != nil {
		t.Fatalf("Failed to create StreamDialer: %v", err)
	}
	// Extend the wait to be safer.
	d.ClientDataWait = 0 * time.Millisecond

	conn, err := d.DialStream(context.Background(), testTargetAddr)
	if err != nil {
		t.Fatalf("StreamDialer.Dial failed: %v", err)
	}

	// Wait for more than 100 milliseconds to ensure that the target
	// address is sent.
	time.Sleep(100 * time.Millisecond)
	// Force the echo server to verify the target address.
	conn.Close()

	proxy.Close()
	running.Wait()
}

func TestStreamDialer_DialFastClose(t *testing.T) {
	// Set up a listener that verifies no data is sent.
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err, "ListenTCP failed: %v", err)
	defer listener.Close()

	var running sync.WaitGroup
	running.Add(2)
	// Server
	go func() {
		defer running.Done()
		conn, err := listener.Accept()
		require.NoError(t, err)
		defer conn.Close()
		buf := make([]byte, 64)
		n, err := conn.Read(buf)
		if n > 0 || err != io.EOF {
			t.Errorf("Expected EOF, got %v, %v", buf[:n], err)
		}
	}()

	// Client
	go func() {
		defer running.Done()
		key := makeTestKey(t)
		proxyEndpoint := &transport.TCPEndpoint{Address: listener.Addr().String()}
		d, err := NewStreamDialer(proxyEndpoint, key)
		require.NoError(t, err, "Failed to create StreamDialer: %v", err)
		// Extend the wait to be safer.
		d.ClientDataWait = 100 * time.Millisecond

		conn, err := d.DialStream(context.Background(), testTargetAddr)
		require.NoError(t, err, "StreamDialer.Dial failed: %v", err)

		// Wait for less than 100 milliseconds to ensure that the target
		// address is not sent.
		time.Sleep(1 * time.Millisecond)
		// Close the connection before the target address is sent.
		conn.Close()
	}()

	// Wait for the listener to verify the close.
	running.Wait()
}

func TestStreamDialer_TCPPrefix(t *testing.T) {
	prefix := []byte("test prefix")

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenTCP failed: %v", err)
	}
	var running sync.WaitGroup
	running.Add(1)
	go func() {
		defer running.Done()
		defer listener.Close()
		clientConn, err := listener.AcceptTCP()
		if err != nil {
			t.Logf("AcceptTCP failed: %v", err)
			return
		}
		defer clientConn.Close()
		prefixReceived := make([]byte, len(prefix))
		if _, err := io.ReadFull(clientConn, prefixReceived); err != nil {
			t.Error(err)
		}
		for i := range prefix {
			if prefixReceived[i] != prefix[i] {
				t.Error("prefix contents mismatch")
			}
		}
	}()

	key := makeTestKey(t)
	d, err := NewStreamDialer(&transport.TCPEndpoint{Address: listener.Addr().String()}, key)
	if err != nil {
		t.Fatalf("Failed to create StreamDialer: %v", err)
	}
	d.SaltGenerator = NewPrefixSaltGenerator(prefix)
	conn, err := d.DialStream(context.Background(), testTargetAddr)
	if err != nil {
		t.Fatalf("StreamDialer.Dial failed: %v", err)
	}
	conn.Write(nil)
	conn.Close()
	running.Wait()
}

func BenchmarkStreamDialer_Dial(b *testing.B) {
	b.StopTimer()
	b.ResetTimer()

	key := makeTestKey(b)
	proxy, running := startShadowsocksTCPEchoProxy(key, testTargetAddr, b)
	d, err := NewStreamDialer(&transport.TCPEndpoint{Address: proxy.Addr().String()}, key)
	if err != nil {
		b.Fatalf("Failed to create StreamDialer: %v", err)
	}
	conn, err := d.DialStream(context.Background(), testTargetAddr)
	if err != nil {
		b.Fatalf("StreamDialer.Dial failed: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	buf := make([]byte, 1024)
	for n := 0; n < b.N; n++ {
		payload := makeTestPayload(1024)
		b.StartTimer()
		expectEchoPayload(conn, payload, buf, b)
		b.StopTimer()
	}

	conn.Close()
	proxy.Close()
	running.Wait()
}

func startShadowsocksTCPEchoProxy(key *EncryptionKey, expectedTgtAddr string, t testing.TB) (net.Listener, *sync.WaitGroup) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenTCP failed: %v", err)
	}
	t.Logf("Starting SS TCP echo proxy at %v\n", listener.Addr())
	var running sync.WaitGroup
	running.Add(1)
	go func() {
		defer running.Done()
		defer listener.Close()
		for {
			clientConn, err := listener.AcceptTCP()
			if err != nil {
				if err != net.ErrClosed {
					t.Logf("AcceptTCP failed: %v", err)
				}
				return
			}
			running.Add(1)
			go func() {
				defer running.Done()
				defer clientConn.Close()
				ssr := NewReader(clientConn, key)
				ssw := NewWriter(clientConn, key)
				ssClientConn := transport.WrapConn(clientConn, ssr, ssw)

				tgtAddr, err := socks.ReadAddr(ssClientConn)
				if err != nil {
					t.Fatalf("Failed to read target address: %v", err)
				}
				if tgtAddr.String() != expectedTgtAddr {
					t.Fatalf("Expected target address '%v'. Got '%v'", expectedTgtAddr, tgtAddr)
				}
				io.Copy(ssw, ssr)
			}()
		}
	}()
	return listener, &running
}
