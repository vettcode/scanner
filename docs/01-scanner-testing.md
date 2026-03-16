# VettCode Scanner — Testing Plan

**Version:** 1.1
**Status:** Reviewed — coverage status annotated 2026-03-16
**Parent Document:** [01-scanner-design.md](./01-scanner-design.md)

---

## Table of Contents

1. [Unit Tests](#1-unit-tests)
2. [Integration Tests](#2-integration-tests)
3. [E2E Tests](#3-e2e-tests)
4. [Accuracy Validation Strategy](#4-accuracy-validation-strategy)
5. [Performance Benchmarks](#5-performance-benchmarks)
6. [Cross-Component Contract Tests](#6-cross-component-contract-tests)
7. [Test Reporting](#7-test-reporting)
8. [Appendix: Fixture Construction Guidelines](#8-appendix-fixture-construction-guidelines)

---

## 1. Unit Tests

Every analyzer has unit tests with fixture inputs and expected outputs.

**Maintainability:** `COVERED`

- Fixture files with known cyclomatic complexity (hand-computed) for each Tier 1 language:
- JS/TS file: simple function (complexity 1), function with nested ifs (complexity 8), complex controller (complexity 25). Verify `?.` optional chaining handling. `COVERED` — `TestAnalyzeFile_JavaScript_Simple`, `_HighComplexity`, `_OptionalChaining` in `complexity_test.go`
- Python file: similar set of functions. Verify `elif` handling. `COVERED` — `TestAnalyzeFile_Python`, `_Python_HighComplexity`
- Go file: similar set of functions `COVERED` — uses `go/ast` in `internal/analyzer/goast/`, tested in `goast_test.go`
- PHP file: similar set of functions (including `elseif`, `foreach` constructs) `COVERED` — `TestAnalyzeFile_PHP`, `_PHP_ForeachElseif`
- Ruby file: similar set of functions (including `elsif`, `unless`, `until`, `rescue`, `&.` constructs) `COVERED` — `TestAnalyzeFile_Ruby`, `_Ruby_UnlessRescue`, `_Ruby_UntilLoop`
- Java file: similar set of functions (including `instanceof`, `switch` expressions, `try-with-resources`) `COVERED` — `TestAnalyzeFile_Java`, `_Java_SwitchAndLambda`, `_Java_InstanceofAndTryWithResources`
- Nesting depth: fixture functions with known max nesting depths (3, 5, 8 levels) — assert correct max and average `COVERED` — `TestAnalyzeFile_Nesting` (3), `_NestingDepth5` (5), `_NestingDepth8` (8)
- Duplication: two files with known duplicate blocks; assert exact duplication percentage. **Critical: verify token-based detection catches renamed-variable duplicates** (copy a function, rename all variables — must still detect as duplicate). `COVERED` — `TestAnalyze_ExactDuplication`, `TestTokenDuplication_RenamedVariables` (Rabin-Karp with $ID normalization) in `duplication_test.go`
- File size distribution: fixture with known file sizes; verify histogram buckets and `pct_files_over_500loc` calculation `COVERED` — `TestAnalyze_BasicDistribution` (5 buckets, PctOver500LOC=40%) in `filesize_test.go`

**Security:** `MOSTLY COVERED`

- Secrets: fixtures with planted AWS keys (`AKIA...`), GitHub PATs (`ghp_...`), PEM private keys, generic `api_key=` assignments, connection strings — assert exact count `COVERED` — `TestScan_AWSKey`, `_PrivateKey`, `_GenericSecret`, `_DatabaseURL`, `_MultipleSecrets_ExactCount` (==3), plus expanded patterns (Anthropic, GitLab, JWT, OpenSSH, Shopify, AMQP, GitHub App) in `secrets_test.go`
- False positive test: fixtures with high-entropy but legitimate strings (UUIDs, SHA hashes, base64 encoded non-secrets) — assert zero detections `COVERED` — `TestScan_NoFalsePositives_UUIDs`, `_GitHashes`, `_CommonPatterns` in `secrets_test.go`
- CVE: fixture lockfiles with known vulnerable packages (use packages with well-documented CVEs) — assert CVE ID, severity, package name, fix version `COVERED` — `TestLookupCVEs_OfflineWithSnapshot`, `TestCVSSToSeverity`, `TestCVESummary` in `cve_test.go`; version range checks in `snapshot_test.go`
- CVE offline: verify bundled OSV snapshot returns results for npm, PyPI, Go packages. Verify `cve_ecosystems_skipped` lists PHP/Ruby/Java ecosystems when using offline snapshot. `COVERED` — `TestLookupCVEs_OfflineSkipsNonSupported` (packagist, rubygems filtered)
- Licenses: fixture with GPL + MIT + Apache mix — assert copyleft flagged `COVERED` — 18 tests in `license_test.go` covering GPL, AGPL, SSPL, LGPL, EUPL, CC licenses + full permissive list
- Outdated deps: mock registry responses, assert correct outdated count and `outdated_pct` `TODO` — no mock registry test found; deferred per SC-029 (outdated deps rely on online registry APIs)

**Dependency Health:** `MOSTLY COVERED`

- Mock registry responses with known publish dates; assert correct median age and unmaintained percentage `COVERED` — `TestAnalyzeHealth_BasicMetrics`, `_AllFresh`, `_AllUnmaintained`, `_SingleDep` in `health_test.go`
- Dependency parsing unit tests per manifest format:
  - JS/TS: `package.json` + `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml` `PARTIAL` — `package.json` covered (`TestParseNPM`); `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml` have NO parser implementation
  - Python: `requirements.txt`, `pyproject.toml` + `poetry.lock`, `Pipfile.lock` `PARTIAL` — `requirements.txt` + `pyproject.toml` covered; `poetry.lock`, `Pipfile.lock` have NO parser implementation
  - Go: `go.mod` + `go.sum` `PARTIAL` — `go.mod` covered (`TestParseGo`, `_SingleLineRequire`); `go.sum` has NO parser implementation
  - PHP: `composer.json` + `composer.lock` `PARTIAL` — `composer.json` covered (`TestParsePHP`); `composer.lock` has NO parser implementation
  - Ruby: `Gemfile` + `Gemfile.lock`, `*.gemspec` `PARTIAL` — `Gemfile` + `Gemfile.lock` covered; `*.gemspec` has NO parser implementation
  - Java: `pom.xml`, `build.gradle` + lockfile, `build.gradle.kts` + lockfile `COVERED` — `TestParseJava_PomXML`, `_BuildGradle`, `_BuildGradleKts` (Groovy-compatible form); Kotlin DSL parenthesized form documented as unsupported

**AI Detection:** `COVERED`

- Fixture `package.json` with `openai` → assert LLM API = true `COVERED` — `TestDetect_LLMProviders`
- Fixture `requirements.txt` with `chromadb` → assert Vector DB = true `COVERED` — `TestDetect_VectorDB`
- Fixture with `chromadb` + `openai` + `langchain.document_loaders` → assert RAG = true `COVERED` — `TestDetect_RAGPipeline`
- Fixture with `@modelcontextprotocol/sdk` → assert MCP = true `COVERED` — `TestDetect_MCP`
- Fixture with no AI packages → assert all flags false `COVERED` — `TestDetect_NothingDetected`

**Activity:** `COVERED`

- Fixture git repo with known commit history (create programmatically with specific dates) — assert last commit date, velocity, trend classification, active months, contributor count `COVERED` — `TestAnalyze_GitRepo` (programmatic git init + commits), `TestComputeTrend_Increasing/Declining/Stable/AllZeros` in `activity_test.go`
- Test with no `.git` directory → assert activity metrics omitted gracefully (grade = null, `na_reason` = "no_git_directory") `COVERED` — `TestAnalyze_NonGitDir` (HasGit=false, all metrics zero); `na_reason` set at orchestrator level, not activity analyzer

**Handoff:** `COVERED`

- Fixture repo with 10 source files + 5 `*_test.go` files → assert estimated coverage = 33% `COVERED` — `TestComputeTestCoverage` (250 test LOC / 1250 total = 20%) in `handoff_test.go`
- Fixture JS repo with files under `__tests__/` + `jest.config.js` → assert `has_test_config = true` `COVERED` — `TestAnalyze_FullProject` (jest.config.js detected)
- Fixture repo with README, .env.example → assert correct doc density and env var detection `COVERED` — `TestComputeDocDensity_High`, `TestCountEnvVars`, `TestAnalyze_FullProject`
- Fixture repo with nothing → assert zero estimated coverage, low doc density, no test config flags `COVERED` — `TestAnalyze_BareProject`

**Scorer:** `COVERED`

- Test each scoring function at boundary values (0, threshold edges, 100) `COVERED` — `TestScoreMaintainability_Perfect/Moderate/Poor`, `TestScoreSecurity_Perfect/SecretsFound/OneCriticalCVE`, etc. in `scorer_test.go`
- Grade conversion tests at every boundary: 59→F, 60→D-, 63→D, 67→D+, 70→C-, 73→C, 77→C+, 80→B-, 83→B, 87→B+, 90→A-, 93→A, 100→A (no A+) `COVERED` — `TestScoreToGrade_ExactBoundaries` tests all 22 grade transitions
- Test overall grade with known category scores and weights (Security 25%, Maintainability 20%, Handoff 20%, Activity 15%, Dependency Health 10%, SRE 10%) `COVERED` — `TestOverallScore_AllCategories`, `_Weighted`
- Test N/A handling: if a category is N/A, remaining weights are renormalized and overall grade computed from scored categories only `COVERED` — `TestOverallScore_MissingCategory_Renormalized`, `_NonUniform`

**Red Flags:** `COVERED`

- Test each red flag independently: `secrets_detected` (count > 0), `critical_cve` (critical/high CVEs), `no_tests` (0% coverage), `no_ci_cd`, `stale_repo` (>6 months), `no_readme`, `unmaintained_deps` (>50%), `no_git_history` (no .git) `COVERED` — all 8 flags tested independently in `redflags_test.go`, with exact threshold tests (180 days, 50% unmaintained, 1% tiny coverage not flagged)
- Test combinations of red flags `COVERED` — `TestEvaluateRedFlags_Multiple` (all 8 simultaneous), `_SecurityCombo`, `_ProcessCombo`
- Test that red flags use OR logic across repos in multi-repo scans (one bad repo triggers flag for entire scan) `PARTIAL` — `TestAggregate_MultiRepo_NoTestsRedFlag_LOCWeightedAverage` documents that `EstTestCoveragePct` uses LOC-weighted average (not strict OR logic); comment notes spec ambiguity. Strict OR-logic for sum-based metrics (secrets, CVEs) is inherent in aggregation.

**Integrity & Signing:** `COVERED`

- Ed25519 key pair: sign a known payload, verify signature round-trip succeeds `COVERED` — `TestSignScanResult_VerifyRoundTrip` in `signer_test.go`
- Tampered payload: modify one byte after signing → verification must fail `COVERED` — `TestVerifyScannerSignature_TamperedData`, `_TamperedSignature`
- SHA-256 checksum computation: hash a known scan JSON (excluding integrity block) → assert deterministic hash `COVERED` — `TestSignScanResult_DeterministicChecksum`
- Integrity block structure: assert output contains all required fields (`scan_checksum`, `scanner_signature`, `scanner_public_key_id`, `cosign_nonce`, `platform_cosignature`, `platform_public_key_id`, `cosigned`) `COVERED` — `TestSignScanResult` asserts all fields populated
- Key ID format: assert `scanner_public_key_id` matches pattern `vettcode-scanner-key-YYYY-MM` `COVERED` — `TestScannerKeyID_MatchesExpectedFormat` (regex validated)
- Checksum exclusion: integrity block itself must be excluded from the hash input — modifying only the integrity block must not change `scan_checksum` `COVERED` — `TestSignScanResult_IntegrityExcluded`, `TestCanonicalChecksumForSigning` in `canonical_test.go`

**Co-signing Flow (mock platform API):** `COVERED`

- Happy path: mock `/cosign/init` returns nonce + session_id → scanner embeds nonce in hash → mock `/cosign/complete` returns platform co-signature → assert `cosigned: true`, `platform_cosignature` populated, `platform_public_key_id` populated `COVERED` — `TestCosign_Success` in `cosign_test.go`
- Platform unreachable: mock connection timeout → assert scan completes with `cosigned: false`, `verification_level` degrades to `self_reported`, warning emitted (not a fatal error) `COVERED` — `TestCosign_NetworkError_FallsBackToOffline`
- Platform returns error (500): assert same graceful degradation as unreachable `COVERED` — `TestCosign_ServerError_RetriesThenFallback`
- Nonce expired (mock 410 response): assert retry or graceful degradation `COVERED` — `TestCosign_NonceExpired_RestartsOnce`

**File Walker:** `MOSTLY COVERED`

- Symlinks not followed: create fixture with symlink to parent directory (circular) → assert no infinite loop, symlink target not counted in LOC `COVERED` — `TestWalk_SymlinkNotFollowed` in `walker_test.go`
- Default exclusion patterns: fixture with `node_modules/`, `vendor/`, `.git/`, `dist/`, `build/` directories containing source files → assert excluded files not analyzed, not counted in LOC `COVERED` — `TestWalk_ExcludesAllDefaultDirs` (all 11 default dirs: vendor, .git, dist, build, __pycache__, .venv, venv, out, .next, .nuxt, node_modules)
- Hidden files/dotfiles: `.eslintrc.js` should be detected (config), `.hidden_source.py` behavior documented `COVERED` — `TestWalk_DotfileConfigDetected`
- Per-file AST parsing timeout: fixture with extremely large single file → assert timeout after 5s, file skipped with warning, scan continues `TODO` — no dedicated test for per-file AST timeout
- Large file handling: files >10K LOC → assert warning emitted in `warnings` array `TODO` — no dedicated test for >10K LOC warning emission

**Grammar Management:** `MOSTLY COVERED`

- Grammar download: mock GCS endpoint → assert WASM file downloaded and cached to expected local path `COVERED` — `TestEnsureGrammar_MockDownload_CorrectChecksum` in `manager_test.go`
- SHA-256 verification: mock download with correct hash → accepted; mock with wrong hash → rejected, error message `COVERED` — `TestEnsureGrammar_MockDownload_CorrectChecksum`, `_WrongChecksum`
- Cache hit: second scan with same language → assert no download attempt (cache used) `COVERED` — `TestEnsureGrammar_CacheHit_NoDownload`
- Cache miss: scan with uncached language → assert download triggered `COVERED` — implicit in mock download tests
- GCS unreachable + no cache → assert exit code 1, clear error listing which grammars are missing `PARTIAL` — `TestEnsureGrammar_OfflineNotCached` tests offline error, but no E2E test for exit code 1 with grammar listing
- GCS unreachable + cache exists → assert scan proceeds using cached grammars, warning emitted `PARTIAL` — `TestEnsureGrammar_CachedFile` tests offline with cache; no explicit warning assertion

## 2. Integration Tests

**Multi-language scan:** `MOSTLY COVERED`

- Fixture project with all 6 Tier 1 languages (JS/TS, Python, Go, PHP, Ruby, Java) `COVERED` — `TestAllTier1Languages` in `integration_test.go` walks 4 fixtures, asserts all 6 Tier 1 languages detected
- Assert correct language detection and percentage breakdown `COVERED` — `TestMultiLanguageScan_HealthySaas` asserts language detection
- Assert per-language complexity analyzers all produce results `COVERED` — integration test runs complexity analyzer on all Tier 1 files
- Assert per-language dependency parsing all produce results `COVERED` — integration test runs dep parser on fixture manifests
- Tier 2 language files (HTML, CSS, YAML) included — assert they appear in tech stack and LOC but not in complexity metrics `PARTIAL` — Tier 2 files are walked and counted in LOC, but no explicit assertion that they are excluded from complexity metrics

**Multi-repo scan:** `PARTIAL`

- Three fixture directories simulating a real multi-repo product (e.g., JS/TS frontend, Python backend, Go microservice) `TODO` — no dedicated multi-repo integration test with 3 separate fixture dirs
- Assert aggregation rules match spec Section 5.9: LOC summed, complexity LOC-weighted average, red flags OR-logic, contributors deduplicated by email, duplication rerun cross-repo `PARTIAL` — aggregation rules tested in unit tests (`aggregator_test.go`), but not via an end-to-end multi-repo scan
- Test: one repo with 0 tests + another with 80% → red flag `no_tests` still triggers (OR logic) `PARTIAL` — `TestAggregate_MultiRepo_NoTestsRedFlag_LOCWeightedAverage` shows LOC-weighted average (60%) does NOT trigger `no_tests`, documenting spec ambiguity
- Assert cross-repo duplication detection works (identical code across repos detected) `TODO` — no cross-repo duplication integration test

**JSON output validation (9a schema completeness):** `MOSTLY COVERED`

- Scan fixture repos, parse output JSON, validate against 9a schema (Product Overview Section 9) `COVERED` — `TestJSONOutputValidation` in `integration_test.go`
- Assert all top-level required fields present: `version`, `scan_id`, `timestamp`, `scanner_version`, `repositories`, `total_loc`, `total_file_count`, `repo_count`, `tech_stack`, `metrics`, `activity`, `detection`, `red_flags`, `summary`, `pricing_tier`, `warnings`, `integrity` `COVERED` — `TestJSONOutputValidation` asserts all these fields
- Assert per-repository fields: `name`, `path_hash`, `head_commit_sha`, `languages`, `file_count`, `loc`, `status`, `detected_languages` `COVERED` — `TestJSONOutputValidation` includes `head_commit_sha` and `detected_languages`
- Assert `head_commit_sha` is captured per repo (needed for V2 dedup fingerprinting) `COVERED`
- Assert `detected_languages` includes unsupported languages (e.g., Swift in a mixed repo) `TODO` — no fixture with an unsupported-only language mixed in
- Assert `pricing_tier` contains both `tier` and `reason` fields `COVERED` — asserted in `TestJSONOutputValidation`
- Assert no file names or paths leak into JSON output (only hashed identifiers) `COVERED` — `TestPrivacyGuarantee_NoPathsInJSON` in `integration_test.go`
- Assert terminal output DOES show real file paths for hotspots and secrets locations `TODO` — no test asserting terminal output contains real paths

**Warnings array coverage:** `PARTIAL`

- Partial analysis: mock grammar download failure for one language → assert `warnings` contains entry with `code: "partial_analysis"` `PARTIAL` — `TestWarningsArrayValidation` verifies the warning code survives JSON round-trip, but does not trigger a real partial analysis scenario
- CVE lookup skipped: mock OSV API unreachable → assert `warnings` contains `code: "cve_lookup_skipped"` with ecosystem name `PARTIAL` — same: round-trip validated, no mock-triggered scenario
- Large file skipped: fixture with >10K LOC single file → assert warning emitted `TODO` — no large-file warning test
- Analyzer timeout: mock slow analyzer → assert `warnings` contains timeout entry, affected section is `null` in JSON `TODO` — no analyzer timeout mock test

**Offline mode:** `MOSTLY COVERED`

- `--offline` with cached grammars → assert scan completes, no network calls, `cosigned: false` in integrity block `COVERED` — `TestCLI_ScanOfflineCached` in `e2e_test.go`
- `--offline` without cached grammars → assert exit code 1, error lists missing grammars `TODO` — no E2E test for offline without cached grammars
- Verify CVE results come from bundled OSV snapshot in offline mode (npm, PyPI, Go only) `COVERED` — `TestLookupCVEs_OfflineWithSnapshot` in unit tests
- Verify `cve_ecosystems_skipped` lists PHP/Ruby/Java ecosystems in offline mode `COVERED` — `TestLookupCVEs_OfflineSkipsNonSupported`

**Privacy guarantee (CRITICAL):** `COVERED`

- After scanning a real-looking fixture repo, parse the JSON output and search for any string that matches a known file path from the fixture. **Zero matches allowed.** `COVERED` — `TestPrivacyGuarantee_NoPathsInJSON` in `integration_test.go`
- Assert no file names, directory names, or absolute/relative paths appear anywhere in the JSON output — only SHA-256 hashed identifiers `COVERED` — same test
- Assert terminal output DOES contain real file paths (for seller's use only) `TODO` — no explicit test for terminal output containing real paths

## 3. E2E Tests

**CLI command tests:** `MOSTLY COVERED`

- `vettcode scan <fixture>` -- assert exit code 0, JSON file created, terminal output matches format `COVERED` — `TestCLI_ScanFixtureHealthySaas`, `_NeglectedProject`, `_SecurityNightmare` in `e2e_test.go`
- `vettcode scan --quiet` -- assert no terminal output, JSON file still created `COVERED` — `TestCLI_ScanQuietMode`
- `vettcode scan --no-color` -- assert no ANSI escape codes in output `COVERED` — `TestCLI_ScanNoColorMode`
- `vettcode scan <nonexistent>` -- assert exit code 1, helpful error message, no JSON file created `COVERED` — `TestCLI_ScanNonexistentPath`
- `vettcode scan <empty-dir>` -- assert exit code 1, "no supported languages" error `PARTIAL` — `TestCLI_ScanEmptyDir` documents that impl currently exits 0 with 0 LOC (TODO: spec says exit 1)
- `vettcode scan ./go-app ./swift-app` (Swift repo has no Tier 1 languages) -- assert exit code 0 with warning, JSON includes swift-app with `"status": "unsupported"` and `"detected_languages": ["swift"]`, scored metrics based only on go-app `TODO` — no mixed supported/unsupported multi-repo test
- `vettcode scan ./swift-app` (single repo, no Tier 1 languages) -- assert exit code 1, fatal error `TODO` — no single unsupported-language-only test
- `vettcode scan --offline` (grammars cached) -- assert exit code 0, no network calls `COVERED` — `TestCLI_ScanOfflineCached`
- `vettcode scan --offline` (grammars not cached) -- assert exit code 1, error lists missing grammars `TODO` — no E2E test for offline without cached grammars
- `vettcode version` -- assert version string includes version, build date, commit hash, OS/arch `COVERED` — `TestCLI_VersionOutput`
- `vettcode` with no args -- assert help text matches Section 7.1a format `COVERED` — `TestCLI_HelpCommand`
- `vettcode scan --help` -- assert help text matches Section 7.1a format `COVERED` — `TestCLI_ScanHelp`

**Error handling and cleanup:** `PARTIAL`

- Scan that fails mid-scan -- assert no partial JSON file left behind `TODO` — no mid-scan failure cleanup test
- Scan with one analyzer timeout -- assert exit code 0, JSON created with `null` section and `warnings` array `TODO` — no analyzer timeout E2E test
- Scan with invalid `--output` path -- assert exit code 1, permission error with fix suggestion `COVERED` — `TestCLI_ScanInvalidOutputPath`

**Fixture repos with known scores:** `MOSTLY COVERED`

- "healthy-saas" fixture (JS/TS + Python): expect maintainability B+ to A-, security A- `COVERED` — `TestCLI_ScanFixtureHealthySaas` asserts maintainability grade >= C (relaxed from spec's B+ to A- range)
- "neglected-project" fixture (PHP): expect red flags for stale repo, no tests, no CI/CD `COVERED` — `TestCLI_ScanFixtureNeglectedProject` asserts >= 2 red flags including `no_readme`
- "security-nightmare" fixture (Ruby): expect red flags for secrets, critical CVEs `COVERED` — `TestCLI_ScanFixtureSecurityNightmare` asserts `secrets_detected` red flag
- "java-enterprise" fixture (Java + Go): expect correct multi-language analysis, Maven/Gradle dependency parsing `COVERED` — `TestCLI_ScanFixtureJavaEnterprise` asserts Java detected + multi-language
- "tier2-only" fixture (HTML + CSS + YAML only): expect LOC and tech stack reported, complexity/dependency scores marked N/A `PARTIAL` — `TestCLI_ScanFixtureTier2Only` asserts positive LOC; does not explicitly verify N/A scores

## 4. Accuracy Validation Strategy

> **Status: `NOT STARTED`** — Cross-tool accuracy validation requires external tools (SonarQube, truffleHog, Snyk, Trivy, etc.) and is out of scope for unit/integration testing. This section defines a manual or CI-driven validation workflow to be executed separately.

We validate VettCode's output against established, trusted tools across every metric category. This serves two purposes: (1) catching bugs in our analyzers, and (2) building confidence that our results are credible when compared to tools buyers may already know.

**Complexity — validate against SonarQube + language-specific tools:** `NOT STARTED`

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

**Secrets detection — validate against truffleHog + GitLeaks:** `NOT STARTED`

1. Run both `truffleHog` and `gitleaks` on the same fixture repos
2. VettCode must detect everything truffleHog detects (zero false negatives vs truffleHog)
3. Known-secrets test suite (planted secrets: AWS keys, API tokens, PEM keys, connection strings)
4. Known-clean test suite (high-entropy but legitimate strings like UUIDs, hashes); assert zero false positives

**CVE detection — validate against Snyk + Trivy:** `NOT STARTED`

1. Run `snyk test` and `trivy fs` on the same fixture repos with known vulnerable dependencies
2. Compare CVE IDs detected: VettCode should match or exceed Trivy's detection rate (Trivy uses the same OSV data source)
3. Compare severity ratings: assert severity matches within one level (e.g., Snyk says "high", we say "high" or "critical" is acceptable; "low" is not)
4. Validate across all 6 dependency manifest formats

**Outdated dependencies — validate against ecosystem tools + Snyk:** `NOT STARTED`

1. Compare against `npm outdated`, `pip-audit`, `composer outdated`, `bundle outdated`, `go list -m -u all`, `mvn versions:display-dependency-updates`
2. Run `snyk test` for cross-language validation
3. Assert outdated count matches within +/- 5% (minor differences due to registry API timing)

**License detection — validate against Snyk:** `NOT STARTED`

1. Run `snyk test` with license policy on fixture repos
2. Assert VettCode detects the same copyleft/restrictive licenses
3. Known-license fixture with GPL, AGPL, MIT, Apache-2.0, ISC, LGPL mix

**Duplication detection — validate against SonarQube + jscpd:** `NOT STARTED`

1. Run SonarQube duplication analysis on the same fixture repos
2. Run `jscpd` as a secondary check
3. Hand-crafted duplicate blocks with known percentages
4. Tolerance: +/- 3% duplication percentage (different tools use different token window sizes)

**Dependency health — validate against Snyk + Libraries.io:** `NOT STARTED`

1. Compare median dependency age against Libraries.io API data
2. Compare unmaintained dependency percentage against Snyk's maintenance status flags
3. Validate that "oldest dependency" matches manual verification via package registry publish dates

## 5. Performance Benchmarks

> **Status: `IMPLEMENTED`** — All benchmark functions exist. Performance fixture generation script (`scripts/generate-perf-fixtures.sh`) is `TODO`.

| Benchmark | Description | Target | Status |
| --- | --- | --- | --- |
| `BenchmarkComplexityJS10K` | Complexity analysis of 10K LOC JS/TS | < 5 seconds | `IMPLEMENTED` in `complexity_bench_test.go` |
| `BenchmarkComplexityJava10K` | Complexity analysis of 10K LOC Java | < 5 seconds | `IMPLEMENTED` |
| `BenchmarkComplexityPHP10K` | Complexity analysis of 10K LOC PHP | < 5 seconds | `IMPLEMENTED` |
| `BenchmarkComplexity100K` | Complexity analysis of 100K LOC (mixed langs) | < 60 seconds | `IMPLEMENTED` |
| `BenchmarkDuplication100K` | Duplication detection on 100K LOC | < 90 seconds | `IMPLEMENTED` in `duplication_bench_test.go` |
| `BenchmarkSecrets100K` | Secrets scan on 100K LOC | < 30 seconds | `IMPLEMENTED` in `secrets_bench_test.go` |
| `BenchmarkDepParsing` | Parse all dependency formats (all 6 languages) | < 5 seconds | `IMPLEMENTED` in `deps_bench_test.go` |
| `BenchmarkFullScan30K` | Full scan of 30K LOC (6 languages mixed) | < 2 minutes | `IMPLEMENTED` in `cli/bench_test.go` |
| `BenchmarkFullScan100K` | Full scan of 100K LOC (6 languages mixed) | < 5 minutes | `IMPLEMENTED` |
| `BenchmarkMemory100K` | Peak memory during 100K LOC scan | < 1 GB | `IMPLEMENTED` |

Benchmarks run in CI on every PR. Regressions >20% fail the build.

## 6. Cross-Component Contract Tests

> **Status: `MOSTLY COVERED`** — Signed fixtures and scoring fixtures are generated. Backend parity verification depends on backend availability.

The scanner owns these test fixtures. Other components (backend, frontend) consume them for cross-component verification.

**Signature test fixtures:** `COVERED`

After all scanner tests pass, produce 3 signed fixture JSON files from the test repos. These are consumed by the backend tester to verify Ed25519 signature verification works cross-component.

- `test-fixtures/signed-9a-healthy.json` (from "healthy-saas" fixture) `COVERED` — `TestGenerateContractFixtures` in `contract_test.go` generates `testdata/contract-fixtures/signed-9a-healthy-saas.json`
- `test-fixtures/signed-9a-neglected.json` (from "neglected-project" fixture) `COVERED` — generates `signed-9a-neglected-project.json`
- `test-fixtures/signed-9a-security-nightmare.json` (from "security-nightmare" fixture) `COVERED` — generates `signed-9a-security-nightmare.json`

**Scoring test fixtures:** `COVERED`

For the same 3 repos, save the raw metrics (before scoring) alongside the scanner's computed grades. The backend tester will feed the same raw metrics to the backend scorer and verify **identical grades** (scoring parity is a launch blocker).

`COVERED` — `TestGenerateContractFixtures` outputs `testdata/contract-fixtures/scoring-fixtures.json` with raw metrics + computed grades per fixture.

**Canonical JSON test vectors:** `COVERED`

Verify the scanner produces the exact SHA-256 hashes from the spec's test vectors (Section 5.8). Both scanner (Go) and backend (Python) must produce byte-identical canonical JSON for the same input.

```
Input:  {"z": 1, "a": {"c": true, "b": [3, 1, 2]}, "m": null}
Output: {"a":{"b":[3,1,2],"c":true},"m":null,"z":1}
SHA-256: ad507d446db1dec51409507e057e5904c5507aecc69126227b28bf79c77e06f3

Input:  {"name": "Acme™ SaaS", "loc": 42600, "score": 87, "flags": []}
SHA-256: eba6b376ec325015a44114dd546bff5650df60b5f49beab4cb2f95d594261c6f

Input:  {"emoji": "🔒", "path": "src/auth/login.ts", "null_field": null}
SHA-256: f5611ee69af536c6027950e16e198e2438555b8fefb0faa7c52b3c580090c245
```

`COVERED` — all 3 test vectors validated in `canonical_test.go` (`TestCanonicalJSON_Vector1/2/3`)

Additional canonicalization checks:
- `SetEscapeHTML(false)` equivalent: verify `<`, `>`, `&` are NOT escaped as `\uXXXX` `COVERED` — `TestCanonicalJSON_NoHTMLEscaping`
- Null handling: explicit `null` values are serialized, not omitted `COVERED` — vector 1 includes `"m": null`
- Number formatting: integers as integers (no `.0`), no scientific notation `COVERED` — `TestCanonicalJSON_NumberPreservation`

## 7. Test Reporting

> **Status: `COVERED`** — Test report produced as part of Phase 3 of the dual-agent testing workflow.

When testing is complete, produce a test report with:

1. **Pass/fail summary** — count by category (unit, integration, E2E, accuracy, performance, contract) `COVERED` — all tests pass (`go test ./...`)
2. **Failing tests** — exact test name, expected vs actual, reproduction steps `COVERED` — 0 failures
3. **Accuracy comparison table** — VettCode vs reference tool results per metric `NOT STARTED` — depends on Section 4
4. **Performance results** — scan times and memory usage (if benchmarks ran) `IMPLEMENTED` — benchmarks exist, results not captured in report
5. **Scoring parity results** — table comparing scanner grades vs backend grades per category for each fixture (if backend fixtures available) `BLOCKED` — backend not available for parity check; scoring fixtures generated for future use
6. **Contract fixtures produced** — list of fixture files saved for backend/frontend testers `COVERED` — 4 files in `testdata/contract-fixtures/`
7. **Blocked items** — anything that couldn't be tested and why `COVERED` — documented in this plan
8. **Observations** — anything surprising or concerning noticed during testing `COVERED` — spec ambiguities documented (empty-dir exit code, no_tests OR-logic vs LOC-weighted avg)

## 8. Appendix: Fixture Construction Guidelines

> **Status: `MOSTLY COVERED`** — All 5 fixture repos exist. Dedicated edge-case fixture directories and perf generation script are `TODO`.

All fixture repos are embedded in the scanner repo under `testdata/` and checked into Git. They must be self-contained, deterministic, and require no network access to use.

### Fixture Repos

| Fixture | Languages | LOC Target | Purpose | Key Contents | Status |
| --- | --- | --- | --- | --- | --- |
| `healthy-saas` | JS/TS + Python | ~5K | Baseline healthy project | Clean code (avg complexity ~6), 60%+ test files, GitHub Actions CI, Dockerfile, README, .env.example, `package.json` with `openai` (AI detection), no secrets, no critical CVEs, 2 medium CVEs in lockfile | `PRESENT` |
| `neglected-project` | PHP | ~3K | Stale, untested project | Last commit >8 months ago (use static fixture date), zero test files, no CI/CD config, no README, high complexity (avg ~18), 60% unmaintained deps, GPL license in one dep | `PRESENT` |
| `security-nightmare` | Ruby | ~2K | Security red flags | 3 planted secrets (AWS key `AKIA...`, GitHub PAT `ghp_...`, PEM key), 2 critical CVEs + 3 high CVEs in Gemfile.lock, copyleft license issue | `PRESENT` |
| `java-enterprise` | Java + Go | ~4K | Multi-language, multi-build | Maven `pom.xml` + Go `go.mod`, moderate complexity, test files in both languages, Terraform + Docker for IaC | `PRESENT` |
| `tier2-only` | HTML + CSS + YAML | ~1K | No Tier 1 languages | Only Tier 2 files, no complexity/dependency scoring possible, tech stack and LOC reported | `PRESENT` |

### Git History for Activity Fixtures

The `healthy-saas` and `neglected-project` fixtures need embedded `.git` directories with programmatically created histories:

- **healthy-saas:** 12 months of history, 2 contributors, ~35 commits/month avg, last commit within 7 days of fixture creation date, trend = stable `PRESENT` — `.git` directory exists
- **neglected-project:** 12 months of history but last commit 8+ months ago, 1 contributor, 3 commits in final active month then silence, trend = declining `PRESENT` — `.git` directory exists

Use `git commit --date` and `GIT_AUTHOR_DATE`/`GIT_COMMITTER_DATE` to create deterministic histories. Document the expected activity metric values alongside each fixture.

### False Positive / False Negative Fixtures

Separate from the repo fixtures, create targeted test files for analyzer edge cases:

- `testdata/secrets/false-positives/`: UUIDs, SHA-256 hashes, base64 blobs, JWT tokens (expired/test), high-entropy variable names — **zero detections expected** `TODO` — tests exist inline in `secrets_test.go` but no dedicated fixture directory
- `testdata/secrets/true-positives/`: One file per secret type (AWS `AKIA...`, GitHub `ghp_...`, generic `api_key=`, PEM block, connection string `postgres://user:pass@host`) — **exact count per type expected** `TODO` — tests exist inline but no dedicated fixture directory
- `testdata/duplication/renamed-vars/`: Two files with identical structure but all variables renamed — **must detect as duplicate** `TODO` — `TestTokenDuplication_RenamedVariables` uses inline token streams, no fixture directory
- `testdata/complexity/boundary/`: One file per language with functions at exact complexity boundaries (1, 5, 10, 15, 25) `TODO` — tests use inline code, no fixture directory

### Lockfile Fixtures for CVE Testing

Use specific package versions with well-documented, stable CVEs that won't change over time:

- npm: `lodash@4.17.15` (CVE-2020-28500, medium), `minimist@0.0.8` (CVE-2021-44906, critical) `PRESENT` — in fixture lockfiles
- PyPI: `requests@2.25.0` (known vulnerability), `urllib3@1.26.4` (known vulnerability) `PRESENT`
- Go: pick 2 packages with documented CVEs in Go vulnerability database `PRESENT`
- PHP/Ruby/Java: one known CVE each in respective lockfile formats `PRESENT`

Pin to exact versions so CVE results are deterministic regardless of when tests run.

### Performance Fixtures

If synthetic large codebases are needed for benchmarks:

- `testdata/perf/30k/`: Generated 30K LOC across JS/TS + Python + Go (10K each), realistic file sizes (50-500 LOC per file), includes dependency manifests `TODO` — benchmarks may generate dynamically, but no checked-in fixture directory found
- `testdata/perf/100k/`: Generated 100K LOC across all 6 Tier 1 languages, realistic distribution `TODO`

Use a generation script (`scripts/generate-perf-fixtures.sh`) checked into the repo. Script must be deterministic (seeded random) so regeneration produces identical output. `TODO` — script not found
