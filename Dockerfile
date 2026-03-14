# Stage 1: Build
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

# Pre-cache grammars during build so the image works fully offline (FR-2).
# VETTCODE_HOME controls where grammars are cached.
ENV VETTCODE_HOME=/vettcode-data
RUN /vettcode grammar install || echo "WARN: grammar download failed; image will download on first scan"

# Stage 2: Runtime
FROM alpine:3.19

RUN apk add --no-cache git ca-certificates

COPY --from=builder /vettcode /usr/local/bin/vettcode
COPY --from=builder /vettcode-data /usr/local/share/vettcode

ENV VETTCODE_HOME=/usr/local/share/vettcode

ENTRYPOINT ["vettcode"]
