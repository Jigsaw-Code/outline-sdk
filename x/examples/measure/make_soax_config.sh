# Copyright 2024 The Outline Authors
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

SESSION_ID=$RANDOM
cat <<EOF
num_attempts: 1
proto: tls
domains:
  # - example.com
  - google.com
strategies:
  - "tlsfrag:1"
  - "tlsfrag:2"
isp_proxies:
  - "socks5://package-${SOAX_PACKAGE_RESIDENTIAL}-sessionlength-3600-sessionid-ABC${SESSION_ID}:${SOAX_KEY_RESIDENTIAL}@proxy.soax.com:5000"
  - "socks5://package-${SOAX_PACKAGE_RESIDENTIAL}-sessionlength-3600-sessionid-ABC${SESSION_ID}:${SOAX_KEY_RESIDENTIAL}@proxy.soax.com:5000"
EOF