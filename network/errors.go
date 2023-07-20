// Copyright 2023 Jigsaw Operations LLC
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

package network

import (
	"errors"
)

// Portable analogs of some common errors.
//
// Errors returned from this package and all sub-packages may be tested against these errors with [errors.Is].
var (
	// ErrClosed is the error returned by an I/O call on a network device or proxy that has already been closed, or that is
	// closed by another goroutine before the I/O is completed. This may be wrapped in another error, and should normally
	// be tested using errors.Is(err, network.ErrClosed).
	ErrClosed = errors.New("network device already closed")

	// ErrUnsupported indicates that a requested stream or packet cannot be handled by the network device, because it is
	// unsupported. For example, when you call dnstruncate.NewPacketProxy().WriteTo() with a non-DNS request packet.
	//
	// The message of this error is "traffic is not supported". So functions should typically wrap this error in another
	// error before returning it (for example, using fmt.Errorf to construct a full message):
	//
	//	 fmt.Errorf("non-DNS UDP %w", ErrUnsupported) // final error message: "Non-DNS UDP traffic is not supported"
	//
	// Types in the network package may check against this error using:
	//
	//   errors.Is(err, ErrUnsupported)
	//
	// And they will switch to a fallback behavior if some functions (e.g. WriteTo) returns this error.
	ErrUnsupported = errors.New("traffic is not supported")
)
