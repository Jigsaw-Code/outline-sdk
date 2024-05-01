package clib

// #include <stdlib.h>
import (
	"C"
	"runtime/cgo"

	"github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
)

type ProxyHandle = cgo.Handle
type StreamDialerHandle = cgo.Handle

//export NewStreamDialerFromConfig
func NewStreamDialerFromConfig(config *C.char) StreamDialerHandle {
	streamDialer, err := mobileproxy.NewStreamDialerFromConfig(C.GoString(config))

	if err != nil {
		// TODO: print something?
		return cgo.NewHandle(nil)
	}

	return cgo.NewHandle(streamDialer)
}

//export RunProxy
func RunProxy(address *C.char, dialerHandle unsafe.Pointer) unsafe.Pointer {
	dialer := dialerHandle.Value().(mobileproxy.StreamDialer)

	proxy, err := mobileproxy.RunProxy(C.GoString(address), &dialer)

	if err != nil {
		// TODO: print something?
		return cgo.NewHandle(nil)
	}

	return cgo.NewHandle(proxy)
}

//export AddURLProxy
func AddURLProxy(proxyHandle ProxyHandle, url *C.char) {
	proxy := proxyHandle.Value().(mobileproxy.Proxy)

	proxy.AddURLProxy(C.GoString(url))
}

//export StopProxy
func StopProxy(proxyHandle ProxyHandle, timeoutSeconds C.uint) {
	proxy := proxyHandle.Value().(mobileproxy.Proxy)

	proxy.Stop(timeoutSeconds)
}

//export DestroyStreamDialer
func DestroyStreamDialer(dialer StreamDialerHandle) {
	dialer.Delete()
}

//export DestroyProxy
func DestroyProxy(proxy ProxyHandle) {
	proxy.Delete()
}
