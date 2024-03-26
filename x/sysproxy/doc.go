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

// Package sysproxy provides a simple interface to set or clear system-wide proxy settings on desktop platforms.
//
// The table below summarizes the support for different proxy types on each platform:
//
//	+------------+---------+--------+--------+
//	| Proxy Type | Windows | MacOS  | Linux  |
//	+------------+---------+--------+--------+
//	| HTTP       | Yes     | Yes    | Yes    |
//	| HTTPS      | Yes     | Yes    | Yes    |
//	| SOCKS      | Yes(v4) | Yes(v5)| Yes(v5)|
//	+------------+---------+--------+--------+
//
// [SetWebProxy] implementation in this package sets up both system HTTP and HTTPS proxy settings when they are distinguished by the platform.
//
// [SetSOCKSProxy] method configures SOCKS proxy settings on the system.
//
// Support for FTP Proxy setting was not included due to lack of adoption and usage.
//
// Username and password authentication is not supported because the intended usage it to connect to a
// proxy server running on localhost.
// Please note that in SOCKS and HTTP proxy, credentials are communicated in plain text.
//
// Other notes:
//
//  1. if you only setup the system HTTP Proxy settings (for example on MacOS)
//     to point to a remote/local HTTP proxy server, and then visit a website over HTTPS,
//     your traffic does NOT go through the proxy. In essence, the system proxy
//     client only route HTTP requests through the proxy.
//
//     However, if you setup only system's HTTPS proxy settings, both HTTP and HTTPS requests
//     go through the proxy. By configuring both HTTP and HTTPS proxy settings,
//     all web traffic will be sent through the proxy.
//
//  2. If SOCKS is setup, both HTTP and HTTPS requests go through the proxy since it is performed at the TCP layer.
//     SOCKS4 (v4) proxies only support TCP traffic. On the other hand, SOCKS5 (v5) proxies support UDP protocol and
//     TCP protocol traffic, making them more versatile.
//
//  3. On Windows, The client only supports SOCKS4 spec and cannot connect to SOCKS5 proxy.
//
//  4. Windows does not explicitly distinguish between HTTP and HTTPS proxy. Also, username/password support authentication is not supported.
//
//  5. MacOS SOCKS client does not seem to correctly support authentication even though it accepts credentials [[1]].
//
//  6. On GNOME, username/password authentication is not currently supported for HTTPS proxy [[2]].
//
// # Usage
//
// To set up system-wide proxy settings, use the [SetWebProxy] or [SetSOCKSProxy] methods to connect to a Web (HTTP & HTTPS) or SOCKS proxy.
// This function takes two arguments: the IP address / hostname and the port of the proxy server.
//
// To clear system-wide proxy settings, use the [ClearWebProxy] or [ClearSOCKSProxy] function.
// This will set the address and port to "127.0.0.1:0" and disable the proxy.
//
// To ensure that the system-wide proxy settings are unset upon program termination, it is recommended to call:
//
//	defer ClearWebProxy()
//
// or
//
//	defer ClearSOCKSProxy()
//
// after the setting the proxy.
//
// The section below provides platform-specific details on how the proxy settings are configured.
//
// # macOS
//
// To configure proxy settings on macOS, we use networksetup utility with following options. To set web proxy:
//
//	networksetup -setwebproxy <networkservice> <domain> <portnumber>
//
// To set secure web proxy:
//
//	networksetup -setsecurewebproxy <networkservice> <domain> <portnumber>
//
// For more information, see the link [here].
//
// # Linux
//
// Currently only GNOME is supported. This package uses gsettings untility to setup proxy settings. The following commands are used to set proxy settings:
//
//	gsetting set org.gnome.system.proxy.http host 'proxy.example.com'
//	gsetting set org.gnome.system.proxy.http port 8080
//
// The following parameters can be set for other types of proxies:
//
//	org.gnome.system.proxy.https
//	org.gnome.system.proxy.ftp
//	org.gnome.system.proxy.socks
//
// For more information, you can checkout the documentation for [gsettings] and its [configuration].
//
// # Windows
//
// On Windows, the package uses HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings + InternetSetOptionW
// to setup proxy settings. For more information, you can checkout the documentation for [InternetSetOptionW].
//
// [here]: https://keith.github.io/xcode-man-pages/networksetup.8.html
// [gsettings]: https://github.com/GNOME/gsettings-desktop-schemas/blob/master/schemas/org.gnome.system.proxy.gschema.xml.in
// [configuration]: https://developer-old.gnome.org/ProxyConfiguration/
// [InternetSetOptionW]: https://learn.microsoft.com/en-us/windows/win32/api/wininet/nf-wininet-internetsetoptionw
// [1]: https://discussions.apple.com/thread/255394737?sortBy=best)
// [2]: https://gitlab.gnome.org/GNOME/gsettings-desktop-schemas/-/issues/42
package sysproxy
