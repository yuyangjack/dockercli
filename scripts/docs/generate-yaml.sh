#!/usr/bin/env bash
# Generate yaml for docker/cli reference docs
set -eu -o pipefail

mkdir -p docs/yaml/gen

go build -o build/yaml-docs-generator github.com/yuyangjack/dockercli/docs/yaml
build/yaml-docs-generator --root "$(pwd)" --target "$(pwd)/docs/yaml/gen"
