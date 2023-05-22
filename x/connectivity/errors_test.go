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

package connectivity

import (
	"fmt"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

func TestErrnoName(t *testing.T) {
	require.Equal(t, "ECONNREFUSED", errnoName(windows.WSAECONNREFUSED))
}

func langID(pri, sub uint16) uint32 { return uint32(sub)<<10 | uint32(pri) }

func TestErrnoNameWithFormat(t *testing.T) {
	b := make([]uint16, 300)
	errno := windows.ERROR_FILE_NOT_FOUND
	n, err := windows.FormatMessage(
		windows.FORMAT_MESSAGE_FROM_SYSTEM|
			windows.FORMAT_MESSAGE_IGNORE_INSERTS,
		0, uint32(errno), langID(0, 0), b, nil)
	var text string
	if err != nil {
		text = fmt.Sprintf("NTSTATUS 0x%08x", uint32(errno))
	}
	// trim terminating \r and \n
	for ; n > 0 && (b[n-1] == '\n' || b[n-1] == '\r'); n-- {
	}
	text = string(utf16.Decode(b[:n]))
	require.Equal(t, "ECONNREFUSED", text)
}
