# WebSocket Reverse Proxy

This package contains a command-line tool to expose a WebSocket endpoint that connects to
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

You can expose your WebSockets on Cloudflare with [clourdflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/do-more-with-tunnels/trycloudflare/). For example:

```console
% cloudflared tunnel --url http://localhost:8080

2024-03-22T22:06:27Z INF Thank you for trying Cloudflare Tunnel. Doing so, without a Cloudflare account, is a quick way to experiment and try it out. However, be aware that these account-less Tunnels have no uptime guarantee. If you intend to use Tunnels in production you should use a pre-created named tunnel by following: https://developers.cloudflare.com/cloudflare-one/connections/connect-apps
2024-03-22T22:06:27Z INF Requesting new quick Tunnel on trycloudflare.com...
2024-03-22T22:06:28Z INF +--------------------------------------------------------------------------------------------+
2024-03-22T22:06:28Z INF |  Your quick Tunnel has been created! Visit it at (it may take some time to be reachable):  |
2024-03-22T22:06:28Z INF |  https://recorders-uganda-starring-stopping.trycloudflare.com                              |
2024-03-22T22:06:28Z INF +--------------------------------------------------------------------------------------------+
```

In this case, use `wss://recorders-uganda-starring-stopping.trycloudflare.com` as the WebSocket url.


Note that the Cloudflare tunnel does not add any user authentication mechanism. You must implement authentication yourself
if you would like to prevent unauthorized access to your service.
