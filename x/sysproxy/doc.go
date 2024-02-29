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
Package sysproxy provides a simple interface to set/unset system-wide proxy settings.

# Platform Support

Currently this package supports desktop platforms only. The following platforms are supported:
  - macOS
  - Linux (Gnome)
  - Windows

# Usage

To set up system-wide proxy settings, use the [SetProxy] function. This function takes two arguments: the IP address and the port of the proxy server.

To unset system-wide proxy settings, use the [UnsetProxy] function.
To ensure that the system-wide proxy settings are unset upon program termination, it is recommended to call:

	defer UnsetProxy()

when the application starts.
*/
package sysproxy
