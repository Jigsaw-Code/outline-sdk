package httpproxy

import (
	"io"
	"net"
	"os"
	"syscall"
	"errors"
)

// Library specific errors.
var (
	ErrPanic                       = errors.New("panic")
	ErrResponseWrite               = errors.New("response write")
	ErrRequestRead                 = errors.New("request read")
	ErrRemoteConnect               = errors.New("remote connect")
	ErrNotSupportHijacking         = errors.New("hijacking not supported")
	ErrRoundTrip                   = errors.New("round trip")
	ErrUnsupportedTransferEncoding = errors.New("unsupported transfer encoding")
	ErrNotSupportHTTPVer           = errors.New("http version not supported")
	ErrProxyConnectionClosed       = errors.New("proxy connection closed")
	ErrNonProxyRequest             = errors.New("non-proxy request")
)

func isConnectionClosed(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, io.EOF) {
		return true
	}

	i := 0
	var newerr = &err
	for opError, ok := (*newerr).(*net.OpError); ok && i < 10; {
		i++
		newerr = &opError.Err
		if syscallError, ok := (*newerr).(*os.SyscallError); ok {
			if syscallError.Err == syscall.EPIPE || syscallError.Err == syscall.ECONNRESET || syscallError.Err == syscall.EPROTOTYPE {
				return true
			}
		}
	}

	return false
}
