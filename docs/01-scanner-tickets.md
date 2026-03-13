# VettCode Scanner — Milestones & Tickets

**Version:** 0.1-draft
**Status:** In Review
**Parent Document:** [01-scanner-design.md](./01-scanner-design.md)

---

## Target: Scanner MVP in ~2 weeks with Claude Code

## Epic 1: CLI Framework (2 days)

| Ticket | Description | Status | Updated_at | Note |
| --- | --- | --- | --- | --- |
| SC-001 | Project scaffolding: Go module, directory structure, CI setup | resolved | 2026-03-12 | |
| SC-002 | cobra CLI with `scan`, `version`, `help` commands | resolved | 2026-03-12 | |
| SC-003 | Config loading (flags, env vars) | resolved | 2026-03-12 | |
| SC-004 | Path validation and multi-path argument handling | resolved | 2026-03-12 | Fixed non-deterministic label iteration, added output path validation |
| SC-005 | Default exclusion patterns (hardcoded, no user-defined exclusions) | resolved | 2026-03-12 | |
| SC-006 | Logging infrastructure (leveled logging, `--verbose`) | resolved | 2026-03-12 | |
| SC-007 | Git version check (validate Git 2.20+ at scan start; warn + auto-fallback to `--no-git` if older or absent) | resolved | 2026-03-12 | |

## Epic 2: Language Detection & Parsing Infrastructure (1.5 days)

| Ticket | Description | Status | Updated_at | Note |
| --- | --- | --- | --- | --- |
| SC-010 | Language detector (file extension + manifest scanning) | resolved | 2026-03-12 | Case-insensitive extension matching, removed redundant compound extension block |
| SC-011 | Tree-sitter Go wrapper (load grammar, parse file, walk AST) | resolved | 2026-03-12 | Interface + types defined; full tree-sitter integration deferred to Epic 3 |
| SC-012 | Go AST wrapper (parse Go files using `go/ast`) | resolved | 2026-03-12 | Complexity: base 1 + decision points; nesting depth via recursive walk |
| SC-013 | Grammar download manager (GCS fetch, SHA-256 verify, version compatibility check, cache) | resolved | 2026-03-12 | Atomic writes, offline mode support, SHA-256 verification |
| SC-014 | File walker with exclusion filtering | resolved | 2026-03-12 | Debug logging for skipped paths, scanner error handling in LOC counter |

## Epic 3: Core Analyzers (6.5 days)

