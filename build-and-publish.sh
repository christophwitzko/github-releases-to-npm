#!/usr/bin/env bash

set -euo pipefail

cd go-builder
docker build -t go-builder .
cd -

#./go-builder/run.sh "GoogleCloudPlatform" "berglas" "Apache-2.0"
./go-builder/run.sh "tcnksm" "ghr" "MIT"
