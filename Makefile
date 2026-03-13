.PHONY: build test lint vet clean

BINARY_NAME=vettcode
BUILD_DIR=bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS  = -ldflags "-X github.com/vettcode/scanner/internal/cli.version=$(VERSION) \
                      -X github.com/vettcode/scanner/internal/cli.commit=$(COMMIT) \
                      -X github.com/vettcode/scanner/internal/cli.date=$(DATE)"

build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/vettcode

test:
	go test -race -count=1 ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)

all: vet lint test build
