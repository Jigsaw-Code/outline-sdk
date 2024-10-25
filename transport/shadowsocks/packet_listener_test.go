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

func TestShadowsocksPacketListener_ListenPacket(t *testing.T) {
	key := makeTestKey(t)
	proxy, running := startShadowsocksUDPEchoServer(key, testTargetAddr, t)
	proxyEndpoint := transport.UDPEndpoint{Address: proxy.LocalAddr().String()}
	d, err := NewPacketListener(proxyEndpoint, key)
	if err != nil {
		t.Fatalf("Failed to create PacketListener: %v", err)
	}
	conn, err := d.ListenPacket(context.Background())
	if err != nil {
		t.Fatalf("PacketListener.ListenPacket failed: %v", err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	pcrw := &packetConnReadWriter{PacketConn: conn}
	pcrw.targetAddr, err = transport.MakeNetAddr("udp", testTargetAddr)
	require.NoError(t, err)
	expectEchoPayload(pcrw, makeTestPayload(1024), make([]byte, 1024), t)

	proxy.Close()
	running.Wait()
}

func BenchmarkShadowsocksPacketListener_ListenPacket(b *testing.B) {
	b.StopTimer()
	b.ResetTimer()

	key := makeTestKey(b)
	proxy, running := startShadowsocksUDPEchoServer(key, testTargetAddr, b)
	targetAddr, err := transport.MakeNetAddr("udp", testTargetAddr)
	require.NoError(b, err)
	proxyEndpoint := transport.UDPEndpoint{Address: proxy.LocalAddr().String()}
	d, err := NewPacketListener(proxyEndpoint, key)
	if err != nil {
		b.Fatalf("Failed to create PacketListener: %v", err)
	}
	conn, err := d.ListenPacket(context.Background())
	if err != nil {
		b.Fatalf("PacketListener.ListenPacket failed: %v", err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	buf := make([]byte, clientUDPBufferSize)
	for n := 0; n < b.N; n++ {
		payload := makeTestPayload(1024)
		pcrw := &packetConnReadWriter{PacketConn: conn, targetAddr: targetAddr}
		b.StartTimer()
		expectEchoPayload(pcrw, payload, buf, b)
		b.StopTimer()
	}

	proxy.Close()
	running.Wait()
}

func startShadowsocksUDPEchoServer(key *EncryptionKey, expectedTgtAddr string, t testing.TB) (net.Conn, *sync.WaitGroup) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Proxy ListenUDP failed: %v", err)
	}
	t.Logf("Starting SS UDP echo proxy at %v\n", conn.LocalAddr())
	cipherBuf := make([]byte, clientUDPBufferSize)
	clientBuf := make([]byte, clientUDPBufferSize)
	var running sync.WaitGroup
	running.Add(1)
	go func() {
		defer running.Done()
		defer conn.Close()
		for {
			n, clientAddr, err := conn.ReadFromUDP(cipherBuf)
			if err != nil {
				t.Logf("Failed to read from UDP conn: %v", err)
				return
			}
			buf, err := Unpack(clientBuf, cipherBuf[:n], key)
			if err != nil {
				t.Fatalf("Failed to decrypt: %v", err)
			}
			tgtAddr := socks.SplitAddr(buf)
			if tgtAddr == nil {
				t.Fatalf("Failed to read target address: %v", err)
			}
			if tgtAddr.String() != expectedTgtAddr {
				t.Fatalf("Expected target address '%v'. Got '%v'", expectedTgtAddr, tgtAddr)
			}
			// Echo both the payload and SOCKS address.
			buf, err = Pack(cipherBuf, buf, key)
			if err != nil {
				t.Fatalf("Failed to encrypt: %v", err)
			}
			conn.WriteTo(buf, clientAddr)
			if err != nil {
				t.Fatalf("Failed to write: %v", err)
			}
		}
	}()
	return conn, &running
}

// io.ReadWriter adapter for net.PacketConn. Used to share code between UDP and TCP tests.
type packetConnReadWriter struct {
	net.PacketConn
	io.ReadWriter
	targetAddr net.Addr
}

func (pc *packetConnReadWriter) Read(b []byte) (n int, err error) {
	n, _, err = pc.PacketConn.ReadFrom(b)
	return
}

func (pc *packetConnReadWriter) Write(b []byte) (int, error) {
	return pc.PacketConn.WriteTo(b, pc.targetAddr)
}
