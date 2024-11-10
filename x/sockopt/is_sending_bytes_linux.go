//go:build linux

package sockopt

import (
	"net"

	"golang.org/x/sys/unix"
)

func isSocketFdSendingBytes(fd int) (bool, error) {
	tcpInfo, err := unix.GetsockoptTCPInfo(fd, unix.IPPROTO_TCP, unix.TCP_INFO)
	if err != nil {
		return false, err
	}

	// 1 == TCP_ESTABLISHED, but for some reason not available in the package
	if tcpInfo.State != unix.BPF_TCP_ESTABLISHED {
		// If the connection is not established, the socket is not sending bytes
		return false, nil
	}

	return tcpInfo.Notsent_bytes != 0, nil
}

func isConnectionSendingBytesImplemented() bool {
	return true
}

func isConnectionSendingBytes(conn *net.TCPConn) (result bool, err error) {
	syscallConn, err := conn.SyscallConn()
	if err != nil {
		return false, err
	}
	syscallConn.Control(func(fd uintptr) {
		result, err = isSocketFdSendingBytes(int(fd))
	})
	return
}
