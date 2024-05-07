package main

// #include <stdlib.h>
import (
	"C"
	"runtime/cgo"
	"unsafe"

	"github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
)

//export NewStreamDialerFromConfig
func NewStreamDialerFromConfig(config *C.char) unsafe.Pointer {
	streamDialer, err := mobileproxy.NewStreamDialerFromConfig(C.GoString(config))

	if err != nil {
		// TODO: print something?
		return unsafe.Pointer(nil)
	}

	handle := cgo.NewHandle(streamDialer)

	return unsafe.Pointer(&handle)
}

//export RunProxy
func RunProxy(address *C.char, dialerHandlerPtr unsafe.Pointer) unsafe.Pointer {
	dialer := (*cgo.Handle)(dialerHandlerPtr).Value().(mobileproxy.StreamDialer)

	proxy, err := mobileproxy.RunProxy(C.GoString(address), &dialer)

	if err != nil {
		// TODO: print something?
		return unsafe.Pointer(nil)
	}

	handle := cgo.NewHandle(proxy)

	return unsafe.Pointer(&handle)
}

//export AddURLProxy
func AddURLProxy(proxyHandlerPtr unsafe.Pointer, url *C.char, dialerHandlerPtr unsafe.Pointer) {
	proxy := (*cgo.Handle)(proxyHandlerPtr).Value().(mobileproxy.Proxy)
	dialer := (*cgo.Handle)(dialerHandlerPtr).Value().(mobileproxy.StreamDialer)

	proxy.AddURLProxy(C.GoString(url), &dialer)
}

//export StopProxy
func StopProxy(proxyHandlerPtr unsafe.Pointer, timeoutSeconds C.uint) {
	proxy := (*cgo.Handle)(proxyHandlerPtr).Value().(mobileproxy.Proxy)

	proxy.Stop(int(timeoutSeconds))
}

//export DeleteStreamDialer
func DeleteStreamDialer(dialerHandlerPtr unsafe.Pointer) {
	(*cgo.Handle)(dialerHandlerPtr).Delete()
}

//export DeleteProxy
func DeleteProxy(proxyHandlerPtr unsafe.Pointer) {
	(*cgo.Handle)(proxyHandlerPtr).Delete()
}

func main() {
	// We need the main function to make possible
	// CGO compiler to compile the package as C shared library
}
