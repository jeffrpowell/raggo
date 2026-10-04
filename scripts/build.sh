#!/usr/bin/env bash
set -euo pipefail
out="${1:-bin}"; mkdir -p "$out"
for d in cmd/*/; do
  name=$(basename "$d")
  cgo=0; [ "$name" = raggo-extract-text ] && cgo=1   # only binary importing pkg/vision
  CGO_ENABLED=$cgo GOOS=linux go build -o "$out/$name" "./cmd/$name"
done