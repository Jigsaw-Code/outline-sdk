#!/bin/bash

# Copyright 2025 The Outline Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

PLATFORM="$1"
TAG="${2:-"x/v0.0.2"}"

git clone --depth 1 --branch "${TAG}" https://github.com/Jigsaw-Code/outline-sdk.git output/outline-sdk
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
