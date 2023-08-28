# Outline Local Proxy

This app illustrates the use of the Local Proxy type of libraries with transports that are available in SDK.
It parse `-transport` key and start a local proxy on `-addr` address using that transport.

Example:
```
KEY=ss://ENCRYPTION_KEY@HOST:PORT/
go run github.com/Jigsaw-Code/outline-sdk/x/examples/http2transport@latest -transport "$KEY" -addr localhost:54321
```
