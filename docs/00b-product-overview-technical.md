# VettCode V1 -- Product Overview (Technical)

**Version:** 0.1-draft
**Status:** In Review
**Companion Document:** [Business Overview](./00a-product-overview-business.md)
**Index:** [Product Overview Index](./00-product-overview.md)

> Section numbers are preserved from the original unified document for cross-reference compatibility with component design docs.

---

## Table of Contents

4. [Product Architecture — High Level](#4-product-architecture--high-level)
5. [Core Workflows](#5-core-workflows)
6. [Component Summary](#6-component-summary)
7. [Scanner Metrics — What We Measure](#7-scanner-metrics--what-we-measure)
8. [Report Structure (Static Scan)](#8-report-structure-static-scan)
9. [Data Contracts (Draft)](#9-data-contracts-draft)
12. [Technical Decisions & Rationale](#12-technical-decisions--rationale)
13. [Security & Privacy Considerations](#13-security--privacy-considerations)
14b. [Infrastructure Capacity & SRE](#14b-infrastructure-capacity--sre)
15. [Open Questions for Discussion](#15-open-questions-for-discussion)
17. [Standard Design Document Template](#17-standard-design-document-template)

---

## 4. Product Architecture — High Level

> For the user personas this architecture serves, see Section 3 in the [Business Overview](./00a-product-overview-business.md).

```
+------------------------------------------------------------------+
|                        VETTCODE ECOSYSTEM                         |
+------------------------------------------------------------------+
|                                                                    |
|  [1] VETTCODE-SCANNER          [2] VETTCODE-PLATFORM              |
|  (Local CLI or Cloud Container) (Cloud - Central Brain)           |
|                                                                    |
|  +------------------------+    +-----------------------------+     |
|  | CLI (Go binary)        |    | Frontend (Next.js)          |     |
|  | ~20MB slim mode        |    |  - Landing / Marketing      |     |
|  |                        |    |  - Dashboard (Seller/Buyer)  |     |
|  | Analyzers:             |    |  - Report Viewer             |     |
|  |  - Core (built-in)     |    |  - Payment Flow              |     |
|  |  - Language plugins    |    +-----------------------------+     |
|  |    (on-demand download |    | Backend (Python/FastAPI)     |     |
|  |     or Docker bundled) |    |  - Auth (OAuth2 + JWT)       |     |
|  |                        |    |  - Report Engine             |     |
|  | Output:                |    |  - Scoring & Benchmarks      |     |
|  |  - Terminal summary    |    |  - Digital Signature (Ed25519)|    |
|  |  - JSON (full metrics) |    |  - Payment (Stripe)          |     |
|  +----------+-------------+    |  - GitHub Integration        |     |
|             |                  +-----------------------------+     |
|             | Upload JSON                                          |
|             +----------------->|                                    |
|                                                                    |
|  [3] VETTCODE-DEEP-SCAN       +-----------------------------+     |
|  (LLM-Powered Analysis)       | Infrastructure               |    |
|                                | +-------------------------------+ |
|  +------------------------+    | | GCP Cloud Run                 | |
|  | Deep Scan Engine       |    | |  - Backend API (scale-to-zero)| |
|  |  - Claude API (LLM)   |    | |  - Scan Workers (GitHub scans)| |
|  |  - Prompt Orchestrator |    | |  - Deep Scan Workers          | |
|  |  - 7 Analysis Categories|   | | GCP Cloud SQL (PostgreSQL)    | |
|  |  - Quality Validation  |    | | GCP Cloud Tasks (job queue)   | |
|  |  - Ephemeral Containers|    | | GCP Cloud Storage (Reports)   | |
|  |    (temp code access)  |    | +-------------------------------+ |
|  +------------------------+    | | Vercel (Frontend/CDN)         | |
|                                | +-------------------------------+ |
|  Note: Report verification is built into the Platform.            |
|  Open-source verify CLI deferred to V2 if marketplace partners    |
|  request it. Verification spec published as open documentation.   |
|                                                                    |
+------------------------------------------------------------------+
```

---

## 5. Core Workflows

### Workflow 1: Static Scan (Privacy-First Path)

```
Seller                          VettCode Scanner (Local)         VettCode Platform
  |                                    |                               |
  |  1. Download CLI / Docker          |                               |
  |<-----------------------------------|                               |
  |                                    |                               |
  |  2. Run: vettcode scan ./myrepo    |                               |
  |     (multi-repo: vettcode scan ./fe ./be ./infra)                  |
  |----------------------------------->|                               |
  |                                    |                               |
  |  3. Analyzer executes locally:     |                               |
  |     - Detect languages             |                               |
  |     - Pull needed analyzers*       |                               |
  |     - Run all checks               |                               |
  |     (* skip if Docker mode)        |                               |
  |                                    |                               |
  |  4. Terminal summary printed       |                               |
  |     + scan-result.json saved       |                               |
  |<-----------------------------------|                               |
  |                                    |                               |
  |  5. Upload JSON to Platform        |                               |
  |------------------------------------------------------>            |
  |                                    |               6. Generate     |
  |                                    |                  signed PDF   |
  |  7. Receive email with PDF         |                               |
  |     + platform link (UUID)         |                               |
  |<------------------------------------------------------|           |
```

Error handling: see [Platform Backend Section 5.8](./components/02-platform-backend-design.md).

### Workflow 2: GitHub-Connected Scan

```
Seller                          VettCode Platform
  |                                    |
  |  1. Connect GitHub + select repos  |
  |----------------------------------->|
  |                                    |
  |  2. Platform clones to ephemeral   |
  |     container, runs scanner,       |
  |     deletes code immediately       |
  |                                    |
  |  3. Show scan progress / status    |
  |<-----------------------------------|
  |                                    |
  |  4. Generate signed PDF report     |
  |                                    |
  |  5. Deliver email with PDF         |
  |     + platform link (UUID)         |
  |<-----------------------------------|
```

Error handling: see [Platform Backend Section 5.8](./components/02-platform-backend-design.md).

### Workflow 3: Buyer Verification (Authenticated Access)

```
Seller                          VettCode Platform              Buyer
  |                                    |                          |
  |  1. Seller shares report link     |                          |
  |     with buyer (off-platform,     |                          |
  |     e.g. email, Acquire.com msg)  |                          |
  |     Link contains UUID, e.g.     |                          |
  |     platform.vettcode.com/        |                          |
  |       reports/{uuid}              |                          |
  |                                    |                          |
  |                                    |  2. Buyer logs in        |
  |                                    |<-------------------------|
  |                                    |                          |
  |                                    |  3. Platform resolves    |
  |                                    |     UUID, shows report   |
  |                                    |------------------------->|
```

**Access control rules:**
- Reports are NEVER publicly accessible — no unauthenticated URLs
- Buyer must have a VettCode account and be logged in to view any report
- Report links use UUIDs exclusively (non-enumerable); there is no sequential report ID to prevent enumeration attacks
- V2 consideration: per-buyer access grants, audit trail, revocation

Error handling: see [Platform Backend Section 5.8](./components/02-platform-backend-design.md).

### Workflow 4: Post-LOI Deep Scan (Premium)

```
Buyer                           VettCode Platform              Seller
  |                                    |                          |
  |  1. Buyer requests deep scan       |                          |
  |     (from report page or           |                          |
  |      standalone request)           |                          |
  |----------------------------------->|                          |
  |                                    |                          |
  |                                    |  2. Platform emails      |
  |                                    |     seller: consent      |
  |                                    |     request with link    |
  |                                    |     to consent page      |
  |                                    |------------------------->|
  |                                    |                          |
  |                                    |  3. Seller opens consent |
  |                                    |     page, reviews privacy|
  |                                    |     disclosure, clicks   |
  |                                    |     [Approve] or         |
  |                                    |     [Decline]            |
  |                                    |<-------------------------|
  |                                    |                          |
  |                                    |  4. If approved: seller  |
  |                                    |     grants repo access   |
  |                                    |     via GitHub App       |
  |                                    |<-------------------------|
  |                                    |                          |
  |  5. Platform emails buyer:         |                          |
  |     "Seller approved —             |                          |
  |      complete payment"             |                          |
  |<-----------------------------------|                          |
  |                                    |                          |
  |  6. Buyer pays on platform         |                          |
  |----------------------------------->|                          |
  |                                    |                          |
  |  7. Platform clones repos,         |                          |
  |     runs LLM analysis,             |                          |
  |     deletes code immediately       |                          |
  |                                    |                          |
  |  8. Deep report delivered to       |                          |
  |     both parties (platform +       |                          |
  |     email notification)            |                          |
  |<-----------------------------------|------------------------->|
```

**Consent & notification flow:**

- **No prerequisite:** Deep scan can be requested as an add-on to an existing static scan report OR as a standalone request. If a static scan exists, the deep report references and extends it. If not, the deep scan is self-contained.
- **Payment timing:** Buyer pays **after** seller approves and grants GitHub access (step 6). No upfront payment, no refund complexity.
- **Seller consent is mandatory:** Platform sends consent request via **email only** (no in-app notification in V1). Email links to a consent page on the platform.
- **Privacy disclosure** (shown on consent page, summarized in email):
  - What deep scan involves (LLM-powered code analysis by AI)
  - Source code will be sent to Anthropic Claude for analysis
  - Which repos will be analyzed (listed explicitly)
  - Code is processed ephemerally — cloned, analyzed, deleted. Not stored, not used for LLM training.
- **Seller approves or declines:** If declined, buyer notified via email ("Seller declined your deep scan request"). If no response in **7 days**, request expires and buyer is notified via email.
- **Report access:** Both buyer and seller receive access to the deep scan report on the platform, with email notifications when the report is ready.

Error handling: see [Platform Backend Section 5.8](./components/02-platform-backend-design.md).

---

## 6. Component Summary


| Component                   | Purpose                         | Repo                      | Tech Stack                                      | Deployment                           |
| --------------------------- | ------------------------------- | ------------------------- | ----------------------------------------------- | ------------------------------------ |
| **vettcode-scanner**        | Static code analysis CLI        | `vettcode-scanner`        | Go (core) + language analyzers                  | Distributed as binary + Docker image |
| **vettcode-platform-be**    | API backend                     | `vettcode-platform-be`    | Python 3.12, FastAPI, SQLAlchemy, Cloud Tasks   | GCP Cloud Run                        |
| **vettcode-deep-scan**      | LLM-powered deep analysis engine | `vettcode-deep-scan`     | Python, Claude API, prompt orchestration        | GCP Cloud Run (ephemeral workers)    |
| **vettcode-platform-fe**    | Web frontend                    | `vettcode-platform-fe`    | Next.js 14, TypeScript, Tailwind CSS, shadcn/ui | Vercel (CDN + SSR)                   |
| **vettcode-platform-infra** | IaC + deployment                | `vettcode-platform-infra` | Terraform, GitHub Actions                       | GCP + Vercel                         |


> **Deferred to V2:** `vettcode-verify` (open-source standalone verification CLI) — will extract from platform if marketplace partners request it. Verification spec published as open docs in V1.

---

## 7. Scanner Metrics — What We Measure

**Design principle:** Static scan provides **scores and flags** (quantitative screening). Deep scan provides **analysis and recommendations** (qualitative decision-making). The static scan answers "Should I pursue this deal?" The deep scan answers "What am I inheriting technically?" Neither replaces full manual DD for large deals ($1M+) — VettCode covers the technical dimension, not team assessment, business logic correctness, or legal/contractual risk. For deals under ~$100K where manual DD is cost-prohibitive, VettCode may be the only technical DD performed.

### Static Scan Metrics (Scores + Flags — No Source Code Leaves Seller's Machine)

The static scan is designed to give buyers enough signal to make a **go/no-go decision** on pursuing a deal. Every metric here is cheap to compute (no LLM), high-signal for screening, and privacy-safe.

| Category | Metric | How | Output |
| --- | --- | --- | --- |
| **Code Maintainability** | Cyclomatic complexity (per-file avg + hotspots) | Language-specific AST analysis | Score + grade |
| | Nesting depth (max + avg) | AST analysis | Numbers |
| | File size distribution | File system scan | Histogram data |
| | Code duplication % | Token-based detection | Percentage |
| **Security Posture** | Hardcoded secrets count | Regex + entropy detection (truffleHog-style) | Count (no content revealed) |
| | Known CVEs in dependencies | OSV database lookup | Count + severity breakdown |
| | Outdated dependencies count | Package registry API checks | Count with severity |
| | License compatibility issues | SPDX license detection | Flag (yes/no) + count |
| **Dependency Health** | Median dependency age | Package registry publish dates | Score + grade |
| | % unmaintained dependencies (no update in 2+ years) | Registry API last-publish check | Percentage |
| | Oldest dependency | Registry API | Name + age |
| **AI Detection** | AI-related libraries/APIs present | Import/dependency scanning | Binary flags: "LLM API: Yes", "Vector DB: Yes", "RAG: Yes", "MCP: Yes" |
| | Proprietary data pipeline detected | ETL/data processing pattern scan | Yes/No flag |
| **Tech Stack** | Frameworks detected | Dependency file parsing (package.json, requirements.txt, go.mod) | e.g. "Next.js 14, FastAPI, PostgreSQL" |
| | Runtime versions | Config/lockfile parsing | e.g. "Node 20, Python 3.12" |
| | Database(s) detected | Dependency + config scanning | e.g. "PostgreSQL, Redis" |
| | External services detected | Import/dependency scanning | e.g. "Stripe, SendGrid, OpenAI" |
| **SRE & Infra** | IaC present | File detection (Dockerfile, Terraform, K8s) | Score + grade |
| | CI/CD present | Config file detection (GitHub Actions, GitLab CI) | Yes/No + provider |
| | Monitoring present | Dependency/config detection | Yes/No + tool name |
| **Development Activity** | Last commit date | Git log (if .git available) | Score + grade |
| | Commit velocity (last 12 months) | Git log — commits/month + trend | Trend line (increasing/stable/declining) |
| | Active development months (of last 12) | Git log — months with >0 commits | Count (e.g. "11 of 12 months") |
| | Contributor count | Git log analysis (if .git available) | Number (raw data, not scored) |
| **Handoff Readiness** | Est. test coverage % | File-ratio heuristic (test files / total files, LOC-weighted) — not execution coverage | Percentage |
| | Documentation density | README, inline comments ratio | High/Medium/Low |
| | Env var count | .env.example / config file scanning | Number (proxy for deployment complexity) |
| **Codebase Profile** | Languages and breakdown | File extension analysis | Language percentages |
| | Total LOC | Line counting (excl. vendor/generated) | Number |
| | Repository count | Scan input | Number |
| **Red Flags** | Instant deal-killers surfaced prominently | Aggregated from above metrics | List of critical findings |

**Red flag triggers** (any of these = prominently surfaced in terminal + report):
- Secrets detected (any count > 0)
- Critical or high CVEs found
- No tests found (0% coverage)
- No CI/CD detected
- Last commit > 6 months ago
- 0 documentation (no README)
- 50%+ dependencies unmaintained

**What static scan deliberately does NOT include** (reserved for deep scan):
- No AI moat scoring or wrapper assessment
- No architecture pattern analysis
- No infrastructure cost or resource analysis
- No bus factor deep analysis (contributor count shown as raw data under Development Activity, not scored)
- No API surface or DB schema analysis
- No recommendations or effort estimates

### Multi-Repo Aggregation Rules

When a seller scans multiple repositories (e.g., `vettcode scan ./frontend ./backend ./infra`), the scanner produces per-repo metrics and then aggregates them into a single report. The report shows **aggregated values** as the primary view; **per-repo breakdowns** are retained in the data model for drill-down on the platform.

**Core principles:**

- **Aggregation should never hide problems.** Grades are computed from aggregated (combined) metrics, NOT averaged from per-repo grades — averaging can mask a terrible repo behind a good one.
- **Counts are summed** (LOC, CVEs, secrets, etc.)
- **Percentages are LOC-weighted** (duplication %, est. test coverage %, etc.)
- **Detection flags use OR logic** — true if detected in any repo
- **Red flags use OR logic** — one repo with zero tests triggers the flag even if other repos are well-tested
- **Contributors are unioned** (unique by git author email, not summed)

> For the full aggregation rules table covering all metric types, see [Scanner Design, Section 5.8](./components/01-scanner-design.md#58-multi-repo-aggregation).

### Deep Scan Metrics (Post-LOI, LLM-Assisted — Requires Code Access)

| Category | Metric | How | Output |
| --- | --- | --- | --- |
| **AI Moat Analysis** | AI wrapper score (0-100) | LLM analysis of AI-integration code vs business logic | Score + narrative |
| | AI integration depth assessment | LLM evaluates RAG quality, model usage, data pipelines | Detailed report |
| | Defensibility rating | LLM assesses proprietary data, custom models, unique pipelines | Rating + explanation |
| **Architecture** | Pattern detection (monolith, microservices, etc.) | LLM code structure analysis | Classification + diagram |
| | API surface analysis | LLM endpoint detection from route definitions | Count + complexity assessment |
| | Database schema complexity | LLM migration/schema file analysis | Complexity rating + narrative |
| | Service dependency mapping | LLM traces inter-service calls | Dependency graph |
| **Code Quality** | Function-level quality assessment | LLM review of critical paths | Per-module ratings |
| | Anti-pattern detection | LLM pattern matching | List with severity + location |
| | Error handling robustness | LLM analysis of error paths | Rating + specific findings |
| **Technical Debt** | Estimated refactoring effort (person-weeks) | LLM assessment + complexity metrics | Effort estimate per area |
| | Critical path fragility | LLM identifies brittle code | Risk map |
| | Specific refactoring recommendations | LLM generates actionable tasks | Prioritized task list |
| **Security (Deep)** | Business logic vulnerabilities | LLM security audit | Findings with severity |
| | Auth/authz implementation review | LLM review of auth flows | Pass/fail + findings |
| | Data handling compliance (GDPR, SOC2 readiness) | LLM compliance pattern check | Readiness score + gaps |
| | Prioritized remediation plan | LLM generates fix recommendations | Ordered action items |
| **Infrastructure (Deep)** | Resource detection + pricing links | LLM parses IaC for instance types and services | Detected resources with provider pricing URLs |
| | Cost caveat | Automatic | Notes which costs are usage-based and cannot be estimated |
| | Scaling readiness | LLM evaluates bottlenecks, single points of failure | Rating + recommendations |
| **Post-Acquisition Risk** | Migration effort estimate | LLM analysis of coupling, vendor lock-in, dependencies | Person-weeks estimate |
| | Key-person dependency areas | LLM identifies domain-knowledge-heavy code | Knowledge concentration map |
| | Onboarding time estimate | LLM assesses docs, setup complexity, code readability | Estimated weeks for new engineer |
| | Post-acquisition roadmap | LLM generates recommended first 90 days | Actionable plan |


---

## 8. Report Structure (Static Scan)

### Terminal Output

Scanner prints a human-readable summary to terminal showing key metrics, grades, red flags, and a deep scan upsell prompt. Full results are saved to `./vettcode-scan-result.json`. Terminal output specification: see [Scanner Design Section 7.5](./components/01-scanner-design.md).

### Signed Report (Platform-Generated — PDF)

The signed report is a **PDF document** generated by the platform from the structured data defined in contracts 9b/9c. The PDF is the official paid deliverable — the JSON data contracts are the internal data model, not what buyers or sellers receive.

**PDF contents:**

- **Score breakdown with explanations**: Plain-English descriptions for non-technical buyers
- **Risk summary**: Top 5 risks with severity ratings
- **Strengths summary**: Top 5 technical strengths
- **Buyer disclosure**: Scan origin, verification level, and trust notes. For offline (non-co-signed) scans, the PDF prominently displays **"Self-Reported — Not Co-Signed"** with a buyer-facing notice: *"This scan was not co-signed by VettCode's platform. The scan data is self-reported by the seller."*
- **Digital signature info**: Ed25519 signature details for tamper-proof verification
- **Report metadata**: Scan timestamp, scanner version, report UUID
- **QR code**: Quick verification link (encodes UUID-based verify URL)
- **(V2) Market benchmarking**: How this score compares to similar-sized SaaS products (requires sufficient historical data)

**Delivery channels:**

1. **Platform viewer**: Report is viewable as a PDF document within the platform
2. **Download**: Downloadable as a PDF file from the platform
3. **Email**: After payment, seller receives an email with the PDF attached and a download link to the report on the platform

---

## 9. Data Contracts (Draft)

### 9a. Static Scan Result JSON — Output of Local Scanner

This is the JSON file generated by `vettcode scan` on the seller's machine. Contains scores, counts, and binary flags only. **No source code, file names, or file paths are included — only hashes and aggregate metrics.**

```jsonc
{
  "version": "1.0",
  "scan_id": "uuid-v4",
  "timestamp": "ISO-8601",
  "scanner_version": "1.0.0",

  // --- Scanned Repositories ---
  "repositories": [
    {
      "name": "frontend",                          // user-provided label
      "path_hash": "sha256-of-absolute-path",      // no real path leaked
      "head_commit_sha": "a1b2c3d4...",            // git HEAD commit SHA (captured for V2 dedup fingerprinting)
      "languages": { "TypeScript": 62.3, "CSS": 12.1 },
      "file_count": 342,
      "loc": 28400,
      "status": "analyzed",                          // analyzed | unsupported | error
      "detected_languages": ["TypeScript", "CSS"]    // all detected languages (including unsupported ones)
    },
    {
      "name": "backend",
      "path_hash": "sha256-of-absolute-path",
      "languages": { "Python": 89.2, "Shell": 10.8 },
      "file_count": 128,
      "loc": 14200,
      "status": "analyzed",
      "detected_languages": ["Python", "Shell"]
    }
  ],
  "total_loc": 42600,
  "total_file_count": 470,
  "repo_count": 2,

  // --- Tech Stack (auto-detected from dependency files + configs) ---
  "tech_stack": {
    "frameworks": ["Next.js 14", "FastAPI"],
    "runtimes": ["Node 20.11", "Python 3.12"],
    "databases": ["PostgreSQL", "Redis"],
    "external_services": ["Stripe", "SendGrid", "OpenAI"]
  },

  // --- Metrics (quantitative data + letter grades, no numeric scores) ---
  "metrics": {
    "maintainability": {
      "grade": "B+",
      "cyclomatic_complexity": { "avg": 12.3, "p90": 28, "max": 45 },
      "nesting_depth": { "avg": 2.1, "max": 7 },
      "duplication_pct": 4.2,
      "hotspot_count": 3,
      "hotspot_files": [
        {
          "file_hash": "sha256",     // no file name or path
          "complexity": 45,
          "loc": 380,
          "repo": "frontend"         // which repo label it belongs to
        }
      ],
      "pct_files_over_500loc": 8.3   // % of source files exceeding 500 lines (used by scoring methodology, Maintainability file size sub-metric, 15% weight)
    },
    "security": {
      "grade": "A-",
      "secrets_found": 0,
      "cves": [
        {
          "id": "CVE-2025-XXXX",
          "severity": "medium",
          "package": "lodash",
          "current_version": "4.17.15",
          "fixed_in": "4.18.0",
          "repo": "frontend"
        }
      ],
      "cve_summary": { "critical": 0, "high": 0, "medium": 1, "low": 1 },
      "outdated_deps": { "total": 48, "outdated": 7, "critical": 0, "outdated_pct": 14.6 },  // outdated_pct = (outdated / total) * 100 — used by scoring methodology
      "license_issues": [],
      "license_issue_count": 0,
      "cve_ecosystems_skipped": []    // ecosystems where CVE lookup failed (e.g., ["npm", "pypi"] if OSV was unreachable)
    },
    "dependency_health": {
      "grade": "B",
      "median_age_months": 14,
      "unmaintained_pct": 4.2,       // % of deps with no update in 2+ years
      "unmaintained_count": 2,
      "oldest": {
        "package": "moment.js",
        "age_years": 4.2,
        "repo": "frontend"
      }
    },
    "handoff_readiness": {
      "grade": "C+",
      // Scoring weights: est_test_coverage 50%, doc_density 25%, env_vars 25%
      // Contributor count is NOT scored — shown as raw data under dev_activity only
      "est_test_coverage_pct": 42,   // File-ratio heuristic, not execution coverage
      "doc_density": "low",          // high | medium | low
      "env_var_count": 14,           // proxy for deployment complexity
      "has_readme": true,
      "has_contributing_guide": false,
      "has_env_template": true,
      "has_setup_script": false
    }
  },

  // --- Development Activity (from git history, if .git available) ---
  "activity": {
    "grade": "A-",
    "last_commit_date": "2026-03-03",
    "days_since_last_commit": 3,
    "commit_velocity": {
      "avg_per_month": 38,
      "trend": "stable",            // increasing | stable | declining
      "last_12_months": [42, 38, 35, 40, 44, 38, 36, 39, 41, 37, 33, 38]
    },
    "active_months": 11,             // months with >0 commits in last 12
    "total_months": 12,
    "contributor_count": 2           // Raw data only — NOT included in handoff scoring
  },

  // --- Detection Flags (binary yes/no, no scoring or analysis) ---
  "detection": {
    "ai": {
      "llm_api": true,               // imports openai, anthropic, etc.
      "llm_provider": "openai",      // detected provider name
      "vector_database": true,        // pinecone, weaviate, chromadb, etc.
      "vector_db_name": "chromadb",
      "rag_pipeline": true,           // detected retrieval-augmented generation pattern
      "mcp_servers": false,           // Model Context Protocol server detected
      "fine_tuned_models": false,     // fine-tuning scripts or model artifacts
      "training_pipeline": false,     // ML training code detected
      "proprietary_dataset": true     // ETL, data processing, custom dataset loading
    },
    "infrastructure": {
      "grade": "A",
      "iac_detected": true,
      "iac_types": ["terraform", "docker"],
      "ci_cd_detected": true,
      "ci_cd_provider": "github_actions",
      "monitoring_detected": true,
      "monitoring_tools": ["datadog"]
    }
  },

  // --- Red Flags (instant deal-killers, prominently surfaced) ---
  "red_flags": {
    "count": 0,
    "flags": []
    // Possible flags:
    // { "flag": "secrets_detected", "detail": "3 hardcoded secrets found", "severity": "critical" }
    // { "flag": "critical_cve", "detail": "2 critical CVEs in dependencies", "severity": "critical" }
    // { "flag": "no_tests", "detail": "0% est. test coverage — no test files found", "severity": "high" }
    // { "flag": "no_ci_cd", "detail": "No CI/CD pipeline detected", "severity": "medium" }
    // { "flag": "stale_repo", "detail": "Last commit 8 months ago", "severity": "high" }
    // { "flag": "no_readme", "detail": "No README found in any repository", "severity": "medium" }
    // { "flag": "unmaintained_deps", "detail": "52% of dependencies unmaintained (2yr+)", "severity": "high" }
  },

  // --- Summary (overall grade + category grades) ---
  "summary": {
    "scored_categories": ["maintainability", "security", "handoff_readiness", "dependency_health", "development_activity", "sre_infrastructure"],
    "overall_grade": "B+",
    "top_risks": [
      { "category": "handoff_readiness", "issue": "Low est. test coverage (42%)", "severity": "medium" },
      { "category": "security", "issue": "1 medium CVE in lodash", "severity": "medium" }
    ],
    "top_strengths": [
      { "category": "security", "detail": "No hardcoded secrets detected" },
      { "category": "maintainability", "detail": "Low code duplication (4.2%)" }
    ]
  },

  // --- Pricing Tier (auto-determined by scanner) ---
  "pricing_tier": {
    "tier": "standard",              // starter | standard | professional | enterprise
    "reason": "42,600 LOC"           // explains why this tier was selected
  },

  // --- Warnings (partial analysis, skipped checks, etc.) ---
  "warnings": [],
  // Possible warnings:
  // { "code": "partial_analysis", "message": "PHP analyzer skipped: tree-sitter grammar download failed", "repo": "backend" }
  // { "code": "cve_lookup_skipped", "message": "CVE lookup skipped for npm: OSV API unreachable", "ecosystem": "npm" }

  // --- Integrity (proves this JSON came from official VettCode scanner + platform co-signature) ---
  "integrity": {
    "scan_checksum": "sha256-of-full-scan-json-excluding-integrity-block",
    "scanner_public_key_id": "vettcode-scanner-key-2026-03",
    "scanner_signature": "ed25519-signature-of-checksum",
    "cosign_nonce": "server-issued-nonce-hex",             // null if --offline
    "platform_cosignature": "ed25519-cosignature",         // null if --offline
    "platform_public_key_id": "vettcode-platform-key-2026-03",  // null if --offline
    "cosigned": true                                        // false if --offline
  }
}
```

### Version Compatibility Policy (Applies to All Data Contracts)

The scan JSON (9a) is the critical interface between the scanner (Go binary distributed to sellers) and the platform (Python backend). Old scanner versions will be in the wild for months after new versions ship. The following rules govern schema evolution:

- **Platform accepts current + previous major version.** When scanner v2.0 ships, the platform continues accepting v1.x scan JSONs for at least 6 months. This overlap period is announced in scanner release notes.
- **Minor versions are additive only.** Within a major version (e.g., 1.0 → 1.1 → 1.2), new fields may be added but existing fields are never removed or renamed. The platform treats missing optional fields as null/absent and applies safe defaults (e.g., a new metric added in v1.2 is simply absent in v1.0 JSONs — the report omits that section rather than failing).
- **Breaking changes require a major version bump.** Removing fields, changing field types, or altering semantics (e.g., a score that was 0-100 becoming a letter grade) require incrementing the major version (1.x → 2.x).
- **Scanner version check.** The scanner should check for updates on each run (non-blocking, advisory only) and warn if it is more than 2 minor versions behind. This nudges sellers toward current versions without breaking their workflow.
- **Platform-side validation.** The platform validates uploaded scan JSON against a version-aware schema. Unsupported versions are rejected with a clear error message directing the seller to update their scanner.

### 9b. Signed Report Data Model

The platform generates an internal signed report data model that wraps the scan data and adds a platform-level Ed25519 signature, risk/strength summaries, buyer disclosure, and plain-English category explanations for non-technical buyers. This internal JSON powers the PDF deliverable (see Section 8). Reports use UUID v4 as their sole identifier — no sequential IDs exist (anti-enumeration). A staleness indicator (computed at render time) warns buyers when reports are >30 or >90 days old.

Full schema: see [Platform Backend Section 5.3a](./components/02-platform-backend-design.md).

### 9c. Deep Scan Report Data Model

Deep scan produces a report extending the static scan with LLM-powered analysis across 7 categories: AI moat, architecture, code quality, technical debt, security, infrastructure, and post-acquisition risk. Signed with Ed25519 and delivered to both buyer and seller as PDF. If a static scan exists, the deep report references it; otherwise the deep scan is self-contained.

Full schema: see [Deep Scan Design Section 7.2](./components/03-deep-scan-design.md).

---

## 12. Technical Decisions & Rationale

> For pricing tiers referenced in the `pricing_tier` field, see Section 10 in the [Business Overview](./00a-product-overview-business.md).


| Decision              | Choice                                          | Rationale                                                                    |
| --------------------- | ----------------------------------------------- | ---------------------------------------------------------------------------- |
| Scanner language      | **Go**                                          | Single binary, cross-platform, fast, small binary size                       |
| Scanner plugin system | **On-demand download + Docker fallback**        | Keeps CLI slim; Docker for air-gapped                                        |
| Deep scan LLM        | **Claude API (Anthropic)**                      | Best-in-class code analysis, large context window for full-repo analysis, structured output |
| Deep scan architecture | **Separate component (`vettcode-deep-scan`)**  | LLM orchestration + prompt engineering is distinct from CRUD backend; different team skills, different release cadence |
| Backend language      | **Python (FastAPI)**                            | Rapid development, strong LLM ecosystem, good for report generation          |
| Frontend framework    | **Next.js 14 (App Router)**                     | SSR for marketing pages, React for dashboard, strong ecosystem               |
| Frontend hosting      | **Vercel**                                      | Native Next.js platform, free tier, global CDN, zero config                  |
| Backend + Workers     | **GCP Cloud Run**                               | Scale-to-zero (pay only during scans), same VPC for API + workers            |
| Database              | **GCP Cloud SQL (PostgreSQL)** + Cloud SQL Auth Proxy (sidecar) | Managed Postgres, same network as Cloud Run, JSONB for flexible reports. Auth Proxy handles connection pooling, IAM auth (no DB passwords), and automatic TLS. |
| Queue                 | **GCP Cloud Tasks**                             | Serverless, scale-to-zero, no idle cost; async job dispatch for GitHub scans and deep scans |
| Digital signatures    | **Ed25519**                                     | Fast, small signatures, widely supported, quantum-resistant-ish              |
| Payments              | **Stripe**                                      | Industry standard, supports one-time + subscription                          |
| Auth                  | **Clerk** (social login: Google, GitHub, Apple)  | Don't build auth; social-only login (no email/password) for simplest UX      |
| Object storage        | **GCP Cloud Storage**                           | Same region/VPC as Cloud Run — fast writes after scan, signed URLs for downloads |
| IaC                   | **Terraform**                                   | Industry standard, manages GCP + Vercel config                               |
| CI/CD                 | **GitHub Actions**                              | Already on GitHub, free tier generous                                        |
| Report format         | **PDF**                                         | Industry-standard shareable document for M&A professionals; specific rendering library is a backend component decision (see doc 02) |

### Infrastructure Provider Strategy

| Layer | Provider | Why |
| --- | --- | --- |
| **Frontend** | Vercel | Native Next.js, free tier CDN, serves static assets + SSR pages |
| **Backend API** | GCP Cloud Run | Scale-to-zero, same network as scan workers |
| **Scan Workers** | GCP Cloud Run | Ephemeral containers for GitHub-connected static scans — clone repo, run scanner, produce JSON. Scale-to-zero. |
| **Deep Scan Workers** | GCP Cloud Run | Ephemeral containers for post-LOI LLM-powered analysis — clone repo, run Claude API prompts, auto-destroyed after scan |
| **Database** | GCP Cloud SQL + Auth Proxy | Managed Postgres, same VPC as Cloud Run. Auth Proxy sidecar for connection pooling + IAM auth. |
| **Job Queue** | GCP Cloud Tasks | Serverless async dispatch — triggers scan workers and deep scan workers. No idle cost. |
| **Report Storage** | GCP Cloud Storage | Same region as Cloud Run — fast writes, signed URLs for buyer downloads |
| **GTM Service (Drumbeat)** | GCP Cloud Run | Separate product — see [drumbeat-design.md](./components/drumbeat-design.md) |

Provider rationale and report download flow: see [Infrastructure & SRE Design](./components/05-infrastructure-sre-design.md).

**Estimated monthly infrastructure cost:**

> Detailed line-item breakdown in [Infrastructure & SRE Design, Section 10.1](./components/05-infrastructure-sre-design.md#101-cost-projection).

| Period | Est. Cost | Notes |
| --- | --- | --- |
| Month 1-3 | **$35-120** | Cloud SQL (~$7-10) is main fixed cost; Cloud Run near-zero at low volume; LLM API $10-75 |
| Month 4-6 | **$130-550** | Cloud Run scales with scan volume; LLM API costs grow with deep scans |
| Month 7-12 | **$450-1,520** | Higher scan volume, Deep Scan LLM costs dominate ($150-1,000) |


---

## 13. Security & Privacy Considerations

**Privacy model by scan type:**

| Scan Type | Trust Level | What leaves the seller's machine | What reaches a third party |
| --- | --- | --- | --- |
| **Local CLI scan** (co-signed, default) | **Platform co-signed** | Scan hash + nonce sent to VettCode API for co-signing. **No source code, no file names, no file paths, no metrics.** Scan JSON uploaded to platform after scan. | Nothing beyond VettCode. |
| **Local CLI scan** (`--offline` flag) | Self-reported | Nothing during scan. Scan JSON uploaded to platform after scan. | Nothing beyond VettCode. |
| **GitHub-connected scan** (static) | **GitHub-verified** | Source code cloned to VettCode's ephemeral container but **never persisted** — deleted immediately after scan. | Nothing — code stays within VettCode's GCP infrastructure. |
| **Deep scan** (premium) | **GitHub-verified** | Source code cloned to ephemeral container and **sent to Anthropic Claude API** for analysis. Deleted after. | **Anthropic (Claude API)** — not used for training, not stored after processing. |

> **Important:** Marketing and sales materials must accurately reflect these distinctions. "Privacy-first" applies to the static scan paths. The deep scan involves sending code to a third-party LLM — sellers must explicitly consent to this (see Workflow 4, Consent & notification flow).

- **Report signing** with Ed25519 — tamper-evident, verifiable without platform access
- **Scanner binary signing** — code-signed releases to prevent supply chain attacks
- **Scan result JSON contains NO file names or paths** — only hashes, preventing reverse engineering of code structure. (The CLI terminal output does show real file paths for hotspots and secrets so sellers can fix issues and rescan — this output is ephemeral and local, never uploaded.)
- **Ownership verification** — Two-tier approach to prevent sellers from misrepresenting ownership:
  - **CLI uploads (co-signed):** Seller must provide official company name at upload time. Reports show `verification_level: "platform_cosigned"`. Co-signing prevents scan data forgery. Legal liability deters fraud.
  - **CLI uploads (offline):** Same as above but `verification_level: "self_reported"`. Platform displays trust notice to buyers.
  - **Provider-connected scans (GitHub + GitLab in V1, Bitbucket V2):** Platform verifies seller has admin/maintain access to the repo via the provider's API. Reports show `verification_level: "provider_verified"` and `scan_origin` identifying the specific provider. Strongest assurance for buyers.
  - **Repo deduplication (V2):** Scanner captures HEAD commit SHA per repository for future dedup fingerprinting. V1 stores the data but does not perform cross-user matching. V2 will check for duplicate repos across users and surface warnings.
  - Company name and verification level are embedded in the signed report and visible to buyers.
- **CLI scan integrity — remote co-signing (V1):** The scanner's embedded private key could theoretically be extracted via binary reverse engineering. V1 mitigates this with **remote co-signing**: during each scan, the scanner sends only a hash + nonce to VettCode's API, which verifies and co-signs it. Forging a co-signed scan would require compromising both the scanner key AND the platform's signing infrastructure. The `--offline` flag disables co-signing for airgapped use, but those reports carry `verification_level: "self_reported"` with a buyer-facing trust notice.

### Abuse & Fraud Mitigation

Key abuse vectors and mitigations: open-source code fraud (V1: self-attestation + GitHub admin verification + buyer disclosure; V2: commit SHA dedup + known-OSS fingerprinting), fake account spam (social-only OAuth login raises cost of fake accounts, UUIDs prevent report enumeration), and rate limiting on all sensitive endpoints to prevent scraping and cost attacks. Detailed controls and per-endpoint rate limits: see [Platform Backend Section 5.9](./components/02-platform-backend-design.md).

### Ed25519 Key Management

Two key pairs exist: a **scanner signing key** (embedded in the CLI binary, rotated per major release) and a **platform signing key** (stored in GCP Secret Manager, rotated annually, backed up to a separate GCP project). The platform key is also used for co-signing CLI scans (see co-signing above). Both have defined compromise response procedures. The platform maintains a public key registry with revocation support for signature verification. Detailed key policies: see component docs below.

**Component ownership:**

| Responsibility | Owner | Details |
| --- | --- | --- |
| Key lifecycle (provisioning, storage, rotation automation, backup, IAM, audit logging) | [Infrastructure / SRE (05)](./components/05-infrastructure-sre-design.md) | Terraform Secret Manager modules, IAM bindings, backup to separate GCP project, Cloud Audit Logs |
| Key usage (signing reports, verifying scanner signatures, public key registry) | [Platform Backend (02)](./components/02-platform-backend-design.md) | Application logic that reads keys from Secret Manager and performs sign/verify operations |
| Scanner key embedding (private key embedded in binary at build time) | [Scanner (01)](./components/01-scanner-design.md) | Build process embeds key; key ID selection per release |

- **SOC2 readiness** should be on the roadmap by Month 6 (we're selling trust)
- **GDPR compliance** — minimal PII collection, data retention policies, account deletion endpoint (`DELETE /api/v1/account`), cookie consent banner (essential cookies only + optional analytics)

---

## 14b. Infrastructure Capacity & SRE

Infrastructure scales using serverless GCP services (Cloud Run, Cloud SQL, Cloud Tasks). Monthly cost grows from ~$35-120 in early months to ~$450-1,520 at scale, with LLM deep scan API costs as the dominant driver. Authoritative line-item breakdown: [Infrastructure & SRE Design, Section 10.1](./components/05-infrastructure-sre-design.md#101-cost-projection). Detailed capacity planning, resource sizing, and SRE metrics: see [Infrastructure & SRE Design, Section 10](./components/05-infrastructure-sre-design.md).

---

## 15. Open Questions for Discussion

> Remaining open questions are tracked in the [Business Overview](./00a-product-overview-business.md#15-open-questions-for-discussion).

---

## 17. Standard Design Document Template

Each component document should include:

1. **Component Overview** — Purpose, scope, boundaries
2. **Functional Requirements** — User stories, acceptance criteria
3. **Technical Requirements** — Performance, scale, compatibility
4. **Architecture** — System design, component diagram, data flow
5. **Solution Design** — Detailed technical approach
6. **Tech Stack** — Languages, frameworks, libraries with rationale
7. **Inputs & Outputs** — Data contracts, API specs
8. **Diagrams** — Architecture, sequence, data flow (Mermaid format)
9. **Testing Plan** — Unit, integration, E2E strategy
10. **Capacity & Performance** — Load estimates, resource sizing
11. **Milestones & Tickets** — Epics, stories, subtasks with estimates
