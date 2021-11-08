export GOLDFLAGS=-s -w -extldflags '-zrelro -znow'
export GOFLAGS=-trimpath
export CGO_ENABLED=0

.PHONY: all
all: dist

.PHONY: dist
dist: dist/amd64/wallhack dist/arm64/wallhack

.PHONY: dist/amd64/wallhack
dist/amd64/wallhack:
	GOARCH=amd64 go build -ldflags "$(GOLDFLAGS)" -o $@ ./cmd/wallhack

.PHONY: dist/arm64/wallhack
dist/arm64/wallhack:
	GOARCH=arm64 go build -ldflags "$(GOLDFLAGS)" -o $@ ./cmd/wallhack

.PHONY: benchmark
benchmark:
	go test -bench=. ./...

.PHONY: test
test:
	CGO_ENABLED=1 go test -race ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: fmt
fmt:
	gofumpt -s -w .

.PHONY: update
update:
	go get -t -u ./...

