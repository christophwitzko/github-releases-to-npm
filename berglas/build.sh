#!/usr/bin/env bash

set -euo pipefail

[[ -z "${1:-}" ]] && {
  echo "version not set"
  exit 1
}

echo "fetching version $1"
wget "https://github.com/GoogleCloudPlatform/berglas/archive/v${1}.tar.gz"
tar -xzf "v${1}.tar.gz"

echo "running build..."
cd "berglas-${1}"
export GOFLAGS="-trimpath"
export CGO_ENABLED=0
gox -parallel 4 -osarch="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 linux/arm" -ldflags="-extldflags '-static' -s -w" -output="bin/{{.Dir}}_v${1}_{{.OS}}_{{.Arch}}" ./
mv bin/* /build/bin/
cd -
