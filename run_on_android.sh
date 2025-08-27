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

set -eu

function main() {
  declare -r host_bin="$1"
  declare -r android_bin="/data/local/tmp/test/$(basename "${host_bin}")"
  adb shell mkdir -p "$(dirname "${android_bin}")"
  adb push "${host_bin}" "${android_bin}"

  # Remove the binary name from the args
  shift 1
  adb shell "${android_bin}" "$@"

  adb shell rm "${android_bin}"
}

main "$@"
