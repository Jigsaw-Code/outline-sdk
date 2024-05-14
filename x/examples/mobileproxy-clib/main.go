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
// #include <stdint.h>
import (
	"C"
	"runtime/cgo"

	"github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
)

type StreamDialerPtr = C.uintptr_t
type ProxyPtr = C.uintptr_t

//export NewStreamDialerFromConfig
func NewStreamDialerFromConfig(config *C.char) StreamDialerPtr {
	streamDialerObject, err := mobileproxy.NewStreamDialerFromConfig(C.GoString(config))

	if err != nil {
		// TODO: return error
		return C.uintptr_t(nil)
	}

	streamDialerHandle := cgo.NewHandle(streamDialerObject)

	return C.uintptr_t(&streamDialerHandle)
}

//export RunProxy
func RunProxy(address *C.char, dialer StreamDialerPtr) ProxyPtr {
	dialerObject := cgo.Handle(dialer).Value().(mobileproxy.StreamDialer)

	proxyObject, err := mobileproxy.RunProxy(C.GoString(address), &dialerObject)

	if err != nil {
		// TODO: return error
		return C.uintptr_t(nil)
	}

	handle := cgo.NewHandle(proxyObject)

	return C.uintptr_t(&handle)
}

//export AddURLProxy
func AddURLProxy(proxy ProxyPtr, url *C.char, dialer StreamDialerPtr) {
	proxyObject := cgo.Handle(proxy).Value().(mobileproxy.Proxy)
	dialerObject := cgo.Handle(dialer).Value().(mobileproxy.StreamDialer)

	proxyObject.AddURLProxy(C.GoString(url), &dialerObject)
}

//export StopProxy
func StopProxy(proxy ProxyPtr, timeoutSeconds C.uint) {
	proxyObject := cgo.Handle(proxy).Value().(mobileproxy.Proxy)

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
