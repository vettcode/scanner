# VettCode Scanner — Milestones & Tickets

**Version:** 0.1-draft
**Status:** In Review
**Parent Document:** [01-scanner-design.md](./01-scanner-design.md)

---

## Target: Scanner MVP in ~2 weeks with Claude Code

## Epic 1: CLI Framework (2 days)

| Ticket | Description | Effort | Priority |
| --- | --- | --- | --- |
| SC-001 | Project scaffolding: Go module, directory structure, CI setup | 2h | MVP |
| SC-002 | cobra CLI with `scan`, `version`, `help` commands | 4h | MVP |
| SC-003 | Config loading (flags, env vars) | 2h | MVP |
| SC-004 | Path validation and multi-path argument handling | 2h | MVP |
| SC-005 | Default exclusion patterns (hardcoded, no user-defined exclusions) | 1h | MVP |
| SC-006 | Logging infrastructure (leveled logging, `--verbose`) | 1h | MVP |
| SC-007 | Git version check (validate Git 2.20+ at scan start; warn + auto-fallback to `--no-git` if older or absent) | 1h | MVP |

## Epic 2: Language Detection & Parsing Infrastructure (1.5 days)

| Ticket | Description | Effort | Priority |
| --- | --- | --- | --- |
| SC-010 | Language detector (file extension + manifest scanning) | 3h | MVP |
| SC-011 | Tree-sitter Go wrapper (load grammar, parse file, walk AST) | 4h | MVP |
| SC-012 | Go AST wrapper (parse Go files using `go/ast`) | 3h | MVP |
| SC-013 | Grammar download manager (GCS fetch, SHA-256 verify, version compatibility check, cache) | 4h | MVP |
| SC-014 | File walker with exclusion filtering | 2h | MVP |

## Epic 3: Core Analyzers (6.5 days)

| Ticket | Description | Effort | Priority |
| --- | --- | --- | --- |
| SC-020 | Cyclomatic complexity analyzer (JS/TS via tree-sitter) | 6h | MVP |
| SC-021 | Cyclomatic complexity analyzer (Python via tree-sitter) | 4h | MVP |
| SC-022 | Cyclomatic complexity analyzer (Go via go/ast) | 4h | MVP |
| SC-022a | Cyclomatic complexity analyzer (PHP via tree-sitter) | 4h | MVP |
| SC-022b | Cyclomatic complexity analyzer (Ruby via tree-sitter) | 4h | MVP |
| SC-022c | Cyclomatic complexity analyzer (Java via tree-sitter) | 5h | MVP |
| SC-023 | Nesting depth analyzer (all languages) | 3h | MVP |
| SC-024 | Code duplication detector (token-based, cross-language) | 8h | MVP |
| SC-025 | File size distribution calculator | 1h | MVP |
| SC-026 | Secrets detector (regex patterns + entropy) | 6h | MVP |
| SC-027 | CVE lookup (OSV API + bundled snapshot) | 6h | MVP |
| SC-028 | License detector (SPDX matching) | 3h | MVP |
| SC-029 | Outdated dependency detector (registry API queries) | 4h | MVP |
| SC-029a | Dependency parsing for PHP (composer.json/composer.lock) | 3h | MVP |
| SC-029b | Dependency parsing for Ruby (Gemfile/Gemfile.lock) | 3h | MVP |
| SC-029c | Dependency parsing for Java (pom.xml/build.gradle + lockfiles) | 5h | MVP |
| SC-030 | Dependency health analyzer (age, unmaintained %) | 3h | MVP |
| SC-031 | AI detection analyzer (import/dependency pattern matching) | 4h | MVP |
| SC-032 | Tech stack detector (frameworks, runtimes, databases, services) | 4h | MVP |
| SC-033 | Infrastructure detector (IaC, CI/CD, monitoring) | 3h | MVP |
| SC-034 | Git activity analyzer (log parsing, velocity, trend) | 4h | MVP |
| SC-035 | Handoff analyzer (tests, docs, env vars) | 4h | MVP |

## Epic 4: Scoring, Aggregation & Red Flags (1 day)

| Ticket | Description | Effort | Priority |
| --- | --- | --- | --- |
| SC-040 | Scorer implementation (metric-to-score functions) | 3h | MVP |
| SC-041 | Grade conversion and category score calculation | 2h | MVP |
| SC-042 | Red flag evaluator (threshold checks, flag generation) | 2h | MVP |
| SC-043 | Multi-repo result aggregator | 4h | MVP |
| SC-044 | Pricing tier auto-determination | 1h | MVP |

## Epic 5: Output Pipeline (1 day)

| Ticket | Description | Effort | Priority |
| --- | --- | --- | --- |
| SC-050 | Terminal formatter (pretty-printer matching Section 8 format, includes scan duration display) | 4h | MVP |
| SC-051 | Color support with `--no-color` flag | 1h | MVP |
| SC-052 | JSON serializer (schema-compliant output) | 3h | MVP |
| SC-053 | Integrity signer (Ed25519 signing of metrics checksum) | 3h | MVP |
| SC-055 | Remote co-signing flow (POST /cosign/init, POST /cosign/complete, nonce embedding, offline fallback, error handling for co-sign API unavailability — see Section 5.8) | 0.5 day | MVP |
| SC-054 | Progress indicator (spinner/progress bar during scan) | 2h | MVP |

## Epic 6: Testing & Validation (2 days)

| Ticket | Description | Effort | Priority |
| --- | --- | --- | --- |
| SC-060 | Create fixture repos (healthy-saas, neglected-project, security-nightmare, java-enterprise, tier2-only) covering all 6 Tier 1 languages | 6h | MVP |
| SC-061 | Unit tests for all analyzers | 6h | MVP |
| SC-062 | Unit tests for scorer and red flag evaluator | 2h | MVP |
| SC-063 | Integration tests (multi-lang, multi-repo, JSON validation) | 4h | MVP |
| SC-064 | E2E tests (CLI commands, fixture repo scans) | 3h | MVP |
| SC-065 | Accuracy validation against reference tools (ESLint, radon, gocyclo, phpmetrics, flog, PMD) | 5h | MVP |
| SC-066 | Performance benchmarks setup | 2h | MVP |

## Epic 7: Distribution (0.5 days)

| Ticket | Description | Effort | Priority |
| --- | --- | --- | --- |
| SC-070 | GoReleaser configuration (cross-compilation, GitHub Releases) | 2h | MVP |
| SC-071 | Dockerfile for Docker image | 2h | MVP |
| SC-072 | Install script (`get.vettcode.com`) | 1h | MVP |
| SC-073 | Homebrew tap formula | 1h | Post-MVP |
| SC-074 | Version check mechanism (24h-throttled, non-blocking, cached in ~/.vettcode/) | 3h | MVP |

## Epic 8: Post-MVP Enhancements

| Ticket | Description | Effort | Priority |
| --- | --- | --- | --- |
| SC-080 | Optional grammar management commands (`vettcode grammar list/install/update`) | 4h | Post-MVP |
| SC-081 | Windows compatibility testing and fixes | 4h | Post-MVP |
| SC-082 | `--format terminal` (terminal only, no JSON) | 1h | Post-MVP |
| SC-083 | Duplication detection sampling for 300K+ LOC repos | 3h | Post-MVP |
| SC-084 | Additional secret patterns (expand regex library) | 2h | Post-MVP |
| SC-085 | CI/CD integration mode (`--ci` flag with exit codes based on score thresholds) | 3h | Post-MVP |

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