| Ticket | Description | Status | Updated_at | Note |
| --- | --- | --- | --- | --- |
| SC-020 | Cyclomatic complexity analyzer (JS/TS via tree-sitter) | merged | 2026-03-12 | Shared tree-sitter analyzer with per-language configs; added Summarize() for avg_nesting |
| SC-021 | Cyclomatic complexity analyzer (Python via tree-sitter) | merged | 2026-03-12 | Handles elif, and/or operators |
| SC-022 | Cyclomatic complexity analyzer (Go via go/ast) | merged | 2026-03-12 | Enhanced; added Summarize() for avg_nesting |
| SC-022a | Cyclomatic complexity analyzer (PHP via tree-sitter) | merged | 2026-03-12 | Handles foreach, elseif |
| SC-022b | Cyclomatic complexity analyzer (Ruby via tree-sitter) | merged | 2026-03-12 | Handles unless, until, rescue |
| SC-022c | Cyclomatic complexity analyzer (Java via tree-sitter) | merged | 2026-03-12 | Handles enhanced for, lambda |
| SC-023 | Nesting depth analyzer (all languages) | merged | 2026-03-12 | Integrated into complexity analyzers; avg_nesting via Summarize() |
| SC-024 | Code duplication detector (token-based, cross-language) | merged | 2026-03-12 | Token-based Rabin-Karp (50-token window) for Tier 1 languages via tree-sitter/go-scanner token extraction with normalization ($ID/$LIT). Line-hash (6-line window) fallback for Tier 2. Block merging + min 6-line filter. |
| SC-025 | File size distribution calculator | merged | 2026-03-12 | LOC buckets, % over 500 LOC |
| SC-026 | Secrets detector (regex patterns + entropy) | merged | 2026-03-12 | Fixed: entropy per-line independent, ByCategory populated, regex moved to pkg var, rune-correct entropy. ~25 patterns + Shannon entropy, allowlist filtering |
| SC-027 | CVE lookup (OSV API + bundled snapshot) | merged | 2026-03-12 | Bundled OSV snapshot: go:embed compressed index, version-range lookup (semver + pre-release + build metadata), npm/PyPI/Go ecosystems. Online fallback to snapshot on API failure with consolidated warning. 10s per-call timeout + 30s total network budget via context. Build tool (cmd/osv-snapshot) fetches from GCS with atomic writes. DEFERRED: OSV API caching (24h TTL), concurrent queries, batch API. |
| SC-028 | License detector (SPDX matching) | merged | 2026-03-12 | GPL/AGPL/SSPL/CC problematic license detection |
| SC-029 | Outdated dependency detector (registry API queries) | merged | 2026-03-12 | Deferred to online-mode integration |
| SC-029a | Dependency parsing for PHP (composer.json/composer.lock) | merged | 2026-03-12 | Skips php/ext- requirements |
| SC-029b | Dependency parsing for Ruby (Gemfile/Gemfile.lock) | merged | 2026-03-12 | Gemfile.lock spec parsing, Gemfile fallback |
| SC-029c | Dependency parsing for Java (pom.xml/build.gradle + lockfiles) | merged | 2026-03-12 | Regex-based pom.xml and gradle parsing |
| SC-030 | Dependency health analyzer (age, unmaintained %) | merged | 2026-03-12 | Median age, unmaintained %, oldest dep |
| SC-031 | AI detection analyzer (import/dependency pattern matching) | merged | 2026-03-12 | LLM, VectorDB, RAG, MCP, fine-tuning, training, data pipeline |
| SC-032 | Tech stack detector (frameworks, runtimes, databases, services) | merged | 2026-03-12 | 50+ frameworks, runtime version detection |
| SC-033 | Infrastructure detector (IaC, CI/CD, monitoring) | merged | 2026-03-12 | Docker, Terraform, K8s, CI/CD, monitoring deps+configs |
| SC-034 | Git activity analyzer (log parsing, velocity, trend) | merged | 2026-03-12 | Monthly commits, trend, contributors, HEAD SHA |
| SC-035 | Handoff analyzer (tests, docs, env vars) | merged | 2026-03-12 | LOC-weighted test coverage, doc density, env var count |

## Epic 4: Scoring, Aggregation & Red Flags (1 day)

| Ticket | Description | Status | Updated_at | Note |
| --- | --- | --- | --- | --- |
| SC-040 | Scorer implementation (metric-to-score functions) | merged | 2026-03-13 | 6 category scorers: maintainability, security, handoff, dependency health, activity, infrastructure. Review fixes: added extreme-value and out-of-range tests |
| SC-041 | Grade conversion and category score calculation | merged | 2026-03-13 | Score-to-grade (A through F), weighted overall score with N/A renormalization. Review fix: OverallScore clamped to [0,100] |
| SC-042 | Red flag evaluator (threshold checks, flag generation) | merged | 2026-03-13 | 8 red flag types. Review fixes: float threshold (< 0.01) for test coverage, rounded month display |
| SC-043 | Multi-repo result aggregator | merged | 2026-03-13 | Review fixes: DocDensity worst-case aggregation, removed single-repo fast path, MedianAgeMonths rounding, negative days clamping, 12-bit ActiveMonths mask |
| SC-044 | Pricing tier auto-determination | merged | 2026-03-13 | Review fix: formatLOC guards against negative input |

## Epic 5: Output Pipeline (1 day)

