# Mobileproxy CLib Wrapper

This is a C wrapper for the mobileproxy library. It is intended to be used downstream by native wrappers like Swift or Java.

## Demo

To run the demo, you first must build the library for your platform. You can build it by running:

```bash
cd x

CGO_ENABLED=1 go build -buildmode=c-shared -o=examples/mobileproxy-clib/demo/bin github.com/Jigsaw-Code/outline-sdk/x/examples/mobileproxy-clib
```

Then, you can run the demo by running:

```bash
TODO
```
