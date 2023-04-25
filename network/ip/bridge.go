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

package ip

import "fmt"

// Bridge forwards IP traffic bidirectionally between two IPDevices, `dev1` and
// `dev2`, until an error or EOF occur.
//
// A successful Bridge function returns err == nil, not err == EOF. It does not
// consider EOF from either device to be an error.
//
// Once Bridge has started, it cannot be interrupted or "un-bridged". The only
// way to stop forwarding traffic is to make one of the devices return either
// error or EOF.
func Bridge(dev1, dev2 IPDevice) error {
	return fmt.Errorf("not implemented yet")
}