| Ticket | Description | Status | Updated_at | Note |
| --- | --- | --- | --- | --- |
| SC-050 | Terminal formatter (pretty-printer matching Section 8 format, includes scan duration display) | merged | 2026-03-13 | All sections from 7.5 spec. Review fixes: hotspot shows real Path (json:"-"), External Services line in INFRASTRUCTURE, safe slice append |
| SC-051 | Color support with `--no-color` flag | merged | 2026-03-13 | ANSI color codes with ColorConfig.Enabled toggle, grade/severity coloring, no external deps |
| SC-052 | JSON serializer (schema-compliant output) | merged | 2026-03-13 | RFC 8785 canonical JSON, SetEscapeHTML(false), UseNumber() round-trip, 3 cross-language test vectors pass, atomic file write |
| SC-053 | Integrity signer (Ed25519 signing of metrics checksum) | merged | 2026-03-13 | Review fix: nonce included in hash (selective field exclusion, not whole integrity block). Ed25519 sign/verify, deterministic dev key |
| SC-055 | Remote co-signing flow (POST /cosign/init, POST /cosign/complete, nonce embedding, offline fallback, error handling for co-sign API unavailability — see Section 5.8) | merged | 2026-03-13 | Review fixes: validate init/complete response fields, honor Retry-After in completeWithRetry. Init/complete with retry (2x), backoff, fatal error classification |
| SC-054 | Progress indicator (spinner/progress bar during scan) | merged | 2026-03-13 | Review fix: sync.Once for safe double-Stop(). Braille spinner, phase + detail + elapsed time |

## Epic 6: Testing & Validation (2 days)

| Ticket | Description | Status | Updated_at | Note |
| --- | --- | --- | --- | --- |
| SC-060 | Create fixture repos (healthy-saas, neglected-project, security-nightmare, java-enterprise, tier2-only) covering all 6 Tier 1 languages | merged | 2026-03-13 | 5 fixtures with structure tests. Review fixes: go.mod→.fixture to prevent build interference, moved CI/CD to root, added package-lock.json/go.sum.fixture/jest.config.js, fixed **glob→WalkDir, runtime.Caller ok check, fixed complexity annotations for &&/||/ternary/catch, replaced AWS example creds, multi-line PEM key |
| SC-061 | Unit tests for all analyzers | merged | 2026-03-13 | High-complexity fixtures (JS,Python,Java,PHP,Ruby), edge cases (empty, nonexistent, no-functions), secrets false positive tests (UUIDs, git hashes), exact count tests, duplication partial/3-file tests, license type coverage (SSPL,LGPL,EUPL,CC-BY-NC/SA, permissive bulk) |
| SC-062 | Unit tests for scorer and red flag evaluator | merged | 2026-03-13 | Exact grade boundary tests, scorer boundary tests (dep health extreme, handoff medium doc, infra IaC/monitoring-only, single-category overall), red flag combos (security, process), exact threshold tests (unmaintained >=50, stale >180) |
| SC-063 | Integration tests (multi-lang, multi-repo, JSON validation) | merged | 2026-03-13 | 7 integration tests: multi-lang (healthy-saas, java-enterprise), tier2-only no-complexity, neglected-project red flags, security-nightmare secrets, multi-repo aggregation, JSON output validation (no path leaks, required fields). Review fixes: percentage assertions, error logging, safe type assertions, JSON round-trip, stronger aggregation. |
| SC-064 | E2E tests (CLI commands, fixture repo scans) | merged | 2026-03-13 | 6 CLI tests: help, scan help (flags), version output, unknown command, scan flags validation, scan no-args. 8 deferred stubs for full scan E2E. Review fixes: execCLI cleanup, version/unknown command assertions, parallel safety comment. |
| SC-065 | Accuracy validation against reference tools (ESLint, radon, gocyclo, phpmetrics, flog, PMD) | merged | 2026-03-13 | 7 accuracy tests across JS/TS, PHP, Ruby, Java, Python, Go fixtures. Validates complexity per-function with +/-1 tolerance. Review fixes: require.Contains, buildFuncMap/assertComplexity helpers, Go accuracy test via goast. |
| SC-066 | Performance benchmarks setup | merged | 2026-03-13 | Benchmarks: complexity (JS 1K/10K, PHP 10K, Java 10K, 100K mixed), secrets (1K/10K/100K with planted secrets), duplication (1K/10K/100K), dep parsing (all 6 ecosystems). All benchmarks use b.ReportAllocs() and sink variables. DEFERRED: BenchmarkFullScan30K/100K, BenchmarkMemory100K (require scan orchestrator). |

