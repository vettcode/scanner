# Enterprise Platform

Multi-language enterprise platform with Java API and Go worker.

## Components

- **api/**: Spring Boot REST API (Java 17, Maven)
- **worker/**: Background job processor (Go 1.21)

## Quick Start

```bash
docker-compose up -d
```

## Development

```bash
# API
cd api && mvn spring-boot:run

# Worker
cd worker && go run ./cmd/main.go
```

## Testing

```bash
cd api && mvn test
cd worker && go test ./...
```
