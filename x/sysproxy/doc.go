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

# macOS

To configure proxy settings on macOS, we use networksetup utility with following options. To set web proxy:

	networksetup -setwebproxy <networkservice> <domain> <portnumber>

To set secure web proxy:

	networksetup -setsecurewebproxy <networkservice> <domain> <portnumber>

For more information, see the link [here].

# Linux

Currently only GNOME is supported. This package uses gsettings untility to setup proxy settings. The following commands are used to set proxy settings:

	gsetting set org.gnome.system.proxy.http host 'proxy.example.com'
	gsetting set org.gnome.system.proxy.http port 8080

The following parameters can be set for other types of proxies:

	org.gnome.system.proxy.https
	org.gnome.system.proxy.ftp
	org.gnome.system.proxy.socks

For more information, you can checkout the documentation for [gsettings] and its [configuration].

# Windows

On Windows, the package uses HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings + InternetSetOptionW
to setup proxy settings. For more information, you can checkout the documentation for [InternetSetOptionW].

# Usage

To set up system-wide proxy settings, use the [SetProxy] function. This function takes two arguments: the IP address and the port of the proxy server.

To unset system-wide proxy settings, use the [UnsetProxy] function.
To ensure that the system-wide proxy settings are unset upon program termination, it is recommended to call:

	defer UnsetProxy()

after the SetProxy call.

[here]: https://keith.github.io/xcode-man-pages/networksetup.8.html
[gsettings]: https://github.com/GNOME/gsettings-desktop-schemas/blob/master/schemas/org.gnome.system.proxy.gschema.xml.in
[configuration]: https://developer-old.gnome.org/ProxyConfiguration/
[InternetSetOptionW]: https://learn.microsoft.com/en-us/windows/win32/api/wininet/nf-wininet-internetsetoptionw
*/
package sysproxy
