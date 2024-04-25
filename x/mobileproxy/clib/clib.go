package clib

// #include <stdlib.h>
import (
	"C"
	"unsafe"

	"github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
)

type ProxyPtr = unsafe.Pointer
type StreamDialerPtr = unsafe.Pointer

func NewStreamDialerFromConfig(config *C.char) StreamDialerPtr {
	streamDialer, err := mobileproxy.NewStreamDialerFromConfig(C.GoString(config))

	if err != nil {
		return nil
	}

	return unsafe.Pointer(streamDialer)
}

func RunProxy(address *C.char, dialer StreamDialerPtr) ProxyPtr {
	proxy, err := mobileproxy.RunProxy(C.GoString(address), (*mobileproxy.StreamDialer)(dialer))

	if err != nil {
		return nil
	}

	return unsafe.Pointer(proxy)
}

func AddURLProxy(proxy ProxyPtr, url *C.char) {
	(*mobileproxy.Proxy)(proxy).AddURLProxy(C.GoString(url))
}

func StopProxy(proxy ProxyPtr, timeoutSeconds C.int) {
	(*mobileproxy.Proxy)(proxy).Stop(timeoutSeconds)
}

func DestroyStreamDialer(dialer StreamDialerPtr) {
	C.free(dialer)
}

func DestroyProxy(proxy ProxyPtr) {
	C.free(proxy)
}
