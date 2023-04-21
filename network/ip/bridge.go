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

// An IPBridge forwards IP traffic bidirectionally between two IPDevices.
type IPBridge struct {
}

// NewIPBridge returns a new IPBridge instance that can be used to bridge the
// traffic between the devices `dev1` and `dev2`. Forwarding will not start
// automatically, you must call the Open() function on the returned instance to
// start forwarding traffic bidirectionally.
func NewIPBridge(dev1, dev2 IPDevice) (*IPBridge, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// Open starts forwarding IP traffic between the two devices that were passed
// to the NewIPBridge function as the `dev1` and `dev2` parameters. To stop the
// traffic forwarding, call the Destroy method.
func (br *IPBridge) Open() error {
	return fmt.Errorf("not implemented yet")
}

// Destroy stops forwarding IP traffic, it won't Close the related IPDevices.
// This means that you can call the Open method again to start forwarding
// traffic.
func (br *IPBridge) Destroy() error {
	return fmt.Errorf("not implemented yet")
}
