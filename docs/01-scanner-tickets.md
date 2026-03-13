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
| SC-010 | Language detector (file extension + manifest scanning) | | | |
| SC-011 | Tree-sitter Go wrapper (load grammar, parse file, walk AST) | | | |
| SC-012 | Go AST wrapper (parse Go files using `go/ast`) | | | |
| SC-013 | Grammar download manager (GCS fetch, SHA-256 verify, version compatibility check, cache) | | | |
| SC-014 | File walker with exclusion filtering | | | |

## Epic 3: Core Analyzers (6.5 days)

| Ticket | Description | Status | Updated_at | Note |
| --- | --- | --- | --- | --- |
| SC-020 | Cyclomatic complexity analyzer (JS/TS via tree-sitter) | | | |
| SC-021 | Cyclomatic complexity analyzer (Python via tree-sitter) | | | |
| SC-022 | Cyclomatic complexity analyzer (Go via go/ast) | | | |
| SC-022a | Cyclomatic complexity analyzer (PHP via tree-sitter) | | | |
| SC-022b | Cyclomatic complexity analyzer (Ruby via tree-sitter) | | | |
| SC-022c | Cyclomatic complexity analyzer (Java via tree-sitter) | | | |
| SC-023 | Nesting depth analyzer (all languages) | | | |
| SC-024 | Code duplication detector (token-based, cross-language) | | | |
| SC-025 | File size distribution calculator | | | |
| SC-026 | Secrets detector (regex patterns + entropy) | | | |
| SC-027 | CVE lookup (OSV API + bundled snapshot) | | | |
| SC-028 | License detector (SPDX matching) | | | |
| SC-029 | Outdated dependency detector (registry API queries) | | | |
| SC-029a | Dependency parsing for PHP (composer.json/composer.lock) | | | |
| SC-029b | Dependency parsing for Ruby (Gemfile/Gemfile.lock) | | | |
| SC-029c | Dependency parsing for Java (pom.xml/build.gradle + lockfiles) | | | |
| SC-030 | Dependency health analyzer (age, unmaintained %) | | | |
| SC-031 | AI detection analyzer (import/dependency pattern matching) | | | |
| SC-032 | Tech stack detector (frameworks, runtimes, databases, services) | | | |
| SC-033 | Infrastructure detector (IaC, CI/CD, monitoring) | | | |
| SC-034 | Git activity analyzer (log parsing, velocity, trend) | | | |
| SC-035 | Handoff analyzer (tests, docs, env vars) | | | |

## Epic 4: Scoring, Aggregation & Red Flags (1 day)

| Ticket | Description | Status | Updated_at | Note |
| --- | --- | --- | --- | --- |
| SC-040 | Scorer implementation (metric-to-score functions) | | | |
| SC-041 | Grade conversion and category score calculation | | | |
| SC-042 | Red flag evaluator (threshold checks, flag generation) | | | |
| SC-043 | Multi-repo result aggregator | | | |
| SC-044 | Pricing tier auto-determination | | | |

## Epic 5: Output Pipeline (1 day)

| Ticket | Description | Status | Updated_at | Note |
| --- | --- | --- | --- | --- |
| SC-050 | Terminal formatter (pretty-printer matching Section 8 format, includes scan duration display) | | | |
| SC-051 | Color support with `--no-color` flag | | | |
| SC-052 | JSON serializer (schema-compliant output) | | | |
| SC-053 | Integrity signer (Ed25519 signing of metrics checksum) | | | |
| SC-055 | Remote co-signing flow (POST /cosign/init, POST /cosign/complete, nonce embedding, offline fallback, error handling for co-sign API unavailability — see Section 5.8) | | | |
| SC-054 | Progress indicator (spinner/progress bar during scan) | | | |

## Epic 6: Testing & Validation (2 days)

| Ticket | Description | Status | Updated_at | Note |
| --- | --- | --- | --- | --- |
| SC-060 | Create fixture repos (healthy-saas, neglected-project, security-nightmare, java-enterprise, tier2-only) covering all 6 Tier 1 languages | | | |
| SC-061 | Unit tests for all analyzers | | | |
| SC-062 | Unit tests for scorer and red flag evaluator | | | |
| SC-063 | Integration tests (multi-lang, multi-repo, JSON validation) | | | |
| SC-064 | E2E tests (CLI commands, fixture repo scans) | | | |
| SC-065 | Accuracy validation against reference tools (ESLint, radon, gocyclo, phpmetrics, flog, PMD) | | | |
| SC-066 | Performance benchmarks setup | | | |

## Epic 7: Distribution (0.5 days)

| Ticket | Description | Status | Updated_at | Note |
| --- | --- | --- | --- | --- |
| SC-070 | GoReleaser configuration (cross-compilation, GitHub Releases) | | | |
| SC-071 | Dockerfile for Docker image | | | |
| SC-072 | Install script (`get.vettcode.com`) | | | |
| SC-073 | Homebrew tap formula | | | |
| SC-074 | Version check mechanism (24h-throttled, non-blocking, cached in ~/.vettcode/) | | | |

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
