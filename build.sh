#!/usr/bin/env sh
set -eu

GOLDFLAGS="-s -w -extldflags '-zrelro -znow -O1'"
export GOFLAGS="-buildmode=pie -trimpath -mod=readonly -modcacherw"
go build -ldflags "$GOLDFLAGS" -o ./bin/ ./cmd/*
