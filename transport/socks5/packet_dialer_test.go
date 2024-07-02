package socks5

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/things-go/go-socks5"
	"github.com/things-go/go-socks5/statute"
)

func TestSOCKS5_Associate(t *testing.T) {
	locIP := net.ParseIP("127.0.0.1")

	// Create a local listener
	// This creates a UDP server that responded to "ping"
	// message with "pong" response
	serverAddr := &net.UDPAddr{IP: locIP, Port: 12399}
	server, err := net.ListenUDP("udp", serverAddr)
	require.NoError(t, err)
	defer server.Close()

	go func() {
		buf := make([]byte, 2048)
		for {
			n, remote, err := server.ReadFrom(buf)
			if err != nil {
				return
			}
			require.Equal(t, []byte("ping"), buf[:n])

			server.WriteTo([]byte("pong"), remote) //nolint: errcheck
		}
	}()

	// Create a socks server to proxy "ping" message
	cator := socks5.UserPassAuthenticator{Credentials: socks5.StaticCredentials{
		"testusername": "testpassword",
	}}
	proxySrv := socks5.NewServer(
		socks5.WithAuthMethods([]socks5.Authenticator{cator}),
		socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
	)
	// Start listening
	proxyServerAddress := "127.0.0.1:12355"
	go func() {
		err := proxySrv.ListenAndServe("tcp", proxyServerAddress)
		require.NoError(t, err)
	}()
	time.Sleep(10 * time.Millisecond)

	// // Get a local conn
	// Connect, auth and connec to local
	dialer, err := NewDialer(&transport.TCPEndpoint{Address: proxyServerAddress})
	require.NotNil(t, dialer)
	require.NoError(t, err)
	err = dialer.SetCredentials([]byte("testusername"), []byte("testpassword"))
	require.NoError(t, err)
	dialer.EnablePacket(transport.PacketListenerDialer{Listener: &transport.UDPListener{Address: ":12800"}})
	//dialer.EnablePacket(&transport.UDPDialer{})
	//net.ListenPacket()
	//net.ListenUDP()
	conn, err := dialer.DialPacket(context.Background(), serverAddr.String())
	require.NoError(t, err)
	defer conn.Close()
	fmt.Printf("local address is: %v\n", conn.LocalAddr())

	// Send "ping" message
	_, err = conn.Write([]byte("ping"))
	require.NoError(t, err)
	// wait time for response
	conn.SetDeadline(time.Now().Add(time.Second))
	response := make([]byte, 1024)
	_, err = conn.Read(response)
	fmt.Printf("response: %s\n", response)
	//conn.SetDeadline(time.Time{})
	require.NoError(t, err)
	require.Equal(t, []byte("pong"), response)
	time.Sleep(time.Second * 1)
}

// Test UDP without Outline dialer
func TestSOCKS5_Associate_2(t *testing.T) {
	locIP := net.ParseIP("127.0.0.1")
	// Create a local listener
	echoServerAddr := &net.UDPAddr{IP: locIP, Port: 12400}
	echoServer, err := net.ListenUDP("udp", echoServerAddr)
	require.NoError(t, err)
	defer echoServer.Close()

	go func() {
		buf := make([]byte, 2048)
		for {
			n, remote, err := echoServer.ReadFrom(buf)
			if err != nil {
				return
			}
			require.Equal(t, []byte("ping"), buf[:n])

			echoServer.WriteTo([]byte("pong"), remote) //nolint: errcheck
		}
	}()

	clientAddr := &net.UDPAddr{IP: locIP, Port: 12599}
	client, err := net.ListenUDP("udp", clientAddr)
	require.NoError(t, err)
	defer client.Close()

	// Create a socks server
	// Create a socks server to proxy "ping" message
	cator := socks5.UserPassAuthenticator{Credentials: socks5.StaticCredentials{
		"foo": "bar",
	}}
	proxySrv := socks5.NewServer(
		socks5.WithAuthMethods([]socks5.Authenticator{cator}),
		socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
	)
	// Start listening
	go func() {
		err := proxySrv.ListenAndServe("tcp", "127.0.0.1:12350")
		require.NoError(t, err)
	}()
	time.Sleep(10 * time.Millisecond)

	// Get a local conn
	conn, err := net.Dial("tcp", "127.0.0.1:12350")
	require.NoError(t, err)

	// Connect, auth and connec to local
	req := bytes.NewBuffer(
		[]byte{
			statute.VersionSocks5, 2, statute.MethodNoAuth, statute.MethodUserPassAuth,
			statute.UserPassAuthVersion, 3, 'f', 'o', 'o', 3, 'b', 'a', 'r',
		})
	reqHead := statute.Request{
		Version:  statute.VersionSocks5,
		Command:  statute.CommandAssociate,
		Reserved: 0,
		DstAddr: statute.AddrSpec{
			FQDN:     "",
			IP:       clientAddr.IP,
			Port:     clientAddr.Port,
			AddrType: statute.ATYPIPv4,
		},
	}
	req.Write(reqHead.Bytes())
	// Send all the bytes
	conn.Write(req.Bytes()) //nolint: errcheck

	// Verify response
	expected := []byte{
		statute.VersionSocks5, statute.MethodUserPassAuth, // use user password auth
		statute.UserPassAuthVersion, statute.AuthSuccess, // response auth success
	}

	out := make([]byte, len(expected))
	conn.SetDeadline(time.Now().Add(time.Second)) //nolint: errcheck
	_, err = io.ReadFull(conn, out)
	conn.SetDeadline(time.Time{}) //nolint: errcheck
	require.NoError(t, err)
	require.Equal(t, expected, out)

	rspHead, err := statute.ParseReply(conn)
	require.NoError(t, err)
	require.Equal(t, statute.VersionSocks5, rspHead.Version)
	require.Equal(t, statute.RepSuccess, rspHead.Response)

	ipByte := []byte(echoServerAddr.IP.To4())
	portByte := make([]byte, 2)
	binary.BigEndian.PutUint16(portByte, uint16(echoServerAddr.Port))

	msgBytes := []byte{0, 0, 0, statute.ATYPIPv4}
	msgBytes = append(msgBytes, ipByte...)
	msgBytes = append(msgBytes, portByte...)
	msgBytes = append(msgBytes, []byte("ping")...)
	client.WriteTo(msgBytes, &net.UDPAddr{IP: locIP, Port: rspHead.BndAddr.Port}) //nolint: errcheck
	// t.Logf("proxy bind listen port: %d", rspHead.BndAddr.Port)
	response := make([]byte, 1024)
	n, _, err := client.ReadFrom(response)
	require.NoError(t, err)
	assert.Equal(t, []byte("pong"), response[n-4:n])
	time.Sleep(time.Second * 1)
}

