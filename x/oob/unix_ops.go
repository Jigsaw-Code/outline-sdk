//go:build linux || darwin

package oob

import (
	"golang.org/x/sys/unix"
	"net"
	"syscall"
)

const MSG_OOB = unix.MSG_OOB

type SocketDescriptor int

func sendTo(fd SocketDescriptor, data []byte, flags int) (err error) {
	return syscall.Sendmsg(int(fd), data, nil, nil, flags)
}

func getSocketDescriptor(conn *net.TCPConn) (SocketDescriptor, error) {
	file, err := conn.File()
	if err != nil {
		return 0, err
	}
	return SocketDescriptor(file.Fd()), nil
}
