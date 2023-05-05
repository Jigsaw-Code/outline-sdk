# Outline Connectivity Test

This app illustrates the use of the Shadowsocks transport to resolve a domain name over TCP or UDP.

Example:
```
go run github.com/Jigsaw-Code/outline-internal-sdk/x/cmd/outline-connectivity@latest -key='ss://ENCRYPTION_KEY@HOST:PORT/[&prefix=PREFIX]' [-v]
```