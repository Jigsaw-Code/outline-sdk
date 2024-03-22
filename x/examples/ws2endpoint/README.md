# Websocket Reverse Proxy

This package contains a command-line tool to expose a Websocket endpoint that connects to
any endpoint over a transport.

```sh
go run ./examples/ws2endpoint --endpoint ipinfo.io:443 --transport tls
```

Then, on a browser console, you can do:

```js
s = new WebSocket("ws://localhost:8080");
s.onmessage = (m) => console.log(m.data);
s.onopen = () => { s.send("GET /json HTTP/1.1\r\nHost: ipinfo.io\r\n\r\n"); }
```

And you will see the response. For example:

```http
HTTP/1.1 200 OK
server: nginx/1.24.0
date: Fri, 22 Mar 2024 22:07:18 GMT
content-type: application/json; charset=utf-8
Content-Length: 321
access-control-allow-origin: *
x-content-type-options: nosniff
x-envoy-upstream-service-time: 3
via: 1.1 google
strict-transport-security: max-age=2592000; includeSubDomains
Alt-Svc: h3=":443"; ma=2592000,h3-29=":443"; ma=2592000

{
  "ip": "[REDACTED]",
  "hostname": "[REDACTED]",
  "city": "New York City",
  "region": "New York",
  "country": "US",
  ...
  "timezone": "America/New_York",
  "readme": "https://ipinfo.io/missingauth"
}
```
