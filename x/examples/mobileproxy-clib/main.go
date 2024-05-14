// Copyright 2024 Jigsaw Operations LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
