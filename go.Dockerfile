FROM golang:1.21.4-alpine3.17 as go-builder

WORKDIR /src

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

ADD . /src
ARG BUILD_VERSION
RUN /src/build.sh "$BUILD_VERSION"
