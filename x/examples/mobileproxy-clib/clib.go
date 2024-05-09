package main

// #include <stdlib.h>
import (
	"C"
	"runtime/cgo"
	"unsafe"

	"github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
)

type StreamDialerPtr = unsafe.Pointer
type ProxyPtr = unsafe.Pointer

//export NewStreamDialerFromConfig
func NewStreamDialerFromConfig(config *C.char) StreamDialerPtr {
	streamDialerObject, err := mobileproxy.NewStreamDialerFromConfig(C.GoString(config))

	if err != nil {
		// TODO: print something?
		return unsafe.Pointer(nil)
	}

	streamDialerHandle := cgo.NewHandle(streamDialerObject)

	return unsafe.Pointer(&streamDialerHandle)
}

//export RunProxy
func RunProxy(address *C.char, dialer StreamDialerPtr) ProxyPtr {
	dialerObject := (*cgo.Handle)(dialer).Value().(mobileproxy.StreamDialer)

	proxyObject, err := mobileproxy.RunProxy(C.GoString(address), &dialerObject)

	if err != nil {
		// TODO: print something?
		return unsafe.Pointer(nil)
	}

	handle := cgo.NewHandle(proxyObject)

	return unsafe.Pointer(&handle)
}

//export AddURLProxy
func AddURLProxy(proxy ProxyPtr, url *C.char, dialer StreamDialerPtr) {
	proxyObject := (*cgo.Handle)(proxy).Value().(mobileproxy.Proxy)
	dialerObject := (*cgo.Handle)(dialer).Value().(mobileproxy.StreamDialer)

	proxyObject.AddURLProxy(C.GoString(url), &dialerObject)
}

//export StopProxy
func StopProxy(proxy ProxyPtr, timeoutSeconds C.uint) {
	proxyObject := (*cgo.Handle)(proxy).Value().(mobileproxy.Proxy)

	proxyObject.Stop(int(timeoutSeconds))
}

//export DeleteStreamDialer
func DeleteStreamDialer(dialer StreamDialerPtr) {
	(*cgo.Handle)(dialer).Delete()
}

//export DeleteProxy
func DeleteProxy(proxy ProxyPtr) {
	(*cgo.Handle)(proxy).Delete()
}

func main() {
	// We need the main function to make possible
	// CGO compiler to compile the package as C shared library
}
