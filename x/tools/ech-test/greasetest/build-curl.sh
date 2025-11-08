#!/bin/bash

set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <output_dir>"
  exit 1
fi

OUTPUT_DIR=$(realpath "$1")
WORKSPACE_DIR=$(mktemp -d)

echo "Using workspace directory: ${WORKSPACE_DIR}"
echo "Using output directory: ${OUTPUT_DIR}"

echo "Cloning OpenSSL..."
git clone --filter=blob:none https://github.com/defo-project/openssl "${WORKSPACE_DIR}/openssl"
cd "${WORKSPACE_DIR}/openssl"

echo "Configuring and building OpenSSL..."
./config --libdir=lib --prefix="${OUTPUT_DIR}"
make -j$(nproc)
make install_sw

echo "Cloning curl..."
git clone --filter=blob:none https://github.com/defo-project/curl "${WORKSPACE_DIR}/curl"
cd "${WORKSPACE_DIR}/curl"

echo "Configuring and building curl..."
autoreconf -fi
./configure --with-openssl="${OUTPUT_DIR}" --prefix="${OUTPUT_DIR}" --enable-ech
make
make install

echo "Cleaning up workspace..."
rm -rf "${WORKSPACE_DIR}"

echo "Done. curl with ECH support is installed in ${OUTPUT_DIR}/bin"