// test request
func TestSOCKS5_Associate_Request(t *testing.T) {
	locIP := net.ParseIP("127.0.0.1")
	// Create a local listener
	echoServerAddr := &net.UDPAddr{IP: locIP, Port: 12399}
	echoServer, err := net.ListenUDP("udp", echoServerAddr)
	require.NoError(t, err)
	defer echoServer.Close()

	go func() {
		buf := make([]byte, 2048)
		for {
			n, remote, err := echoServer.ReadFrom(buf)
			if err != nil {
				return
			}
			require.Equal(t, []byte("ping"), buf[:n])

			echoServer.WriteTo([]byte("pong"), remote) //nolint: errcheck
		}
	}()

	clientAddr := &net.UDPAddr{IP: locIP, Port: 12499}
	client, err := net.ListenUDP("udp", clientAddr)
	require.NoError(t, err)
	defer client.Close()

	// Create a socks server
	// Create a socks server to proxy "ping" message
	cator := socks5.UserPassAuthenticator{Credentials: socks5.StaticCredentials{
		"foo": "bar",
	}}
	proxySrv := socks5.NewServer(
		socks5.WithAuthMethods([]socks5.Authenticator{cator}),
		socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
	)
	// Start listening
	go func() {
		err := proxySrv.ListenAndServe("tcp", "127.0.0.1:12355")
		require.NoError(t, err)
	}()
	time.Sleep(10 * time.Millisecond)

	// Connect, auth and connec to local
	dialer, err := NewDialer(&transport.TCPEndpoint{Address: "127.0.0.1:12355"})
	require.NotNil(t, dialer)
	require.NoError(t, err)
	err = dialer.SetCredentials([]byte("foo"), []byte("bar"))
	require.NoError(t, err)
	sc, bindAddr, err := dialer.request(context.Background(), CmdUDPAssociate, clientAddr.String())
	require.NoError(t, err)
	defer sc.Close()

	ipByte := []byte(echoServerAddr.IP.To4())
	portByte := make([]byte, 2)
	binary.BigEndian.PutUint16(portByte, uint16(echoServerAddr.Port))

	msgBytes := []byte{0, 0, 0, statute.ATYPIPv4}
	msgBytes = append(msgBytes, ipByte...)
	msgBytes = append(msgBytes, portByte...)
	msgBytes = append(msgBytes, []byte("ping")...)
	_, port, _ := net.SplitHostPort(bindAddr)
	portInt, _ := strconv.Atoi(port)
	client.WriteTo(msgBytes, &net.UDPAddr{IP: locIP, Port: portInt}) //nolint: errcheck
	// t.Logf("proxy bind listen port: %d", rspHead.BndAddr.Port)
	response := make([]byte, 1024)
	n, _, err := client.ReadFrom(response)
	require.NoError(t, err)
	assert.Equal(t, []byte("pong"), response[n-4:n])
	time.Sleep(time.Second * 1)
}
