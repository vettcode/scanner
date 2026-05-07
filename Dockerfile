# ─────────────────────────────────────────────────────
# Stage 1: Build the scanner binary
# ─────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

RUN CGO_ENABLED=1 go build \
    -ldflags "-s -w \
      -X github.com/vettcode/scanner/internal/cli.version=${VERSION} \
      -X github.com/vettcode/scanner/internal/cli.commit=${COMMIT} \
      -X github.com/vettcode/scanner/internal/cli.date=${DATE}" \
    -o /vettcode ./cmd/vettcode

# ─────────────────────────────────────────────────────
# Stage 2: Bundle grammars (placeholder — populate from
# GCS or local cache once grammar files are published)
# ─────────────────────────────────────────────────────
FROM alpine:3.19 AS grammars

# When grammar WASM files are available, COPY or download them here.
# For now, create the directory structure so the scanner's grammar
# manager finds it and skips downloads.
#
# Production build command will add:
#   COPY grammars/ /grammars/0.1.0/
#
# Or download at build time:
#   RUN wget -q https://storage.googleapis.com/vettcode-grammars/0.1.0/tree-sitter-javascript.wasm ...
RUN mkdir -p /grammars/0.1.0

# ─────────────────────────────────────────────────────
# Stage 3: Runtime — minimal image
# ─────────────────────────────────────────────────────
FROM alpine:3.19

# git is required for development activity analysis (git log)
# ca-certificates for TLS (co-signing, grammar downloads, version check)
RUN apk add --no-cache git ca-certificates \
    && addgroup -S vettcode \
    && adduser -S -G vettcode -h /home/vettcode -s /sbin/nologin vettcode

# Scanner binary
COPY --from=builder /vettcode /usr/local/bin/vettcode

# Pre-bundled grammars (per AC-2.7, AC-4.7: no downloads needed in Docker mode)
COPY --from=grammars /grammars /home/vettcode/.vettcode/grammars

# Ensure the vettcode user owns its home directory
RUN chown -R vettcode:vettcode /home/vettcode

# Run as non-root
USER vettcode
WORKDIR /scan

# Labels (populated by GoReleaser build args)
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
LABEL org.opencontainers.image.title="vettcode-scanner" \
      org.opencontainers.image.description="Privacy-first code health scanner for due diligence" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${DATE}" \
      org.opencontainers.image.source="https://github.com/vettcode/scanner" \
      org.opencontainers.image.vendor="VettCode" \
      org.opencontainers.image.licenses="Proprietary"

ENTRYPOINT ["vettcode"]
