#!/usr/bin/env bash

set -euo pipefail

[[ -z "${1:-}" ]] && {
  echo "owner not set"
  exit 1
}
owner=$1

[[ -z "${2:-}" ]] && {
  echo "repo not set"
  exit 1
}
repo=$2

[[ -z "${3:-}" ]] && {
  echo "version not set"
  exit 1
}
version=$3

echo "fetching version ${version}"
wget "https://github.com/${owner}/${repo}/archive/v${version}.tar.gz"
tar -xzf "v${version}.tar.gz"

echo "running build..."
cd "${repo}-${version}"
export GOFLAGS="-trimpath"
export CGO_ENABLED=0
gox -parallel 4 -osarch="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 linux/arm" -ldflags="-extldflags '-static' -s -w" -output="bin/{{.Dir}}_v${version}_{{.OS}}_{{.Arch}}" ./
mv bin/* /build/bin/
cd -
