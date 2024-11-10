//go:build !linux

package sockopt

import (
	"errors"
	"fmt"
	"net"
)

func isConnectionSendingBytesImplemented() bool {
	return false
}

func isConnectionSendingBytes(_ *net.TCPConn) (bool, error) {
	return false, fmt.Errorf("%w: checking if socket is sending bytes is not implemented on this platform", errors.ErrUnsupported)
}
