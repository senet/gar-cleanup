BINARY_NAME := gar-cleanup
MODULE := github.com/senet/gar-cleanup
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build test lint clean release

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/gar-cleanup/

test:
	go test -v -race -count=1 ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

cover:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out
	@rm -f coverage.out

release:
	goreleaser release --clean
