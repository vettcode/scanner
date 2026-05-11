.PHONY: build test lint vet clean release

BINARY_NAME=vettcode
BUILD_DIR=bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# SCANNER_SIGNING_SEED: base64-encoded 32-byte Ed25519 seed for the production signing key.
# Set this in CI/CD to embed the production key in release binaries.
# Leave unset for local dev builds (dev fallback key is used instead).
SCANNER_SIGNING_SEED ?=

LDFLAGS_BASE = -X github.com/vettcode/scanner/internal/cli.version=$(VERSION) \
               -X github.com/vettcode/scanner/internal/cli.commit=$(COMMIT) \
               -X github.com/vettcode/scanner/internal/cli.date=$(DATE)

# Append signing seed only when provided
ifneq ($(SCANNER_SIGNING_SEED),)
LDFLAGS_BASE += -X github.com/vettcode/scanner/internal/output.embeddedSigningKeySeed=$(SCANNER_SIGNING_SEED)
endif

LDFLAGS = -ldflags "$(LDFLAGS_BASE)"

build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/vettcode

# release: production build with signing key embedded. Requires SCANNER_SIGNING_SEED.
release:
	@if [ -z "$(SCANNER_SIGNING_SEED)" ]; then \
		echo "ERROR: SCANNER_SIGNING_SEED is required for release builds"; \
		exit 1; \
	fi
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
