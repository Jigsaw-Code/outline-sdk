# Outline Fetch

This app illustrates how to use different transports to fetch a URL in Go.

Direct fetch:

```sh
$ go run github.com/Jigsaw-Code/outline-sdk/x/examples/fetch@latest https://ipinfo.io
{
  ...
  "city": "Amsterdam",
  "region": "North Holland",
  "country": "NL",
  ...
}                                  
```

Using a Shadowsocks server:

```sh
$ go run github.com/Jigsaw-Code/outline-sdk/x/examples/fetch@latest -transport ss://[redacted]@[redacted]:80 https://ipinfo.io
{
  ...
  "region": "New Jersey",
  "country": "US",
  "org": "AS14061 DigitalOcean, LLC",
  ...
}
```

Using a SOCKS5 server:

```sh
$ go run github.com/Jigsaw-Code/outline-sdk/x/examples/fetch@latest -transport socks5://[redacted]:5703 https://ipinfo.io
{
  ... 
  "city": "Berlin",
  "region": "Berlin",
  "country": "DE",
  ...
}
```

Using packet splitting:

```sh
$ go run github.com/Jigsaw-Code/outline-sdk/x/examples/fetch@latest -transport split:3  https://ipinfo.io
{
  ...
  "city": "Amsterdam",
  "region": "North Holland",
  "country": "NL",
  ...
}                                  
```

You should see this on Wireshark:

<img width="652" alt="image" src="https://github.com/Jigsaw-Code/outline-sdk/assets/113565/9c19667d-d0fb-4d33-b0a6-275674481dce">

