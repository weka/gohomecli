#!/usr/bin/env sh

BUILD_VERSION=$1
BUILD_TIME=$(date +'%Y-%m-%d_%T')

echo "Building homecli version: ${BUILD_VERSION:-<no version specified>}"

LD_FLAGS="-X main.BuildVersion=$BUILD_VERSION -X main.BuildTime=$BUILD_TIME"
LD_FLAGS="$LD_FLAGS -s -w"

echo "Building Linux amd64 binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$LD_FLAGS" -o bin/homecli_linux_amd64 cmd/homecli/*.go

echo "Building macOS amd64 binary..."
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$LD_FLAGS" -o bin/homecli_darwin_amd64 cmd/homecli/*.go

# Add UPX compression (only on Linux, only for Linux binary)
# UPX doesn't support macOS binaries and the Linux UPX binary can't run on macOS

if [ "$(uname)" = "Linux" ]; then
  echo "Compressing Linux binary with UPX..."
  UPX_VERSION="${UPX_VERSION:-4.2.2}"
  UPX_PLATFORM="${UPX_PLATFORM:-amd64_linux}"

  if [ ! -d "upx-${UPX_VERSION}-${UPX_PLATFORM}" ]; then
    echo "Downloading UPX ${UPX_VERSION}..."
    wget -q "https://github.com/upx/upx/releases/download/v${UPX_VERSION}/upx-${UPX_VERSION}-${UPX_PLATFORM}.tar.xz"
    tar xf "upx-${UPX_VERSION}-${UPX_PLATFORM}.tar.xz" "upx-${UPX_VERSION}-${UPX_PLATFORM}/upx"
  fi

  "upx-${UPX_VERSION}-${UPX_PLATFORM}/upx" bin/homecli_linux_amd64
else
  echo "Skipping UPX compression (not running on Linux)"
fi

echo ""
echo "Build completed successfully!"
