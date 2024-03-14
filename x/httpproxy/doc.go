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

/*
Package httpproxy provides HTTP handlers for routing HTTP traffic through a local web proxy.

# Important Security Considerations

This package is designed primarily for use with private, internal forward proxies typically integrated within an application.
It is not suitable for public-facing proxies due to the following security concerns:

  - Authentication: Public proxies must restrict access to only authorized users. This package does not provide built-in authentication mechanisms.
  - Probing Resistance: A public proxy should ideally not reveal its identity as a proxy, even under targeted probing. Implementing authentication can aid in this.
  - Protection of Local Resources: The dialer used by the proxy handlers should prevent connections to both localhost and the local network to avoid unintended access by clients.
  - Resource Limits:  Implement limits on resources (number of connections, time connected, memory used, etc.) per user.  This helps prevent denial-of-service attacks.

If you intend to build a public-facing proxy, you will need to address these security issues using additional libraries or custom solutions.
*/
package httpproxy
