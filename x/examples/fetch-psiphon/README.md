# Using the Psiphon StreamDialer example

This fetch tool illustrates how to use Psiphon as a stream dialer.

Usage:

```sh
go run -tags PSIPHON_DISABLE_QUIC . -config config.json https://ipinfo.io
```

You will need a config file of a Psiphon server. You can run one yourself and generate the config as per the
[official instructions](https://github.com/Psiphon-Labs/psiphon-tunnel-core/tree/master#generate-configuration-data),
or obtain a server from the Psiphon team.
