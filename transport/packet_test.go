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

package transport

import (
	"context"
	"net"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// UDPEndpoint

func TestUDPEndpointIPv4(t *testing.T) {
	const serverAddr = "127.0.0.10:8888"
	ep := &UDPEndpoint{Address: serverAddr}
	ep.Dialer.Control = func(network, address string, c syscall.RawConn) error {
		require.Equal(t, "udp4", network)
		require.Equal(t, serverAddr, address)
		return nil
	}
	conn, err := ep.Connect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "udp", conn.RemoteAddr().Network())
	assert.Equal(t, serverAddr, conn.RemoteAddr().String())
}

func TestUDPEndpointIPv6(t *testing.T) {
	const serverAddr = "[::1]:8888"
	ep := &UDPEndpoint{Address: serverAddr}
	ep.Dialer.Control = func(network, address string, c syscall.RawConn) error {
		require.Equal(t, "udp6", network)
		require.Equal(t, serverAddr, address)
		return nil
	}
	conn, err := ep.Connect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "udp", conn.RemoteAddr().Network())
	assert.Equal(t, serverAddr, conn.RemoteAddr().String())
}

func TestUDPEndpointDomain(t *testing.T) {
	const serverAddr = "localhost:53"
	ep := &UDPEndpoint{Address: serverAddr}
	var resolvedAddr string
	ep.Dialer.Control = func(network, address string, c syscall.RawConn) error {
		resolvedAddr = address
		return nil
	}
	conn, err := ep.Connect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "udp", conn.RemoteAddr().Network())
	assert.Equal(t, resolvedAddr, conn.RemoteAddr().String())
}

// UDPPacketListener

func TestUDPPacketListenerLocalIPv4Addr(t *testing.T) {
	listener := &UDPPacketListener{Address: "127.0.0.1:0"}
	pc, err := listener.ListenPacket(context.Background())
	require.NoError(t, err)
	require.Equal(t, "udp", pc.LocalAddr().Network())
	listenIP, _, err := net.SplitHostPort(pc.LocalAddr().String())
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", listenIP)
}

func TestUDPPacketListenerLocalIPv6Addr(t *testing.T) {
	listener := &UDPPacketListener{Address: "[::1]:0"}
	pc, err := listener.ListenPacket(context.Background())
	require.NoError(t, err)
	require.Equal(t, "udp", pc.LocalAddr().Network())
	listenIP, _, err := net.SplitHostPort(pc.LocalAddr().String())
	require.NoError(t, err)
	require.Equal(t, "::1", listenIP)
}

func TestUDPPacketListenerLocalhost(t *testing.T) {
	listener := &UDPPacketListener{Address: "localhost:0"}
	pc, err := listener.ListenPacket(context.Background())
	require.NoError(t, err)
	require.Equal(t, "udp", pc.LocalAddr().Network())
	listenIP, _, err := net.SplitHostPort(pc.LocalAddr().String())
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", listenIP)
}

func TestUDPPacketListenerDefaulAddr(t *testing.T) {
	listener := &UDPPacketListener{}
	pc, err := listener.ListenPacket(context.Background())
	require.Equal(t, "udp", pc.LocalAddr().Network())
	require.NoError(t, err)
	listenIP, _, err := net.SplitHostPort(pc.LocalAddr().String())
	require.NoError(t, err)
	require.Equal(t, "::", listenIP)
}

// UDPPacketDialer

func TestUDPPacketDialer(t *testing.T) {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)
	require.Equal(t, "udp", server.LocalAddr().Network())

	dialer := &UDPDialer{}
	conn, err := dialer.Dial(context.Background(), server.LocalAddr().String())
	require.NoError(t, err)

	request := []byte("PING")
	conn.Write(request)
	receivedRequest := make([]byte, 5)
	n, clientAddr, err := server.ReadFrom(receivedRequest)
	require.NoError(t, err)
	require.Equal(t, request, receivedRequest[:n])

	response := []byte("PONG")
	n, err = server.WriteTo(response, clientAddr)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	receivedResponse := make([]byte, 5)
	n, err = conn.Read(receivedResponse)
	require.NoError(t, err)
	require.Equal(t, response, receivedResponse[:n])
}

// PacketListenerDialer

func TestPacketListenerDialer(t *testing.T) {
	request := []byte("Request")
	response := []byte("Response")

	serverListener := UDPPacketListener{Address: "127.0.0.1:0"}
	serverPacketConn, err := serverListener.ListenPacket(context.Background())
	require.NoError(t, err, "Failed to create UDP listener: %v", err)
	t.Logf("Listening on %v", serverPacketConn.LocalAddr())
	defer serverPacketConn.Close()

	var running sync.WaitGroup
	running.Add(2)

	// Server
	go func() {
		defer running.Done()
		receivedRequest := make([]byte, len(request)+1)
		n, clientAddr, err := serverPacketConn.ReadFrom(receivedRequest)
		require.NoError(t, err, "ReadFrom failed: %v", err)
		require.Equal(t, request, receivedRequest[:n])

		n, err = serverPacketConn.WriteTo(response, clientAddr)
		require.NoError(t, err, "WriteTo failed: %v", err)
		require.Equal(t, len(response), n)
	}()

	// Client
	go func() {
		defer func() {
			if t.Failed() {
				t.Log("Closing server")
				serverPacketConn.Close()
			}
			running.Done()
		}()

		serverEndpoint := &PacketListenerDialer{
			Listener: UDPPacketListener{Address: "127.0.0.1:0"},
		}
		conn, err := serverEndpoint.Dial(context.Background(), serverPacketConn.LocalAddr().String())
		require.NoError(t, err)
		t.Logf("Connected to %v from %v", conn.RemoteAddr(), conn.LocalAddr())
		defer func() {
			require.Nil(t, conn.Close())
		}()
		_, ok := conn.(net.PacketConn)
		require.True(t, ok)

		n, err := conn.Write(request)
		require.NoError(t, err, "Failed Write: %v", err)
		require.Equal(t, len(request), n)

		receivedResponse := make([]byte, len(response))
		n, err = conn.Read(receivedResponse)
		require.NoError(t, err)
		require.Equal(t, response, receivedResponse[:n])

	}()

	running.Wait()
}

// PacketConn assertions

func TestPacketConnInvalidArgument(t *testing.T) {
	serverListener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)
	t.Logf("Listening on %v", serverListener.LocalAddr())

	netAddr, err := MakeNetAddr("udp", "localhost:8888")
	require.NoError(t, err)

	_, err = serverListener.WriteTo([]byte("PING"), netAddr)
	// This returns Invalid Argument because netAddr is not a *UDPAddr
	require.ErrorIs(t, err, syscall.EINVAL)
}
