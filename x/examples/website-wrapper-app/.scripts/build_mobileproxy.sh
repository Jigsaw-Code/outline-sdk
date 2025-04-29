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
OUTPUT="${2:-$(pwd)/output}"

if [[ "$OUTPUT" = "/" ]] || [[ "$OUTPUT" = "*" ]]; then
  echo "Error: OUTPUT cannot be '/' or '*'. These are dangerous values."
  exit 1
fi

mkdir -p "$OUTPUT/mobileproxy"

go build -o "$OUTPUT/mobileproxy" golang.org/x/mobile/cmd/gomobile golang.org/x/mobile/cmd/gobind

if [[ "$PLATFORM" = "ios" ]]; then
  echo "Building for iOS..."
  PATH="$OUTPUT/mobileproxy/:$PATH" gomobile bind -ldflags='-s -w' -target=ios -iosversion=11.0 -o "$OUTPUT/mobileproxy/mobileproxy.xcframework" github.com/Jigsaw-Code/outline-sdk/x/mobileproxy
elif [[ "$PLATFORM" = "android" ]]; then
  echo "Building for Android..."
  PATH="$OUTPUT/mobileproxy/:$PATH" gomobile bind -ldflags='-s -w' -target=android -androidapi=21 -o "$OUTPUT/mobileproxy/mobileproxy.aar" github.com/Jigsaw-Code/outline-sdk/x/mobileproxy
else
  echo "Invalid platform: $PLATFORM. Must be 'ios' or 'android'."
  exit 1
fi
