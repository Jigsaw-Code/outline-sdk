# Outline Local Proxy

This app illustrates the use of the Local Proxy type of libraries with transports that are available in SDK.
It parse `-transport` key and start a local proxy on `-addr` address using that transport.

The main suggestion is to add a new type of libraries in SDK which will be named "proxy" and will make easy for 
developers to start local proxy using it.

Suggested interfaces are:
```Go
someproxy.NewConnectHanlder(d transport.StreamDialer)
someproxy.RunProxy(d transport.StreamDialer, addr string)
```

Example:
```
KEY=ss://ENCRYPTION_KEY@HOST:PORT/
go run github.com/Jigsaw-Code/outline-sdk/x/local-proxy@latest -transport "$KEY" -addr localhost:54321
```
