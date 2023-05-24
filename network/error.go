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
	"fmt"
	"os"
)

// Portable analogs of some common errors.
//
// Errors returned from this package and all sub-packages may be tested against these errors with [errors.Is].
var (
	// ErrClosed is the error returned by an I/O call on a network device that has already been closed, or that is closed
	// by another goroutine before the I/O is completed. This may be wrapped in another error, and should normally be
	// tested using errors.Is(err, network.ErrClosed).
	ErrClosed = fmt.Errorf("the network device is closed: %w", os.ErrClosed)

	// ErrMsgSize is the error returned by an [io.Write] on a network device that the size of the message to be sent is
	// bigger than the maximum message size the device can process.
	ErrMsgSize = errors.New("packet size is too big")
)
