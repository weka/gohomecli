FROM golang:1.21.4-alpine3.17 as go-builder
# https://stackoverflow.com/questions/36279253/go-compiled-binary-wont-run-in-an-alpine-docker-container-on-ubuntu-host
# RUN apk add --no-cache libc6-compat bash util-linux zip

WORKDIR /src

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

ADD . /src
ARG BUILD_VERSION
RUN ls && /src/build.sh "$BUILD_VERSION"
