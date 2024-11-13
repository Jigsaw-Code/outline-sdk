//go:build windows

package oob

import (
	"fmt"
	"net"
	"syscall"
)

const MSG_OOB = windows.MSG_OOB

type SocketDescriptor uintptr

func sendTo(fd SocketDescriptor, data []byte, flags int) (err error) {
	var wsaBuf [1]syscall.WSABuf
	wsaBuf[0].Len = uint32(len(data))
	wsaBuf[0].Buf = &data[0]

	bytesSent := uint32(0)

	return syscall.WSASend(syscall.Handle(fd), &wsaBuf[0], 1, &bytesSent, uint32(flags), nil, nil)
}

func getSocketDescriptor(conn *net.TCPConn) (SocketDescriptor, error) {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return SocketDescriptor(0), fmt.Errorf("oob strategy was unable to get raw conn: %w", err)
	}

	var sysFd syscall.Handle
	err = rawConn.Control(func(fd uintptr) {
		sysFd = syscall.Handle(fd)
	})

	return SocketDescriptor(sysFd), err
}
