# Platform support

The table below summarizes system-wide support this package offers for various types of proxies on desktop platforms.

| Proxy Type | Windows | MacOS | Linux |
| --- | ----------- | ------ | ------ |
| HTTP | Yes | Yes | Yes
| HTTPS | Yes | Yes | Yes
| SOCKS | No | Yes | Yes


`SetupWebProxy` implementation in this package setups both HTTP and HTTPS proxy when they distinguished by the platform. 

For example, on MacOS, if you have HTTP Proxy setup and visit a website over HTTPS, your traffic does NOT go through the proxy. 

However, if only HTTPS proxy is setup, both HTTP and HTTPS requests go through the HTTPS proxy.

If SOCKS is setup, both HTTP and HTTPS requests go through the proxy since it is performed at the TCP layer. SOCKS does not support UDP.

Support for FTP Proxy setting was not included due lack of adoption and usage. Username and password authentication was not included due to potential unreliability and untestd behavior. Please note that in SOCKS and HTTP proxy, credentials are communicated in plain text.

If you have a need for any of those, feel free to open an issue and let me know about the use case.

## Other notes

1. Windows does not explicitly distinguish between HTTP and HTTPS proxy. Also, username/password support authentication is not supported.

2. MacOS SOCKS client does not seems to correctly support authentication even thoguh it accepts credentials [[ref](https://discussions.apple.com/thread/255394737?sortBy=best)].

3. On GNOME, username/Pass authenticartion is not currently supported for HTTPS proxy [[ref](https://gitlab.gnome.org/GNOME/gsettings-desktop-schemas/-/issues/42)].