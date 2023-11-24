#!/usr/bin/env sh

BUILD_VERSION=$1
BUILD_TIME=$(date +'%Y-%m-%d_%T')
LD_FLAGS="-X main.BuildVersion=$BUILD_VERSION -X main.BuildTime=$BUILD_TIME"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$LD_FLAGS" -o bin/homecli_linux_amd64 cmd/homecli/*.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$LD_FLAGS" -o bin/homecli_darwin_amd64 cmd/homecli/*.go
