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

/*
The network package defines interfaces and provides utilities for network layer (OSI layer 3) functionalities. For
example, the [IPDevice] interface can be used to read and write IP packets from a physical or virtual network device.

In addition, user-space network stack implementations are also included in the sub-packages (such as
network/lwip2transport) that can translate raw IP packets into TCP/UDP connections. You can implement a [PacketProxy]
to handle UDP traffic, and a transport.StreamDialer to handle TCP traffic.
*/
package network
