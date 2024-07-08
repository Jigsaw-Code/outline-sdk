package socks5

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/things-go/go-socks5"
)

func TestSOCKS5Associate(t *testing.T) {

	// Create a local listener
	// This creates a UDP server that responded to "ping"
	// message with "pong" response
	locIP := net.ParseIP("127.0.0.1")
	// Create a local listener
	echoServerAddr := &net.UDPAddr{IP: locIP, Port: 12199}
	echoServer := setupUDPEchoServer(t, echoServerAddr)
	defer echoServer.Close()

	// Create a socks server to proxy "ping" message
	cator := socks5.UserPassAuthenticator{Credentials: socks5.StaticCredentials{
		"testusername": "testpassword",
	}}
	proxySrv := socks5.NewServer(
		socks5.WithAuthMethods([]socks5.Authenticator{cator}),
		//socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
	)
	// Start listening
	proxyServerAddress := "127.0.0.1:12355"
	go func() {
		err := proxySrv.ListenAndServe("tcp", proxyServerAddress)
		require.NoError(t, err)
	}()
	time.Sleep(10 * time.Millisecond)

	// Connect, auth and connec to local server
	dialer, err := NewDialer(&transport.TCPEndpoint{Address: proxyServerAddress})
	require.NotNil(t, dialer)
	require.NoError(t, err)
	err = dialer.SetCredentials([]byte("testusername"), []byte("testpassword"))
	require.NoError(t, err)
	dialer.EnablePacket(&transport.UDPDialer{})
	conn, err := dialer.DialPacket(context.Background(), echoServerAddr.String())
	require.NoError(t, err)
	defer conn.Close()

	// Send "ping" message
	_, err = conn.Write([]byte("ping"))
	require.NoError(t, err)
	// max wait time for response
	conn.SetDeadline(time.Now().Add(time.Second))
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	//conn.SetDeadline(time.Time{})
	require.NoError(t, err)
	require.Equal(t, []byte("pong"), response[:n])
}

func TestUDPLoopBack(t *testing.T) {
	// Create a local listener
	locIP := net.ParseIP("127.0.0.1")
	// Create a local listener
	echoServerAddr := &net.UDPAddr{IP: locIP, Port: 12199}
	echoServer := setupUDPEchoServer(t, echoServerAddr)
	defer echoServer.Close()

	packDialer := transport.UDPDialer{}
	conn, err := packDialer.DialPacket(context.Background(), echoServerAddr.String())
	require.NoError(t, err)
	conn.Write([]byte("ping"))
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	require.NoError(t, err)
	assert.Equal(t, []byte("pong"), response[:n])
}

func setupUDPEchoServer(t *testing.T, serverAddr *net.UDPAddr) *net.UDPConn {
	server, err := net.ListenUDP("udp", serverAddr)
	require.NoError(t, err)
	go func() {
		buf := make([]byte, 2048)
		for {
			n, remote, err := server.ReadFrom(buf)
			if err != nil {
				//log.Printf("Error reading: %v", err)
				return
			}
			if bytes.Equal(buf[:n], []byte("ping")) {
				server.WriteTo([]byte("pong"), remote)
			}
		}
	}()

	t.Cleanup(func() {
		server.Close()
	})

	return server
}
