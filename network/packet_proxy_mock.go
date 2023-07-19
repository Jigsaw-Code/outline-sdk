package network

import "net"

type PacketProxy interface {
	NewSession(PacketResponseReceiver) (PacketRequestSender, error)
}

type PacketRequestSender interface {
	WriteTo(p []byte, destination net.Addr) (int, error)
	Close() error
}

type PacketResponseReceiver interface {
	WriteFrom(p []byte, source net.Addr) (int, error)
	Close() error
}
