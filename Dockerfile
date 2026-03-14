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

# Stage 2: Runtime
FROM alpine:3.19

RUN apk add --no-cache git ca-certificates

COPY --from=builder /vettcode /usr/local/bin/vettcode

ENTRYPOINT ["vettcode"]
