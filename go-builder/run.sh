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
  echo "license not set"
  exit 1
}
license=$3

echo "building ${owner}/${repo} (${license})"

latest_release=$(curl "https://api.github.com/repos/${owner}/${repo}/releases/latest" | jq -r '.tag_name')
version="${latest_release:1}"
echo "latest release: $version"

rm -rf ./bin && mkdir "bin"
docker run -it --rm -v "$(pwd)/bin:/build/bin" go-builder "$owner" "$repo" "$version"

./npm-binary-releaser -n "${repo}" \
  -r "$version" \
  --package-name-prefix "@install-binary/" \
  --license "${license}" \
  --homepage "https://github.com/${owner}/${repo}" \
  --repository "github:${owner}/${repo}" \
  --publish
