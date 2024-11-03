//go:build linux || darwin

package oob

import (
	"net"
	"syscall"
)

type SocketDescriptor int

func sendTo(fd SocketDescriptor, data []byte, flags int) (err error) {
	return syscall.Sendto(int(fd), data, flags, nil)
}

func getSocketDescriptor(conn *net.TCPConn) (SocketDescriptor, error) {
	file, err := conn.File()
	if err != nil {
		return 0, err
	}
	return SocketDescriptor(file.Fd()), nil
}
