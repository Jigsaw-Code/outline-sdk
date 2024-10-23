# Smart Dialer

## JSON config for the Smart Dialer

The Smart Dialer uses a JSON config to dynamically find serverless strategies for circumvention. The config is used to search for a strategy that unblocks DNS and TLS for a given list of test domains.

Here is an example of a JSON config:

```json
{
  "dns": [
    {
      "https": {
        "name": "doh.sb"
      }
    }
  ],
  "tls": [
    "override:host=cloudflare.net|tlsfrag:1"
  ]
}

```

### DNS Configuration

*   The `dns` field specifies a list of DNS resolvers to test.
*   Each DNS resolver can be one of the following types:
    *   `system`: Use the system resolver.
    *   `https`: Use an encrypted DNS over HTTPS (DoH) resolver.
    *   `tls`: Use an encrypted DNS over TLS (DoT) resolver.
    *   `udp`: Use a UDP resolver.
    *   `tcp`: Use a TCP resolver.

#### HTTPS Resolver

```json
{
  "https": {
    "name": "doh.sb",
    "address": "doh.sb:443"
  }
}

```

*   `name`: The domain name of the DoH server.
*   `address`: The host:port of the DoH server. Defaults to `name`:443.

#### TLS Resolver

```json
{
  "tls": {
    "name": "dns.google",
    "address": "dns.google:853"
  }
}

```

*   `name`: The domain name of the DoT server.
*   `address`: The host:port of the DoT server. Defaults to `name`:853.

#### UDP Resolver

```json
{
  "udp": {
    "address": "8.8.8.8:53"
  }
}

```

*   `address`: The host:port of the UDP resolver.

#### TCP Resolver

```json
{
  "tcp": {
    "address": "1.1.1.1:53"
  }
}

```

*   `address`: The host:port of the TCP resolver.

### TLS Configuration

*   The `tls` field specifies a list of TLS transports to test.
*   Each TLS transport is a string that specifies the transport to use.
*   For example, `override:host=cloudflare.net|tlsfrag:1` specifies a transport that uses domain fronting with Cloudflare and TLS fragmentation. The available transports are defined in the `outline-sdk/x/configurl` package.&#x20;

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
    {
      "https": {
        "name": "doh.sb"
      }
    }
  ],
  "tls": [
    "override:host=cloudflare.net|tlsfrag:1"
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