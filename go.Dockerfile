FROM golang:1.15.6-alpine3.12 as go-builder
# https://stackoverflow.com/questions/36279253/go-compiled-binary-wont-run-in-an-alpine-docker-container-on-ubuntu-host
RUN apk add --no-cache libc6-compat bash util-linux zip
COPY go.mod /src/go.mod
COPY go.sum /src/go.sum
WORKDIR /src
RUN go mod download
ADD . /src
ARG BUILD_VERSION
RUN ./build.sh "$BUILD_VERSION"
