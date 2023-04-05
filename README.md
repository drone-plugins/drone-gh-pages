# drone-gh-pages

[![Build Status](http://harness.drone.io/api/badges/drone-plugins/drone-gh-pages/status.svg)](http://harness.drone.io/drone-plugins/drone-gh-pages)
[![Slack](https://img.shields.io/badge/slack-drone-orange.svg?logo=slack)](https://join.slack.com/t/harnesscommunity/shared_invite/zt-y4hdqh7p-RVuEQyIl5Hcx4Ck8VCvzBw)
[![Join the discussion at https://community.harness.io](https://img.shields.io/badge/discourse-forum-orange.svg)](https://community.harness.io)
[![Drone questions at https://stackoverflow.com](https://img.shields.io/badge/drone-stackoverflow-orange.svg)](https://stackoverflow.com/questions/tagged/drone.io)
[![Go Doc](https://godoc.org/github.com/drone-plugins/drone-gh-pages?status.svg)](http://godoc.org/github.com/drone-plugins/drone-gh-pages)
[![Go Report](https://goreportcard.com/badge/github.com/drone-plugins/drone-gh-pages)](https://goreportcard.com/report/github.com/drone-plugins/drone-gh-pages)

Drone plugin for publishing static website to GitHub Pages. For the usage information and a listing of the available options please take a look at [the docs](http://plugins.drone.io/drone-plugins/drone-gh-pages/).

## Build

Build the binary with the following command:

```console
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

go build -v -a -tags netgo -o release/linux/amd64/drone-gh-pages
```

## Docker

Build the Docker image with the following command:

```console
docker build \
  --label org.label-schema.build-date=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --label org.label-schema.vcs-ref=$(git rev-parse --short HEAD) \
  --file docker/Dockerfile.linux.amd64 --tag plugins/gh-pages .
```

## Usage

```console
docker run --rm \
  -e PLUGIN_USERNAME="octocat" \
  -e PLUGIN_PASSWORD="p455w0rd" \
  -e PLUGIN_PAGES_DIRECTORY="docs" \
  -e DRONE_COMMIT_AUTHOR="Drone" \
  -e DRONE_COMMIT_AUTHOR_EMAIL="drone@example.com" \
  -e DRONE_REMOTE_URL="https://github.com/drone-plugins/drone-docker.git" \
  -e DRONE_WORKSPACE="$(pwd)" \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  plugins/gh-pages
```
