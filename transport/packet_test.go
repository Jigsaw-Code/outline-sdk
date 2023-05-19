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
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.org/x/sys/unix"
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
	require.Nil(t, err)
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
	require.Nil(t, err)
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
	require.ErrorIs(t, err, nil)
	assert.Equal(t, "udp", conn.RemoteAddr().Network())
	assert.Equal(t, resolvedAddr, conn.RemoteAddr().String())
}

// UDPPacketListener

func TestUDPPacketListenerLocalIPv4Addr(t *testing.T) {
	listener := &UDPPacketListener{Address: "127.0.0.1:0"}
	pc, err := listener.ListenPacket(context.Background())
	require.Nil(t, err)
	require.Equal(t, "udp", pc.LocalAddr().Network())
	listenIP, _, err := net.SplitHostPort(pc.LocalAddr().String())
	require.Nil(t, err)
	require.Equal(t, "127.0.0.1", listenIP)
}

func TestUDPPacketListenerLocalIPv6Addr(t *testing.T) {
	listener := &UDPPacketListener{Address: "[::1]:0"}
	pc, err := listener.ListenPacket(context.Background())
	require.Nil(t, err)
	require.Equal(t, "udp", pc.LocalAddr().Network())
	listenIP, _, err := net.SplitHostPort(pc.LocalAddr().String())
	require.Nil(t, err)
	require.Equal(t, "::1", listenIP)
}

func TestUDPPacketListenerLocalhost(t *testing.T) {
	listener := &UDPPacketListener{Address: "localhost:0"}
	pc, err := listener.ListenPacket(context.Background())
	require.Nil(t, err)
	require.Equal(t, "udp", pc.LocalAddr().Network())
	listenIP, _, err := net.SplitHostPort(pc.LocalAddr().String())
	require.Nil(t, err)
	require.Equal(t, "127.0.0.1", listenIP)
}

func TestUDPPacketListenerDefaulAddr(t *testing.T) {
	listener := &UDPPacketListener{}
	pc, err := listener.ListenPacket(context.Background())
	require.Equal(t, "udp", pc.LocalAddr().Network())
	require.Nil(t, err)
	listenIP, _, err := net.SplitHostPort(pc.LocalAddr().String())
	require.Nil(t, err)
	require.Equal(t, "::", listenIP)
}

// UDPPacketDialer

func TestUDPPacketDialer(t *testing.T) {
	server, err := net.ListenUDP("udp", &net.UDPAddr{})
	require.Nil(t, err)
	require.Equal(t, "udp", server.LocalAddr().Network())

	dialer := &UDPPacketDialer{}
	conn, err := dialer.Dial(context.Background(), server.LocalAddr().String())
	require.Nil(t, err)

	request := []byte("PING")
	conn.Write(request)
	receivedRequest := make([]byte, 5)
	n, clientAddr, err := server.ReadFrom(receivedRequest)
	require.Nil(t, err)
	require.Equal(t, request, receivedRequest[:n])

	response := []byte("PONG")
	n, err = server.WriteTo(response, clientAddr)
	require.ErrorIs(t, err, nil)
	require.Equal(t, 4, n)
	receivedResponse := make([]byte, 5)
	n, err = conn.Read(receivedResponse)
	require.Nil(t, err)
	require.Equal(t, response, receivedResponse[:n])
}

// PacketListenerDialer

func TestPacketListenerDialer(t *testing.T) {
	request := []byte("Request")
	response := []byte("Response")

	serverListener := UDPPacketListener{Address: "127.0.0.1:0"}
	serverPacketConn, err := serverListener.ListenPacket(context.Background())
	require.Nilf(t, err, "Failed to create UDP listener: %v", err)
	t.Logf("Listening on %v", serverPacketConn.LocalAddr())
	defer serverPacketConn.Close()

	var running sync.WaitGroup
	running.Add(2)

	// Server
	go func() {
		defer running.Done()
		receivedRequest := make([]byte, len(request)+1)
		n, clientAddr, err := serverPacketConn.ReadFrom(receivedRequest)
		require.Nilf(t, err, "ReadFrom failed: %v", err)
		require.Equal(t, request, receivedRequest[:n])

		n, err = serverPacketConn.WriteTo(response, clientAddr)
		require.Nilf(t, err, "WriteTo failed: %v", err)
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
		require.Nil(t, err)
		t.Logf("Connected to %v from %v", conn.RemoteAddr(), conn.LocalAddr())
		defer func() {
			require.Nil(t, conn.Close())
		}()
		_, ok := conn.(net.PacketConn)
		require.True(t, ok)

		n, err := conn.Write(request)
		require.Nilf(t, err, "Failed Write: %v", err)
		require.Equal(t, len(request), n)

		receivedResponse := make([]byte, len(response))
		n, err = conn.Read(receivedResponse)
		require.Nil(t, err)
		require.Equal(t, response, receivedResponse[:n])

	}()

	running.Wait()
}

