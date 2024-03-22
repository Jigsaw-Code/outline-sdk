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

And you will see the response.
