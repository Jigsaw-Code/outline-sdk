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

//go:build unix

package smart

/*
#include <stdlib.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netdb.h>
*/
import "C"

import (
	"context"
	"fmt"
	"unsafe"
)

// lookupCNAME provides functionality equivalent to net.DefaultResolver.LookupCNAME. However,
// net.DefaultResolver.LookupCNAME uses libresolv on unix, and, on Android and iOS, it tries
// to connect to [::1]:53 (probably from /etc/resolv.conf) and the connection is refused.
// Instead, we use getaddrinfo to get the canonical name.
func lookupCNAME(ctx context.Context, domain string) (string, error) {
	type result struct {
		cname string
		err   error
	}

	results := make(chan result)
	go func() {
		cname, err := lookupCNAMEBlocking(domain)
		results <- result{cname, err}
	}()

	select {
	case r := <-results:
		return r.cname, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func lookupCNAMEBlocking(host string) (string, error) {
	var hints C.struct_addrinfo
	var result *C.struct_addrinfo

	chost := C.CString(host)
	defer C.free(unsafe.Pointer(chost))

	hints.ai_family = C.AF_UNSPEC
	hints.ai_flags = C.AI_CANONNAME

	res := C.getaddrinfo(chost, nil, &hints, &result)
	if res != 0 {
		return "", fmt.Errorf("getaddrinfo error: %s", C.GoString(C.gai_strerror(res)))
	}
	defer C.freeaddrinfo(result)

	// Extract canonical name
	cname := C.GoString(result.ai_canonname)
	return cname, nil
}
