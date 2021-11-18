#!/usr/bin/env bash

set -euo pipefail

latest_release=$(curl https://api.github.com/repos/GoogleCloudPlatform/berglas/releases/latest | jq -r '.tag_name')
version="${latest_release:1}"
echo "latest release: $version"

docker build -t berglas-builder .
docker run -it --rm -v "$(pwd)/bin:/build/bin" berglas-builder "$version"
curl -SL "https://get-release.xyz/christophwitzko/npm-binary-releaser/darwin/$(uname -m)" > ./npm-binary-releaser
chmod +x ./npm-binary-releaser
./npm-binary-releaser -n berglas \
  -r "$version" \
  --package-name-prefix "@install-binary/" \
  --license "Apache-2.0" \
  --homepage "https://github.com/GoogleCloudPlatform/berglas" \
  --repository "github:GoogleCloudPlatform/berglas" \
  --publish
