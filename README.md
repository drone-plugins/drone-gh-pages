# drone-gh-pages

[![Build Status](http://beta.drone.io/api/badges/drone-plugins/drone-gh-pages/status.svg)](http://beta.drone.io/drone-plugins/drone-gh-pages)
[![Join the discussion at https://discourse.drone.io](https://img.shields.io/badge/discourse-forum-orange.svg)](https://discourse.drone.io)
[![Drone questions at https://stackoverflow.com](https://img.shields.io/badge/drone-stackoverflow-orange.svg)](https://stackoverflow.com/questions/tagged/drone.io)
[![Go Doc](https://godoc.org/github.com/drone-plugins/drone-gh-pages?status.svg)](http://godoc.org/github.com/drone-plugins/drone-gh-pages)
[![Go Report](https://goreportcard.com/badge/github.com/drone-plugins/drone-gh-pages)](https://goreportcard.com/report/github.com/drone-plugins/drone-gh-pages)
[![](https://images.microbadger.com/badges/image/plugins/gh-pages.svg)](https://microbadger.com/images/plugins/gh-pages "Get your own image badge on microbadger.com")

Drone plugin for publishing static website to GitHub Pages. For the usage information and a listing of the available options please take a look at [the docs](http://plugins.drone.io/drone-plugins/drone-gh-pages/).

## Build

Build the binary with the following commands:

```
go build
```

## Docker

Build the Docker image with the following commands:

```
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -tags netgo -o release/linux/amd64/drone-gh-pages
docker build --rm -t plugins/gh-pages .
```

### Usage

```
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
