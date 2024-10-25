# Smart Dialer

The **Smart Dialer** searches for a strategy that unblocks DNS and TLS for a given list of test domains. It takes a config describing multiple strategies to pick from.

## JSON config for the Smart Dialer

The config that the Smart Dialer takes is in a JSON format. Here is an example:

```json
{
  "dns": [
      {"system": {}},
      {"https": {"name": "8.8.8.8"}},
      {"https": {"name": "9.9.9.9"}}
  ],
  "tls": [
      "",
      "split:2",
      "tlsfrag:1"
  ]
}
```

### DNS Configuration

*   The `dns` field specifies a list of DNS resolvers to test.
*   Each DNS resolver can be one of the following types:
    *   `system`: Use the system resolver. Specify with an empty object.
    *   `https`: Use an encrypted DNS over HTTPS (DoH) resolver.
    *   `tls`: Use an encrypted DNS over TLS (DoT) resolver.
    *   `udp`: Use a UDP resolver.
    *   `tcp`: Use a TCP resolver.

#### DNS-over-HTTPS Resolver (DoH)

```json
{
  "https": {
    "name": "dns.google",
    "address": "8.8.8.8"
  }
}

```

*   `name`: The domain name of the DoH server.
*   `address`: The host:port of the DoH server. Defaults to `name`:443.

#### DNS-over-TLS Resolver (DoT)

```json
{
  "tls": {
    "name": "dns.google",
    "address": "8.8.8.8"
  }
}
```

*   `name`: The domain name of the DoT server.
*   `address`: The host:port of the DoT server. Defaults to `name`:853.

#### UDP Resolver

```json
{
  "udp": {
    "address": "8.8.8.8"
  }
}
```

*   `address`: The host:port of the UDP resolver.

#### TCP Resolver

```json
{
  "tcp": {
    "address": "8.8.8.8"
  }
}
```

*   `address`: The host:port of the TCP resolver.

### TLS Configuration

*   The `tls` field specifies a list of TLS transports to test.
*   Each TLS transport is a string that specifies the transport to use.
*   For example, `override:host=cloudflare.net|tlsfrag:1` specifies a transport that uses domain fronting with Cloudflare and TLS fragmentation. See the [config documentation](https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk/x/config#hdr-Config_Format) for details.

### Using the Smart Dialer

To use the Smart Dialer, create a `StrategyFinder` object and call the `NewDialer` method, passing in the list of test domains and the JSON config. The `NewDialer` method will return a `transport.StreamDialer` that can be used to create connections using the found strategy. For example:

```go
finder := &smart.StrategyFinder{
    TestTimeout:  5 * time.Second,
    LogWriter:   os.Stdout,
    StreamDialer: &transport.TCPDialer{},
    PacketDialer: &transport.UDPDialer{},
}

configBytes := []byte(`
{
  "dns": [
      {"system": {}},
      {"https": {"name": "8.8.8.8"}},
      {"https": {"name": "9.9.9.9"}}
  ],
  "tls": [
      "",
      "split:2",
      "tlsfrag:1"
  ]
}
`)

dialer, err := finder.NewDialer(context.Background(), []string{"www.google.com"}, configBytes)
if err != nil {
    // Handle error.
}

// Use dialer to create connections.
```

Please note that this is a basic example and may need to be adapted for your specific use case.
