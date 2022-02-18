#!/usr/bin/env sh
set -eu

GOLDFLAGS="-s -w -extldflags '-zrelro -znow -Wl -O1 --sort-common --as-needed'"
export CGO_ENABLED=0 GOFLAGS="-buildmode=pie -trimpath -mod=readonly -modcacherw"
go build -ldflags "$GOLDFLAGS" -o ./bin/ ./cmd/*