// PacketConn assertions

func TestPacketConnInvalidArgument(t *testing.T) {
	serverListener, err := net.ListenUDP("udp", nil)
	require.ErrorIs(t, err, nil)
	t.Logf("Listening on %v", serverListener.LocalAddr())

	netAddr, err := MakeNetAddr("udp", "localhost:8888")
	require.ErrorIs(t, err, nil)

	_, err = serverListener.WriteTo([]byte("PING"), netAddr)
	// This returns Invalid Argument because netAddr is not a *UDPAddr
	require.ErrorIs(t, err, syscall.EINVAL)
}

func setsockoptInt(conn interface {
	SyscallConn() (syscall.RawConn, error)
}, level, opt, value int) error {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	rawConn.Control(func(fd uintptr) {
		err = syscall.SetsockoptInt(int(fd), level, opt, value)
	})
	return err
}

func TestPacketConnDestAddr(t *testing.T) {
	server, err := net.ListenUDP("udp", nil)
	require.ErrorIs(t, err, nil)

	// Set RECVPKTINFO. See https://blog.cloudflare.com/everything-you-ever-wanted-to-know-about-udp-sockets-but-were-afraid-to-ask-part-1/#sourcing-packets-from-a-wildcard-socket
	// To check for supported options, use getsockopt and check for ENOPROTOOPT
	//
	// Windows Compatibility:
	// - IPPROTO_IP: https://learn.microsoft.com/en-us/windows/win32/winsock/ipproto-ip-socket-options
	//   - IP_PKTINFO, IP_TTL
	// - IPPROTO_IPV6: https://learn.microsoft.com/en-us/windows/win32/winsock/ipproto-ipv6-socket-options
	//   - IP_ORIGINAL_ARRIVAL_IF, IPV6_PKTINFO, IPV6_RECVIF
	if server.LocalAddr().(*net.UDPAddr).IP.To4() != nil {
		t.Logf("IPv4 socket. Address: %v", server.LocalAddr())
		// darwin: ok, linux & windows: not defined
		// May be able to use syscall.IP_RECVDSTADDR or syscall.IP_PKTINFO
		require.ErrorIs(t, setsockoptInt(server, syscall.IPPROTO_IP, syscall.IP_RECVPKTINFO, 1), nil)
	} else if server.LocalAddr().(*net.UDPAddr).IP.To16() != nil {
		t.Logf("IPv6 socket. Address: %v", server.LocalAddr())
		// TODO(fortuna): need to make it work on Windows. (unix not defined)
		// From https://www.rfc-editor.org/rfc/rfc3542.
		require.ErrorIs(t, setsockoptInt(server, syscall.IPPROTO_IPV6, unix.IPV6_RECVPKTINFO, 1), nil)
	} else {
		t.Error("Invalid address")
	}

	serverPort := server.LocalAddr().(*net.UDPAddr).Port

	serverBuf := make([]byte, 6)

	{
		client4, err := net.ListenUDP("udp4", nil)
		require.ErrorIs(t, err, nil)

		n, err := client4.WriteToUDP([]byte("PING4"), &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: serverPort})
		require.ErrorIs(t, err, nil)
		require.Equal(t, 5, n)

		// oob := ipv4.NewControlMessage(ipv4.FlagDst)
		oob := make([]byte, 100)
		n, oobn, flags, clientAddr, err := server.ReadMsgUDP(serverBuf, oob)
		require.ErrorIs(t, err, nil)
		require.Equal(t, string(serverBuf[:n]), "PING4")
		require.Equal(t, net.IP{127, 0, 0, 1}, clientAddr.IP.To4())
		require.Equal(t, client4.LocalAddr().(*net.UDPAddr).Port, clientAddr.Port)

		var cm6 ipv6.ControlMessage
		require.ErrorIs(t, cm6.Parse(oob[:oobn]), nil)
		t.Logf("Control Message: %#v", cm6)
		require.Equal(t, net.IP{127, 0, 0, 1}, cm6.Dst.To4())
		require.Equal(t, 1, cm6.IfIndex)
		require.Equal(t, 0, flags)

		// var cm4 ipv4.ControlMessage
		// require.ErrorIs(t, cm4.Parse(oob[:oobn]), nil)
		// t.Logf("Control Message: %#v", cm4)
		// require.Equal(t, net.IP{127, 0, 0, 1}, cm4.Dst)
		// require.Equal(t, 1, cm4.IfIndex)
		// require.Equal(t, 0, flags)
	}

	{
		client6, err := net.ListenUDP("udp6", nil)
		require.ErrorIs(t, err, nil)

		n, err := client6.WriteToUDP([]byte("PING6"), &net.UDPAddr{IP: net.IPv6loopback, Port: serverPort})
		require.ErrorIs(t, err, nil)
		require.Equal(t, 5, n)

		oob := ipv6.NewControlMessage(ipv6.FlagDst)
		n, oobn, flags, clientAddr, err := server.ReadMsgUDP(serverBuf, oob)
		require.ErrorIs(t, err, nil)
		require.Equal(t, string(serverBuf[:n]), "PING6")
		require.Equal(t, net.IPv6loopback, clientAddr.IP)
		require.Equal(t, client6.LocalAddr().(*net.UDPAddr).Port, clientAddr.Port)

		var cm6 ipv6.ControlMessage
		require.ErrorIs(t, cm6.Parse(oob[:oobn]), nil)
		t.Logf("Control Message: %#v", cm6)
		require.Equal(t, net.IPv6loopback, cm6.Dst)
		require.Equal(t, 1, cm6.IfIndex)
		require.Equal(t, 0, flags)
	}
}

