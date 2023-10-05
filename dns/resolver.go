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

package dns

/*
	A resolver is a function that resolves a hostname to a list of IP addresses.

	TODO: provide factory methods to create a Resolver. For example:

	```go
		manualDns := makeManualResolver(
			map[string][]string{ "example.com": []string{"255.255.255.255:80" }
		})	
	
		dialer := &shadowsocks.NewStreamDialer(endpoint, key, manualDns)
	````
*/
type Resolver func(host string) ([]string, error)