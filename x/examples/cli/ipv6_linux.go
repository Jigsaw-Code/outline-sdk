// Copyright 2023 The Outline Authors
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

import (
	"fmt"
	"os"
)

const disableIPv6ProcFile = "/proc/sys/net/ipv6/conf/all/disable_ipv6"

// enableIPv6 enables or disables the IPv6 support for the Linux system.
// It returns the previous setting value so the caller can restore it.
// Non-nil error means we cannot find the IPv6 setting.
func enableIPv6(enabled bool) (bool, error) {
	disabledStr, err := os.ReadFile(disableIPv6ProcFile)
	if err != nil {
		return false, fmt.Errorf("failed to read IPv6 config: %w", err)
	}
	if disabledStr[0] != '0' && disabledStr[0] != '1' {
		return false, fmt.Errorf("invalid IPv6 config value: %v", disabledStr)
	}

	prevEnabled := disabledStr[0] == '0'

	if enabled {
		disabledStr[0] = '0'
	} else {
		disabledStr[0] = '1'
	}
	if err := os.WriteFile(disableIPv6ProcFile, disabledStr, 0o644); err != nil {
		return prevEnabled, fmt.Errorf("failed to write IPv6 config: %w", err)
	}

	logging.Info.Printf("updated global IPv6 support: %v\n", enabled)
	return prevEnabled, nil
}
