#!/usr/bin/env sh

set -e  # Exit on any error

BUILD_VERSION=$1
BUILD_TIME=$(date +'%Y-%m-%d_%T')

echo "Building homecli version: ${BUILD_VERSION:-<no version specified>}"

LD_FLAGS="-X main.BuildVersion=$BUILD_VERSION -X main.BuildTime=$BUILD_TIME"
LD_FLAGS="$LD_FLAGS -s -w"

echo "Building Linux amd64 binary..."
if ! CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$LD_FLAGS" -o bin/homecli_linux_amd64 cmd/homecli/*.go; then
  echo "::error::Failed to build Linux binary"
  exit 1
fi

echo "Building macOS amd64 binary..."
if ! CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$LD_FLAGS" -o bin/homecli_darwin_amd64 cmd/homecli/*.go; then
  echo "::error::Failed to build macOS binary"
  exit 1
fi

# Add UPX compression (only on Linux, only for Linux binary)
# UPX doesn't support macOS binaries and the Linux UPX binary can't run on macOS

if [ "$(uname)" = "Linux" ]; then
  echo "Compressing Linux binary with UPX..."
  UPX_VERSION="${UPX_VERSION:-4.2.2}"
  UPX_PLATFORM="${UPX_PLATFORM:-amd64_linux}"
  # SHA256 checksum verified from official UPX 4.2.2 release
  UPX_SHA256="915c8e844f835de03b9cc311ff185aedec79d757aee9d7133a528b9e89c463bb"

  UPX_ARCHIVE="upx-${UPX_VERSION}-${UPX_PLATFORM}.tar.xz"

  if [ ! -d "upx-${UPX_VERSION}-${UPX_PLATFORM}" ]; then
    echo "Downloading UPX ${UPX_VERSION}..."
    if ! wget -q "https://github.com/upx/upx/releases/download/v${UPX_VERSION}/${UPX_ARCHIVE}"; then
      echo "::error::Failed to download UPX"
      exit 1
    fi

    echo "Verifying UPX checksum..."
    ACTUAL_SHA256=$(sha256sum "$UPX_ARCHIVE" | cut -d' ' -f1)
    if [ "$ACTUAL_SHA256" != "$UPX_SHA256" ]; then
      echo "::error::UPX checksum verification failed!"
      echo "Expected: $UPX_SHA256"
      echo "Actual:   $ACTUAL_SHA256"
      rm -f "$UPX_ARCHIVE"
      exit 1
    fi
    echo "Checksum verified."

    if ! tar xf "$UPX_ARCHIVE" "upx-${UPX_VERSION}-${UPX_PLATFORM}/upx"; then
      echo "::error::Failed to extract UPX"
      exit 1
    fi
  fi

  if ! "upx-${UPX_VERSION}-${UPX_PLATFORM}/upx" bin/homecli_linux_amd64; then
    echo "::error::Failed to compress Linux binary with UPX"
    exit 1
  fi
else
  echo "Skipping UPX compression (not running on Linux)"
fi

echo ""
echo "Build completed successfully!"
