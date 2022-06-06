#!/usr/bin/env bash

# Copyright 2021 The Clusternet Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

APP_ROOT=$(dirname "${BASH_SOURCE[0]}")/../..
PLATFORMS=${PLATFORMS:-linux/amd64}
GO_ENABLED=${CGO_ENABLED:-0}

source "${APP_ROOT}/hack/lib/version.sh"

function abspath() {
    (
        if [[ -d "${1}" ]]; then
            cd "${1}"
            pwd -P
        else
            cd "$(dirname "${1}")"
            local f
            f=$(basename "${1}")
            if [[ -L "${f}" ]]; then
                readlink "${f}"
            else
                echo "$(pwd -P)/${f}"
            fi
        fi
    )
}

sample-controller::setup_platform() {
    local platform=$1
    local goos
    local goarch

    case "${platform}" in
    "darwin/amd64")
      goos=darwin
      goarch=amd64
    ;;
    "darwin/arm64")
      goos=darwin
      goarch=arm64
      ;;
    "linux/amd64")
      goos=linux
      goarch=amd64
      ;;
    "linux/arm")
      goos=linux
      goarch=arm
      ;;
    "linux/arm64")
      goos=linux
      goarch=arm64
      ;;
    "linux/ppc64le")
      goos=linux
      goarch=ppc64le
      ;;
    "linux/s390x")
      goos=linux
      goarch=s390x
      ;;
    "linux/386")
      goos=linux
      goarch=386
      ;;
    *)
      echo "Unsupported platform. Must be in darwin/amd64, darwin/arm64, linux/amd64, linux/arm, linux/arm64, linux/ppc64le, linux/s390x, linux/386"
      exit 1
      ;;
  esac
  
  export GOOS=${goos}
  export GOARCH=${goarch}
}

sample-controller::build_binary() {
   sample-controller::verify_golang
   (
       echo "building with $(go version)"
       
       local goldflag
       goldflags="$(sample-controller::ldflags)"

       local platform=$1
       sample-controller::setup_platform "${platform}"

       local target=$2
       echo "Building cmd/${target} binary for ${platform} ..."

       GOOS=${GOOS} GOARCH=${GOARCH} \
        CGO_ENABLED=${CGO_ENABLED-} \
        GOPATH="$(abspath ${APP_ROOT}/../../../../)" \
        go build -ldflags "$goldflags" -o ./_output/${platform}/bin/${target} ./cmd/${target}/
   ) 
}

sample-controller::verify_golang() {
    if [[ -z "$(command -v go)" ]]; then
    echo """
Can't find 'go' in PATH, please fix and retry.
See http://golang.org/doc/install for installation instructions.
"""
    return 2
  fi
}

sample-controller::host_platform() {
  if [[ "$(go env GOHOSTOS)" == "darwin" ]]; then
    echo "linux/$(go env GOHOSTARCH)"
  else
    echo "$(go env GOHOSTOS)/$(go env GOHOSTARCH)"
  fi
}

sample-controller::docker-image() {
    (
        local platform=$1
        local target=$2
        local LDFLAGS="$(sample-controller::ldflags)"

        sample-controller::setup_platform "${platform}"

        tag=$(git describe --tags --always)
        echo "Building docker image ${REGISTRY}/sample-controller/${target}-${GOARCH}:${tag} ..."

        docker buildx build \
        --load \
        --platform="$1" \
        -t "${REGISTRY}/sample-controller/$2-${GOARCH}:${tag}" \
        --build-arg BASEIMAGE="${BASEIMAGE}" \
        --build-arg GOVERSION="${GOVERSION}" \
        --build-arg LDFLAGS="${LDFLAGS}" \
        --build-arg PKGNAME="${target}" .
    )
}