## Epic 7: Distribution (0.5 days)

| Ticket | Description | Status | Updated_at | Note |
| --- | --- | --- | --- | --- |
| SC-070 | GoReleaser configuration (cross-compilation, GitHub Releases) | merged | 2026-03-13 | .goreleaser.yml with 4 build targets, ldflags, checksums, changelog, Homebrew tap, Docker publishing with build args. release.yml with lint+test gates. |
| SC-071 | Dockerfile for Docker image | merged | 2026-03-13 | Multi-stage: golang:1.23-alpine → alpine:3.19 with git + ca-certs. DEFERRED: grammar bundling. |
| SC-072 | Install script (`get.vettcode.com`) | merged | 2026-03-13 | POSIX shell: platform validation, SHA-256 verification, sudo check, Windows .exe, rate-limit hint. |
| SC-073 | Homebrew tap formula | merged | 2026-03-13 | Auto-generated by GoReleaser. Reference template in packaging/homebrew/. |
| SC-074 | Version check mechanism (24h-throttled, non-blocking, cached in ~/.vettcode/) | merged | 2026-03-13 | 24h cache, 2s timeout, 0600 perms, semver comparison, VETTCODE_NO_UPDATE_CHECK, 18 tests. |

## Epic 8: Post-MVP Enhancements

| Ticket | Description | Status | Updated_at | Note |
| --- | --- | --- | --- | --- |
| SC-080 | Optional grammar management commands (`vettcode grammar list/install/update`) | | | |
| SC-081 | Windows compatibility testing and fixes | | | |
| SC-082 | `--format terminal` (terminal only, no JSON) | | | |
| SC-083 | Duplication detection sampling for 300K+ LOC repos | | | |
| SC-084 | Additional secret patterns (expand regex library) | | | |
| SC-085 | CI/CD integration mode (`--ci` flag with exit codes based on score thresholds) | | | |

## Summary

| Epic | Effort | Priority |
| --- | --- | --- |
| CLI Framework | 14h (~1.5 days) | MVP |
| Language Detection & Parsing | 16h (~1.5 days) | MVP |
| Core Analyzers | 87h (~6.5 days) | MVP |
| Scoring, Aggregation & Red Flags | 12h (~1 day) | MVP |
| Output Pipeline | 13h (~1 day) | MVP |
| Testing & Validation | 28h (~2 days) | MVP |
| Distribution | 6h (~0.5 days) | MVP |
| **MVP Total** | **~176h (~14 days)** | |
| Post-MVP Enhancements | ~17h | Post-MVP |

**Note:** These estimates assume a single developer working with Claude Code, which provides substantial acceleration on boilerplate, test writing, and pattern-matching code. The parallelizable nature of analyzer implementation (each analyzer is independent) means a developer can rapidly iterate through them with AI assistance.

## Recommended Build Order

1. **Days 1-2:** CLI framework + language detection + file walker + tree-sitter setup (SC-001 through SC-014)
2. **Days 3-5:** Complexity analyzers for all 6 languages + duplication detection (SC-020 through SC-025)
3. **Days 6-8:** Security analyzers + dependency parsing & health for all 6 languages (SC-026 through SC-030)
4. **Days 9-10:** Detection analyzers (AI, tech stack, infra, activity, handoff) (SC-031 through SC-035)
5. **Days 11-12:** Scoring + aggregation + red flags + output pipeline (SC-040 through SC-054)
6. **Days 13-14:** Testing, validation, distribution (SC-060 through SC-072)
