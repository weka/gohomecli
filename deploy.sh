#!/bin/bash

set -e

. get_version.sh

docker build -t gohomecli-deploy . --build-arg BUILD_VERSION="$BUILD_VERSION" -f go.Dockerfile

container_id=$(docker create gohomecli-deploy:latest)
rm -rf bin
docker cp "$container_id":/src/bin bin
docker rm -v "$container_id"

if [[ "$BUILDKITE_BRANCH" != "staging" &&  "$BUILDKITE_BRANCH" != "ga" ]]; then
  folder=dev/$(uuidgen)
else
  if [[ $(git status --porcelain) != "" ]]; then
    echo "Refusing to build release on dirty repository"
    exit 1
  else
    folder=release/$(git rev-parse HEAD)
  fi
fi

aws s3 cp "bin" "s3://gohomecli/$folder/"  --region "eu-central-1" --recursive
