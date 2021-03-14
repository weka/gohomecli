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

if [[ "$BUILDKITE_BRANCH" == "ga" ]]; then
  if [ -z "$DEPLOY_APP_ID" ]; then
    echo "You must supply DEPLOY_APP_ID environment variable !"
    exit 1
  fi

  if [ -z "$DEPLOY_APP_PRIVATE_KEY" ]; then
    echo "You must supply DEPLOY_APP_PRIVATE_KEY environment variable !"
    exit 1
  fi
  docker build -t github-token . -f python.Dockerfile
  eval "$(docker run -e "DEPLOY_APP_ID=$DEPLOY_APP_ID" -e "DEPLOY_APP_PRIVATE_KEY=$DEPLOY_APP_PRIVATE_KEY" github-token)"

  AUTH="Authorization: token $GITHUB_TOKEN"
  result=$(curl \
    -X POST \
    -H "$AUTH" \
    https://api.github.com/repos/weka/gohomecli/releases \
    -d "{\"tag_name\":\"$BUILD_VERSION\", \"body\":\"GA release\"}")

  id=$(echo "$result" | jq -c ".id")

  filenames="bin/homecli_linux_amd64 bin/homecli_darwin_amd64"
  for filename in $filenames; do
    curl \
      -H "$AUTH" \
      -H "Content-Type: $(file -b --mime-type "$filename")" \
      --data-binary @"$filename" \
      "https://uploads.github.com/repos/weka/gohomecli/releases/$id/assets?name=$(basename "$filename")"
  done
fi
