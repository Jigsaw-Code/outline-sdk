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

# This is currently failing on CI:
#
# + declare -r host_bin=/tmp/go-build323420225/b001/dns.test
# ++ basename /tmp/go-build323420225/b001/dns.test
# + declare -r android_bin=/data/local/tmp/test/dns.test
# + adb push /tmp/go-build323420225/b001/dns.test /data/local/tmp/test/dns.test
# /tmp/go-build323420225/b001/dns.test: 1 file pushed, 0 skipped. 119.2 MB/s (6852900 bytes in 0.055s)
# + shift 1
# + adb shell chmod +x /data/local/tmp/test/dns.test
# + adb shell /data/local/tmp/test/dns.test -test.paniconexit0 -test.timeout=10m0s -test.v=true
# /system/bin/sh: /data/local/tmp/test/dns.test: No such file or directory
# FAIL	github.com/Jigsaw-Code/outline-sdk/dns	0.713s
#
# It's unclear why the binary file is not being found, despite the successful adb push.

set -eu

function main() {
  set -x
  declare -r host_bin="$1"
  declare -r android_bin="/data/local/tmp/test/$(basename "${host_bin}")"
  adb push "${host_bin}" "${android_bin}"

  # Remove the binary name from the args
  shift 1
  adb shell chmod +x "${android_bin}"
  adb shell "${android_bin}" "$@"

  adb shell rm "${android_bin}"
}

main "$@"
