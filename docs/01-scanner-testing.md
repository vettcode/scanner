# VettCode Scanner — Testing Plan

**Version:** 0.1-draft
**Status:** In Review
**Parent Document:** [01-scanner-design.md](./01-scanner-design.md)

---

## Table of Contents

1. [Unit Tests](#1-unit-tests)
2. [Integration Tests](#2-integration-tests)
3. [E2E Tests](#3-e2e-tests)
4. [Accuracy Validation Strategy](#4-accuracy-validation-strategy)
5. [Performance Benchmarks](#5-performance-benchmarks)

---

## 1. Unit Tests

Every analyzer has unit tests with fixture inputs and expected outputs.

**Maintainability:**

- Fixture files with known cyclomatic complexity (hand-computed) for each Tier 1 language:
- JS/TS file: simple function (complexity 1), function with nested ifs (complexity 8), complex controller (complexity 25)
- Python file: similar set of functions
- Go file: similar set of functions
- PHP file: similar set of functions (including `elseif`, `foreach` constructs)
- Ruby file: similar set of functions (including `elsif`, `unless`, `until`, `rescue`, `&.` constructs)
- Java file: similar set of functions (including `instanceof`, `switch` expressions, `try-with-resources`)
- Duplication: two files with known duplicate blocks; assert exact duplication percentage

**Security:**

- Fixture files with planted secrets (AWS keys, API tokens, PEM keys); assert exact count
- Fixture files with NO secrets (high-entropy but legitimate strings like UUIDs, hashes); assert zero false positives
- Fixture lockfiles with known CVE-affected packages; assert CVE detection
- Fixture with copyleft + permissive license mix; assert license issue detection

**Dependency Health:**

- Mock registry responses with known publish dates; assert correct median age and unmaintained percentage
- Dependency parsing unit tests per manifest format:
  - JS/TS: `package.json` + `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`
  - Python: `requirements.txt`, `pyproject.toml` + `poetry.lock`, `Pipfile.lock`
  - Go: `go.mod` + `go.sum`
  - PHP: `composer.json` + `composer.lock`
  - Ruby: `Gemfile` + `Gemfile.lock`, `*.gemspec`
  - Java: `pom.xml`, `build.gradle` + lockfile, `build.gradle.kts` + lockfile

**AI Detection:**

- Fixture package.json with `openai` dependency; assert LLM API = true
- Fixture requirements.txt with `chromadb`; assert vector DB = true
- Fixture with no AI packages; assert all flags false

**Activity:**

- Fixture git repo with known commit history; assert correct velocity, trend, active months, contributor count

**Handoff:**

- Fixture repo with 10 source files + 5 `*_test.go` files → assert estimated coverage = 33%
- Fixture JS repo with files under `__tests__/` + `jest.config.js` → assert `has_test_config = true`
- Fixture repo with README, .env.example → assert correct doc density and env var detection
- Fixture repo with nothing → assert zero estimated coverage, low doc density, no test config flags

**Scorer:**

- Unit tests for each scoring function with boundary values
- Grade conversion tests at every grade boundary

**Red Flags:**

- Test each red flag trigger condition independently
- Test combinations of red flags

## 2. Integration Tests

**Multi-language scan:**

- Fixture project with all 6 Tier 1 languages (JS/TS, Python, Go, PHP, Ruby, Java)
- Assert correct language detection and percentage breakdown
- Assert per-language complexity analyzers all produce results
- Assert per-language dependency parsing all produce results
- Tier 2 language files (HTML, CSS, YAML) included — assert they appear in tech stack and LOC but not in complexity metrics

**Multi-repo scan:**

- Three fixture directories simulating a real multi-repo product (e.g., JS/TS frontend, Python backend, Go microservice)
- Assert correct aggregation, cross-repo duplication detection

**JSON output validation:**

- Scan fixture repos, parse output JSON, validate against JSON schema
- Assert all required fields are present
- Assert no file names or paths leak into JSON output (only hashed identifiers)
- Assert terminal output DOES show real file paths for hotspots and secrets locations

**Offline mode:**

- Scan with `--offline` flag and no network access
- Assert scan completes (with degraded dependency metrics)
- Assert no network calls attempted

## 3. E2E Tests

**CLI command tests:**

- `vettcode scan <fixture>` -- assert exit code 0, JSON file created, terminal output matches format
- `vettcode scan --quiet` -- assert no terminal output, JSON file still created
- `vettcode scan --no-color` -- assert no ANSI escape codes in output
- `vettcode scan <nonexistent>` -- assert exit code 1, helpful error message, no JSON file created
- `vettcode scan <empty-dir>` -- assert exit code 1, "no supported languages" error
- `vettcode scan ./go-app ./swift-app` (Swift repo has no Tier 1 languages) -- assert exit code 0 with warning, JSON includes swift-app with `"status": "unsupported"` and `"detected_languages": ["swift"]`, scored metrics based only on go-app
- `vettcode scan ./swift-app` (single repo, no Tier 1 languages) -- assert exit code 1, fatal error
- `vettcode scan --offline` (grammars cached) -- assert exit code 0, no network calls
- `vettcode scan --offline` (grammars not cached) -- assert exit code 1, error lists missing grammars
- `vettcode version` -- assert version string format

**Error handling and cleanup:**

- Scan that fails mid-scan -- assert no partial JSON file left behind
- Scan with one analyzer timeout -- assert exit code 0, JSON created with `null` section and `warnings` array
- Scan with invalid `--output` path -- assert exit code 1, permission error with fix suggestion

**Fixture repos with known scores:**

- "healthy-saas" fixture (JS/TS + Python): expect maintainability B+ to A-, security A-
- "neglected-project" fixture (PHP): expect red flags for stale repo, no tests, no CI/CD
- "security-nightmare" fixture (Ruby): expect red flags for secrets, critical CVEs
- "java-enterprise" fixture (Java + Go): expect correct multi-language analysis, Maven/Gradle dependency parsing
- "tier2-only" fixture (HTML + CSS + YAML only): expect LOC and tech stack reported, complexity/dependency scores marked N/A

## 4. Accuracy Validation Strategy

We validate VettCode's output against established, trusted tools across every metric category. This serves two purposes: (1) catching bugs in our analyzers, and (2) building confidence that our results are credible when compared to tools buyers may already know.

**Complexity — validate against SonarQube + language-specific tools:**

1. Run SonarQube Community Edition on the same fixture repos; compare complexity scores per file
2. Per-language cross-checks:
   - JS/TS: Compare with ESLint complexity rule output
   - Python: Compare with `radon` complexity output
   - Go: Compare with `gocyclo` output
   - PHP: Compare with `phpmetrics` complexity output
   - Ruby: Compare with `flog` complexity output
   - Java: Compare with `PMD` complexity output
3. Hand-compute complexity for 20 representative functions across 6 languages
4. Tolerance: +/- 1 per function (minor differences expected due to operator counting variations)

**Secrets detection — validate against truffleHog + GitLeaks:**

1. Run both `truffleHog` and `gitleaks` on the same fixture repos
2. VettCode must detect everything truffleHog detects (zero false negatives vs truffleHog)
3. Known-secrets test suite (planted secrets: AWS keys, API tokens, PEM keys, connection strings)
4. Known-clean test suite (high-entropy but legitimate strings like UUIDs, hashes); assert zero false positives

**CVE detection — validate against Snyk + Trivy:**

1. Run `snyk test` and `trivy fs` on the same fixture repos with known vulnerable dependencies
2. Compare CVE IDs detected: VettCode should match or exceed Trivy's detection rate (Trivy uses the same OSV data source)
3. Compare severity ratings: assert severity matches within one level (e.g., Snyk says "high", we say "high" or "critical" is acceptable; "low" is not)
4. Validate across all 6 dependency manifest formats

**Outdated dependencies — validate against ecosystem tools + Snyk:**

1. Compare against `npm outdated`, `pip-audit`, `composer outdated`, `bundle outdated`, `go list -m -u all`, `mvn versions:display-dependency-updates`
2. Run `snyk test` for cross-language validation
3. Assert outdated count matches within +/- 5% (minor differences due to registry API timing)

**License detection — validate against Snyk:**

1. Run `snyk test` with license policy on fixture repos
2. Assert VettCode detects the same copyleft/restrictive licenses
3. Known-license fixture with GPL, AGPL, MIT, Apache-2.0, ISC, LGPL mix

**Duplication detection — validate against SonarQube + jscpd:**

1. Run SonarQube duplication analysis on the same fixture repos
2. Run `jscpd` as a secondary check
3. Hand-crafted duplicate blocks with known percentages
4. Tolerance: +/- 3% duplication percentage (different tools use different token window sizes)

**Dependency health — validate against Snyk + Libraries.io:**

1. Compare median dependency age against Libraries.io API data
2. Compare unmaintained dependency percentage against Snyk's maintenance status flags
3. Validate that "oldest dependency" matches manual verification via package registry publish dates

## 5. Performance Benchmarks

| Benchmark | Description | Target |
| --- | --- | --- |
| `BenchmarkComplexityJS10K` | Complexity analysis of 10K LOC JS/TS | < 5 seconds |
| `BenchmarkComplexityJava10K` | Complexity analysis of 10K LOC Java | < 5 seconds |
| `BenchmarkComplexityPHP10K` | Complexity analysis of 10K LOC PHP | < 5 seconds |
| `BenchmarkComplexity100K` | Complexity analysis of 100K LOC (mixed langs) | < 60 seconds |
| `BenchmarkDuplication100K` | Duplication detection on 100K LOC | < 90 seconds |
| `BenchmarkSecrets100K` | Secrets scan on 100K LOC | < 30 seconds |
| `BenchmarkDepParsing` | Parse all dependency formats (all 6 languages) | < 5 seconds |
| `BenchmarkFullScan30K` | Full scan of 30K LOC (6 languages mixed) | < 2 minutes |
| `BenchmarkFullScan100K` | Full scan of 100K LOC (6 languages mixed) | < 5 minutes |
| `BenchmarkMemory100K` | Peak memory during 100K LOC scan | < 1 GB |

Benchmarks run in CI on every PR. Regressions >20% fail the build.