func TestPacketConnDestAddr2(t *testing.T) {
	server, err := net.ListenUDP("udp", nil)
	require.ErrorIs(t, err, nil)

	// Set RECVPKTINFO. See https://blog.cloudflare.com/everything-you-ever-wanted-to-know-about-udp-sockets-but-were-afraid-to-ask-part-1/#sourcing-packets-from-a-wildcard-socket
	// To check for supported options, use getsockopt and check for ENOPROTOOPT
	//
	// Windows Compatibility:
	// - IPPROTO_IP: https://learn.microsoft.com/en-us/windows/win32/winsock/ipproto-ip-socket-options
	//   - IP_PKTINFO, IP_TTL
	// - IPPROTO_IPV6: https://learn.microsoft.com/en-us/windows/win32/winsock/ipproto-ipv6-socket-options
	//   - IP_ORIGINAL_ARRIVAL_IF, IPV6_PKTINFO, IPV6_RECVIF
	if server.LocalAddr().(*net.UDPAddr).IP.To4() != nil {
		t.Logf("IPv4 socket. Address: %v", server.LocalAddr())
		sc := ipv4.NewPacketConn(server)
		require.ErrorIs(t, sc.SetControlMessage(ipv4.FlagDst, true), nil)
	} else if server.LocalAddr().(*net.UDPAddr).IP.To16() != nil {
		t.Logf("IPv6 socket. Address: %v", server.LocalAddr())
		sc := ipv6.NewPacketConn(server)
		require.ErrorIs(t, sc.SetControlMessage(ipv6.FlagDst, true), nil)
	} else {
		t.Error("Invalid address")
	}

	serverPort := server.LocalAddr().(*net.UDPAddr).Port

	serverBuf := make([]byte, 6)
	// // TODO: How big should this be?
	oob := make([]byte, 100)

	client4, err := net.ListenUDP("udp4", nil)
	require.ErrorIs(t, err, nil)

	n, err := client4.WriteToUDP([]byte("PING4"), &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: serverPort})
	require.ErrorIs(t, err, nil)
	require.Equal(t, 5, n)

	n, oobn, flags, clientAddr, err := server.ReadMsgUDP(serverBuf, oob)
	require.ErrorIs(t, err, nil)
	require.Equal(t, string(serverBuf[:n]), "PING4")
	require.Equal(t, net.IP{127, 0, 0, 1}, clientAddr.IP.To4())
	require.Equal(t, client4.LocalAddr().(*net.UDPAddr).Port, clientAddr.Port)
	require.Equal(t, []byte{}, oob[:oobn])
	require.Equal(t, 0, flags)

	client6, err := net.ListenUDP("udp6", nil)
	require.ErrorIs(t, err, nil)

	n, err = client6.WriteToUDP([]byte("PING6"), &net.UDPAddr{IP: net.IPv6loopback, Port: serverPort})
	require.ErrorIs(t, err, nil)
	require.Equal(t, 5, n)

	n, oobn, flags, clientAddr, err = server.ReadMsgUDP(serverBuf, oob)
	require.ErrorIs(t, err, nil)
	require.Equal(t, string(serverBuf[:n]), "PING6")
	require.Equal(t, net.IPv6loopback, clientAddr.IP)
	require.Equal(t, client6.LocalAddr().(*net.UDPAddr).Port, clientAddr.Port)
	require.Equal(t, []byte{}, oob[:oobn])
	require.Equal(t, 0, flags)
}
