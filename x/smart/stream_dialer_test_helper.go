package smart

import (
	"context"
	"net"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/mock"
)

// MockStreamDialer is a mock implementation of transport.StreamDialer.
type MockStreamDialer struct {
	mock.Mock
}

func (m *MockStreamDialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	args := m.Called(ctx, addr)
	// The first argument is the StreamConn, the second is the error.
	return args.Get(0).(transport.StreamConn), args.Error(1)
}

// MockPacketDialer is a mock implementation of transport.PacketDialer.
type MockPacketDialer struct {
	mock.Mock
}

func (m *MockPacketDialer) DialPacket(ctx context.Context, addr string) (net.Conn, error) {
	args := m.Called(ctx, addr)
	// The first argument is the PacketConn, the second is the error.
	return args.Get(0).(net.Conn), args.Error(1)
}

// MockStreamConn is a mock implementation of transport.StreamConn.
type MockStreamConn struct {
	mock.Mock
}

func (m *MockStreamConn) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStreamConn) Read(b []byte) (n int, err error) {
	args := m.Called(b)
	return args.Int(0), args.Error(1)
}

func (m *MockStreamConn) Write(b []byte) (n int, err error) {
	args := m.Called(b)
	return args.Int(0), args.Error(1)
}

func (m *MockStreamConn) LocalAddr() net.Addr {
	args := m.Called()
	return args.Get(0).(net.Addr)
}

func (m *MockStreamConn) RemoteAddr() net.Addr {
	args := m.Called()
	return args.Get(0).(net.Addr)
}

func (m *MockStreamConn) SetDeadline(t time.Time) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *MockStreamConn) SetReadDeadline(t time.Time) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *MockStreamConn) SetWriteDeadline(t time.Time) error {
	args := m.Called(t)
	return args.Error(0)
}

// MockPacketConn is a mock implementation of net.PacketConn.
type MockPacketConn struct {
	mock.Mock
}

func (m *MockPacketConn) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPacketConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	args := m.Called(b)
	return args.Int(0), args.Get(1).(net.Addr), args.Error(2)
}

func (m *MockPacketConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	args := m.Called(b, addr)
	return args.Int(0), args.Error(1)
}

func (m *MockPacketConn) LocalAddr() net.Addr {
	args := m.Called()
	return args.Get(0).(net.Addr)
}

func (m *MockPacketConn) SetDeadline(t time.Time) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *MockPacketConn) SetReadDeadline(t time.Time) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *MockPacketConn) SetWriteDeadline(t time.Time) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *MockPacketConn) SetReadBuffer(bytes int) error {
	args := m.Called(bytes)
	return args.Error(0)
}

func (m *MockPacketConn) SetWriteBuffer(bytes int) error {
	args := m.Called(bytes)
	return args.Error(0)
}
