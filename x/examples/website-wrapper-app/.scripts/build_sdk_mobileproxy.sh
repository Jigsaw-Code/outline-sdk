#!/bin/bash

PLATFORM="$1"

git clone https://github.com/Jigsaw-Code/outline-sdk.git output/outline-sdk
cd output/outline-sdk/x
go build -o "$(pwd)/out/" golang.org/x/mobile/cmd/gomobile golang.org/x/mobile/cmd/gobind

if [ "$PLATFORM" = "ios" ]; then
  echo "Building for iOS..."
  PATH="$(pwd)/out/:$PATH" gomobile bind -ldflags='-s -w' -target=ios -iosversion=11.0 -o "$(pwd)/out/mobileproxy.xcframework" github.com/Jigsaw-Code/outline-sdk/x/mobileproxy
elif [ "$PLATFORM" = "android" ]; then
  echo "Building for Android..."
  PATH="$(pwd)/out/:$PATH" gomobile bind -ldflags='-s -w' -target=android -androidapi=21 -o "$(pwd)/out/mobileproxy.aar" github.com/Jigsaw-Code/outline-sdk/x/mobileproxy
else
  echo "Invalid platform: $PLATFORM. Must be 'ios' or 'android'."
  exit 1
fi

cd ../..
rm -rf "$(pwd)/mobileproxy"
mv "$(pwd)/outline-sdk/x/out" "$(pwd)/mobileproxy"
