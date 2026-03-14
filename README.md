# VettCode Scanner

Privacy-first static analysis for software M&A technical due diligence. Scan your codebase locally — no source code ever leaves your machine.

## What It Does

VettCode Scanner analyzes a codebase and produces a structured health assessment covering:

- **Maintainability** — cyclomatic complexity, nesting depth, code duplication, hotspot files
- **Security** — hardcoded secrets (~40 patterns), known CVEs (bundled OSV database), license risks
- **Dependency Health** — median age, unmaintained percentage, outdated counts
- **Development Activity** — commit velocity, trends, contributor count, staleness
- **Infrastructure** — IaC detection (Docker, Terraform, K8s), CI/CD, monitoring
- **AI Detection** — LLM APIs, vector databases, RAG pipelines, MCP servers
- **Tech Stack** — frameworks, runtime versions, databases, external services
- **Handoff Readiness** — test coverage heuristic, documentation density, environment variables

Output is a color-coded terminal summary with letter grades (A through F) plus a signed JSON file for upload to the VettCode platform.

## Supported Languages

**Tier 1 — Full analysis** (AST complexity + dependency parsing + all metrics):
JavaScript/TypeScript, Python, Go, PHP, Ruby, Java

**Tier 2 — Detection + LOC only:**
HTML, CSS, SQL, Shell, Markdown, YAML, XML, Dockerfile, Terraform

## Installation

### Homebrew

```bash
brew install vettcode/tap/vettcode
```

### Shell Script

```bash
curl -sSfL https://get.vettcode.com | sh
```

### Docker

```bash
docker run --rm -v $(pwd):/src vettcode/scanner scan /src
```

### From Source

```bash
go install github.com/vettcode/scanner/cmd/vettcode@latest
```

Requires Go 1.22+.

## Quick Start

```bash
# Scan current directory
vettcode scan .

# Scan multiple repos as one product
vettcode scan ./backend ./frontend ./infra

# Label repos in the report
vettcode scan --label api:./backend --label web:./frontend

# Fully offline (no network calls)
vettcode scan . --offline

# JSON only, no terminal output
vettcode scan . --format json -q

# Custom output path
vettcode scan . -o ~/Desktop/my-scan.json
```

## CI/CD Integration

Use `--ci` to fail pipelines on quality gates:

```bash
# Fail if overall grade < C or critical red flags found
vettcode scan . --ci

# Stricter: require grade B+
vettcode scan . --ci --ci-threshold B+

# Fail on any red flag (not just critical)
vettcode scan . --ci --ci-fail-on medium
```

Exit codes: `0` = pass, `1` = quality gate failed or scan error.

### GitHub Actions Example

```yaml
- name: VettCode Scan
  run: |
    curl -sSfL https://get.vettcode.com | sh
    vettcode scan . --ci --ci-threshold C
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--output, -o` | `./vettcode-scan-result.json` | Output JSON file path |
| `--format` | `both` | Output format: `terminal`, `json`, `both` |
| `--label` | | Label repos as `name:path` (repeatable) |
| `--offline` | `false` | Skip co-signing, no network calls |
| `--quiet, -q` | `false` | Suppress terminal output |
| `--no-color` | `false` | Disable ANSI color codes |
| `--no-git` | `false` | Skip git-based analysis |
| `--verbose, -v` | `false` | Debug logging |
| `--timeout` | `30m` | Maximum scan duration |
| `--ci` | `false` | Enable CI quality gate |
| `--ci-threshold` | `C` | Minimum grade to pass CI |
| `--ci-fail-on` | `critical` | Red flag severity that fails CI |

## Red Flags

The scanner evaluates 8 deal-killer conditions:

| Flag | Severity | Trigger |
|------|----------|---------|
| Secrets detected | Critical | Any hardcoded secret found |
| Critical/High CVEs | Critical | Known vulnerabilities in dependencies |
| No tests | High | Zero test coverage detected |
| Stale repository | High | No commits in 180+ days |
| Unmaintained deps | High | 50%+ dependencies older than 2 years |
| No git history | High | Repository has no git history |
| No CI/CD | Medium | No CI/CD pipeline detected |
| No README | Medium | No README file found |

## Privacy

The JSON output contains **no source code, file paths, or content**. File identifiers are SHA-256 hashed. Only aggregate metrics, package names, and computed scores are included. The terminal output shows real paths for your reference but is never uploaded.

## Performance

| Codebase Size | Scan Time | Memory |
|---------------|-----------|--------|
| 30K LOC | < 1 second | ~130 MB |
| 100K LOC | < 3 seconds | ~450 MB |
| 300K+ LOC | < 15 minutes | < 2 GB |

Duplication detection uses sampling for repos over 300K LOC to bound memory and runtime.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `VETTCODE_HOME` | Config/cache directory (default: `~/.vettcode`) |
| `VETTCODE_OFFLINE` | Force offline mode |
| `VETTCODE_NO_COLOR` | Disable color output |
| `VETTCODE_NO_UPDATE_CHECK` | Disable version update checks |
| `VETTCODE_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` |

## Development

```bash
# Run tests
go test ./...

# Run benchmarks
go test ./internal/cli/ -bench BenchmarkFullScan -benchtime 1x

# Build
go build -o vettcode ./cmd/vettcode

# Lint
golangci-lint run
```

## License

Proprietary. See LICENSE file.
