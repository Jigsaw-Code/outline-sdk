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
  declare -r android_run_dir="/data/local/tmp/run/$(basename "${host_bin}")"
  # Set up cleanup to run whenever the script exits. `adb push` creates the directory.
  trap "adb shell rm -r '${android_run_dir}'" EXIT
  declare -r android_bin="${android_run_dir}/bin"
  adb push "${host_bin}" "${android_bin}"

  declare -r testdata_dir="$(pwd)/testdata"
  if [[ "${host_bin##*.}" = "test" && -d "${testdata_dir}" ]]; then
    adb push "${testdata_dir}" "${android_run_dir}/testdata"
  fi

  # Remove the binary name from the args
  shift 1
  adb shell cd "${android_run_dir}"";" ./bin "$@"
}

main "$@"
