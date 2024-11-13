package oob

import (
	"bufio"
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

const (
	msg    = "Hello OOB!\n"
	msgLen = len(msg)
)

// OOBDialerTestSuite - test suite for testing oobDialer and oobWriter
type OOBDialerTestSuite struct {
	suite.Suite
	server     net.Listener
	dataChan   chan []byte
	serverAddr string
}

// SetupSuite - runs once before all tests
func (suite *OOBDialerTestSuite) SetupSuite() {
	// Start TCP server
	listener, dataChan := startTestServer(suite.T())
	suite.server = listener
	suite.dataChan = dataChan
	suite.serverAddr = listener.Addr().String()
}

// TearDownSuite - runs once after all tests
func (suite *OOBDialerTestSuite) TearDownSuite() {
	suite.server.Close()
	close(suite.dataChan)
}

// startTestServer - starts a test server and returns listener and data channel

func startTestServer(t *testing.T) (net.Listener, chan []byte) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err, "Failed to create server")

	dataChan := make(chan []byte, 10)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			go func(conn net.Conn) {
				defer conn.Close()

				scanner := bufio.NewScanner(conn)
				for scanner.Scan() {
					line := scanner.Bytes()
					dataChan <- append([]byte{}, line...)
					break
				}

				if err := scanner.Err(); err != nil {
					t.Logf("Error reading data: %v", err)
				}
			}(conn)
		}
	}()

	return listener, dataChan
}

// TestDialStreamWithDifferentParameters - test data transmission with different parameters
func (suite *OOBDialerTestSuite) TestDialStreamWithDifferentParameters() {
	tests := []struct {
		oobPosition int64
		oobByte     byte
		disOOB      bool
		delay       time.Duration
	}{
		{oobPosition: 0, oobByte: 0x01, disOOB: false, delay: 100 * time.Millisecond},
		{oobPosition: 0, oobByte: 0x01, disOOB: true, delay: 100 * time.Millisecond},

		{oobPosition: 2, oobByte: 0x02, disOOB: true, delay: 200 * time.Millisecond},
		{oobPosition: 2, oobByte: 0x02, disOOB: false, delay: 200 * time.Millisecond},

		{oobPosition: int64(msgLen) - 2, oobByte: 0x02, disOOB: true, delay: 200 * time.Millisecond},
		{oobPosition: int64(msgLen) - 2, oobByte: 0x02, disOOB: false, delay: 200 * time.Millisecond},

		{oobPosition: int64(msgLen) - 1, oobByte: 0x02, disOOB: true, delay: 200 * time.Millisecond},
		{oobPosition: int64(msgLen) - 1, oobByte: 0x02, disOOB: false, delay: 200 * time.Millisecond},
	}

	for _, tt := range tests {
		suite.Run("Testing with different parameters", func() {
			ctx := context.Background()

			dialer := &transport.TCPDialer{
				Dialer: net.Dialer{},
			}
			oobDialer, err := NewStreamDialer(dialer, tt.oobPosition, tt.oobByte, tt.disOOB, tt.delay)

			conn, err := oobDialer.DialStream(ctx, suite.serverAddr)

			require.NoError(suite.T(), err)

			// Send test message
			message := []byte("Hello OOB!\n")
			n, err := conn.Write(message)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), len(message), n)

			// Check that the server received the message
			receivedData := <-suite.dataChan
			assert.Equal(suite.T(), string(message[0:len(message)-1]), string(receivedData))
		})
	}
}

// TestOOBDialerSuite - main test suite
func TestOOBDialerSuite(t *testing.T) {
	suite.Run(t, new(OOBDialerTestSuite))
}
