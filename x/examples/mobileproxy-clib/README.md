# Mobileproxy CLib Wrapper

This is a C wrapper for the mobileproxy library. It is intended to be used downstream by native wrappers like Swift or Java.

## Demo

To run the demo, you first must build the library for your platform. You can build the library by running:

```bash
cd x

CGO_ENABLED=1 go build -buildmode=c-shared -o=examples/mobileproxy-clib/demo/mobileproxy-clib github.com/Jigsaw-Code/outline-sdk/x/examples/mobileproxy-clib
```

Then, you can build and run the demo by doing the following:

```bash
# build the demo
gcc -o examples/mobileproxy-clib/demo/demo examples/mobile
proxy-clib/demo/demo.c /Users/daniellacosse/code/outline-sdk/x/examples/mobileproxy-clib/demo/mobilep
roxy-clib

cd examples/mobileproxy-clib/demo

# run the demo
./demo
```
