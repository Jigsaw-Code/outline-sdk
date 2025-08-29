#!/bin/sh

# From https://medium.com/using-go-in-mobile-apps/using-go-in-mobile-apps-part-2-building-an-ios-app-with-go-build-eb1fc3b56c99

# This uses the latest available iOS SDK, which is recommended.
# To select a specific SDK, run ‘xcodebuild -showsdks’
# to see the available SDKs and replace iphoneos with one of them.
SDK=iphoneos
SDK_PATH="$(xcrun --sdk "${SDK}" --show-sdk-path)"
# export IPHONEOS_DEPLOYMENT_TARGET=7.0
# cmd/cgo doesn’t support llvm-gcc-4.2, so we have to use clang.
CLANG="$(xcrun --sdk iphoneos --find clang)"
#  -arch armv7 -isysroot "$SDK_PATH" "$@"
exec "$CLANG" -isysroot "$SDK_PATH" "$@"