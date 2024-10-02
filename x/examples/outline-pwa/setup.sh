#!/bin/bash

set -e

INDEX_HTML_LOCATION=${1:-'https://www.radiozamaneh.com/'}

mkdir -p dist
curl -s $INDEX_HTML_LOCATION > dist/index.html
npx cap sync