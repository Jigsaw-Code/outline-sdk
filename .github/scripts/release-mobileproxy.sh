#!/usr/bin/env bash
set -euo pipefail

# This script builds the mobileproxy for iOS and Android.
# It clones the repository at a specific tag, builds the artifacts, and uploads them to a GitHub release.
# Expected environment variables:
# - RELEASE_TAG (e.g., "v1.0.0" or "x/v0.0.3"). Defaults to the first script argument if not set.
# - GH_TOKEN (for cloning private repos if necessary, and for gh release upload). Upload is skipped if not set.
# - OUTPUT_DIR (absolute path for build artifacts). Defaults to "${PWD}/out" if not set.
#
# Usage Examples:
#
# 1. Build for a specific tag (passed as argument), output to default "./out", no upload:
#    ./release-mobileproxy.sh x/v1.0.0
#
# 2. Build for a tag from environment, output to default "./out", and upload artifacts:
#    RELEASE_TAG="x/v1.0.1" GH_TOKEN="your_github_token" ./release-mobileproxy.sh
#
# 3. Build for a tag (passed as argument), specify output directory, and upload:
#    GH_TOKEN="your_github_token" OUTPUT_DIR="/tmp/my_builds" ./release-mobileproxy.sh x/v1.0.2
#
# 4. Build for a tag from environment, specify output directory, no upload (GH_TOKEN not provided):
#    RELEASE_TAG="x/v1.0.3" OUTPUT_DIR="/tmp/another_build" ./release-mobileproxy.sh
#
# Note: `gh` CLI must be installed and in PATH if GH_TOKEN is provided for uploads.

if [[ -z "${OUTPUT_DIR:-}" ]]; then
  OUTPUT_DIR="${PWD}/out"
fi

if [[ -z "${RELEASE_TAG:-}" ]]; then
  if [[ -z "${1:-}" ]]; then
    echo "ERROR: RELEASE_TAG environment variable is not set and no tag argument provided."
    exit 1
  fi
  RELEASE_TAG="$1"
fi

CLONE_DIR=$(mktemp -d)
trap "echo 'Cleaning up temporary clone directory: ${CLONE_DIR}'; rm -rf '${CLONE_DIR}'" EXIT

echo "Cloning repository at tag ${RELEASE_TAG} into ${CLONE_DIR}..."
CLONE_URL_BASE="github.com/Jigsaw-Code/outline-sdk.git"
if [[ -n "${GH_TOKEN:-}" ]]; then
  CLONE_URL="https://x-access-token:${GH_TOKEN}@${CLONE_URL_BASE}"
else
  CLONE_URL="https://%s@${CLONE_URL_BASE}"
fi
git clone --depth 1 --branch "${RELEASE_TAG}" "${CLONE_URL}" "${CLONE_DIR}"

X_DIR_IN_CLONE="${CLONE_DIR}/x"
MOBILEPROXY_BUILD_TOOLS_DIR="${OUTPUT_DIR}/mobileproxy_build_tools"
MOBILEPROXY_ARTIFACT_DIR="${OUTPUT_DIR}/mobileproxy"

mkdir -p "${MOBILEPROXY_BUILD_TOOLS_DIR}"
mkdir -p "${MOBILEPROXY_ARTIFACT_DIR}"

echo "Building gomobile and gobind from cloned tag ${RELEASE_TAG}..."
(
  cd "${X_DIR_IN_CLONE}"
  go build -o "${MOBILEPROXY_BUILD_TOOLS_DIR}/gomobile" golang.org/x/mobile/cmd/gomobile
  go build -o "${MOBILEPROXY_BUILD_TOOLS_DIR}/gobind" golang.org/x/mobile/cmd/gobind
)

export PATH="${MOBILEPROXY_BUILD_TOOLS_DIR}:${PATH}"

IOS_FRAMEWORK_PATH="${MOBILEPROXY_ARTIFACT_DIR}/mobileproxy.xcframework"
echo "Building Mobileproxy for iOS..."
(
  cd "${X_DIR_IN_CLONE}"
  gomobile bind -ldflags='-s -w' -target=ios -iosversion=11.0  -o "${IOS_FRAMEWORK_PATH}" \
    "github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
)

IOS_ZIP_PATH="${MOBILEPROXY_ARTIFACT_DIR}/mobileproxy.xcframework.zip"
echo "Zipping Mobileproxy.xcframework for iOS..."
(
  cd "${MOBILEPROXY_ARTIFACT_DIR}"
  zip -qr "$(basename "${IOS_ZIP_PATH}")" "$(basename "${IOS_FRAMEWORK_PATH}")"
)

ANDROID_AAR_PATH="${MOBILEPROXY_ARTIFACT_DIR}/mobileproxy.aar"
echo "Building Mobileproxy for Android..."
(
  cd "${X_DIR_IN_CLONE}"
  gomobile bind -ldflags='-s -w' -target=android -androidapi=21 -o "${ANDROID_AAR_PATH}" \
    "github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
)

echo "Mobileproxy build complete. Artifacts are in ${MOBILEPROXY_ARTIFACT_DIR}"

if [[ -n "${GH_TOKEN:-}" ]]; then
  echo "GH_TOKEN is set. Uploading artifacts to release tag ${RELEASE_TAG}..."
  gh release upload "${RELEASE_TAG}" "${IOS_ZIP_PATH}" "${ANDROID_AAR_PATH}"
  echo "Artifacts uploaded successfully to release ${RELEASE_TAG}."
else
  echo "GH_TOKEN is not set. Skipping artifact upload."
  echo "Built artifacts are available at:"
  echo "  iOS: ${IOS_ZIP_PATH}"
  echo "  Android: ${ANDROID_AAR_PATH}"
fi
