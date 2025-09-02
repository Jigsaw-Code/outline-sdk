# Outline Fetch

This app illustrates how to use different transports to fetch a URL in Go.

Direct fetch:

```sh
$ go run github.com/Jigsaw-Code/outline-sdk/x/tools/fetch@latest https://ipinfo.io
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
$ go run github.com/Jigsaw-Code/outline-sdk/x/tools/fetch@latest -transport ss://[redacted]@[redacted]:80 https://ipinfo.io
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
$ go run github.com/Jigsaw-Code/outline-sdk/x/tools/fetch@latest -transport socks5://[redacted]:5703 https://ipinfo.io
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
$ go run github.com/Jigsaw-Code/outline-sdk/x/tools/fetch@latest -transport split:3  https://ipinfo.io
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

## Using ECH

Pass the `-ech-config` flag with the base64-encoded ECH Config in binary format (as per the standard proposal).

```console
$ go -C x run ./tools/fetch https://test.defo.ie | grep SSL_ECH 
    <p>SSL_ECH_OUTER_SNI: NONE <br />
SSL_ECH_INNER_SNI: NONE <br />
SSL_ECH_STATUS: not attempted <img src="redx-small.png" alt="bummer" /> <br/>

$ dig +short test.defo.ie HTTPS
1 . ech=AEb+DQBCqQAgACBlm7cfDx/gKuUAwRTe+Y9MExbIyuLpLcgTORIdi69uewAEAAEAAQATcHVibGljLnRlc3QuZGVmby5pZQAA

$ go -C x run ./tools/fetch --ech-config=AEb+DQBCqQAgACBlm7cfDx/gKuUAwRTe+Y9MExbIyuLpLcgTORIdi69uewAEAAEAAQATcHVibGljLnRlc3QuZGVmby5pZQAA https://test.defo.ie | grep SSL_ECH
    <p>SSL_ECH_OUTER_SNI: public.test.defo.ie <br />
SSL_ECH_INNER_SNI: test.defo.ie <br />
SSL_ECH_STATUS: success <img src="greentick-small.png" alt="good" /> <br/>
```
