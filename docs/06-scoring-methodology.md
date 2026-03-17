# 06 — Scoring Methodology

**Version:** 0.1-draft
**Status:** In Review
**Parent:** [Product Overview (Technical)](../00b-product-overview-technical.md)

---

## 1. Overview

VettCode grades codebases across **6 scored categories** (letter grades A through F), **3 data-only categories** (structured data without grades), and produces **1 overall grade** (weighted average of the 6 scored categories).

This document defines:
- How each metric is collected
- How each category score is calculated (0–100)
- How scores map to letter grades
- How the overall grade is weighted
- Why certain categories are data-only
- Limitations and disclaimers

**Design principles:**
- Grades are based on **absolute thresholds from industry best practices**, not comparative benchmarks against other VettCode users
- Aggregation across multi-repo scans **never hides problems** — grades are computed from aggregated metrics, not averaged from per-repo grades
- **No user-defined exclusions** — all scans use identical rules; sellers cannot game scores
- All scoring logic is deterministic and reproducible — same code in, same grade out

---

## 2. Grade Scale

All scored categories use the same 0–100 → letter grade mapping:

| Score | Grade | | Score | Grade |
| --- | --- | --- | --- | --- |
| 93–100 | A | | 73–76 | C |
| 90–92 | A- | | 70–72 | C- |
| 87–89 | B+ | | 67–69 | D+ |
| 83–86 | B | | 63–66 | D |
| 80–82 | B- | | 60–62 | D- |
| 77–79 | C+ | | < 60 | F |

**No A+ grade exists.** The highest possible grade is A (93–100). This is intentional — the scale follows standard academic conventions with a hard ceiling. Any component producing an "A+" grade has a bug.

The **overall grade** uses the same mapping applied to the weighted average of all 6 category scores.

---

## 3. Scored Categories

### 3.1 Code Maintainability

**What it measures:** How easy the codebase is to understand, modify, and extend.

**Metrics collected by scanner:**
- Cyclomatic complexity (average, p90, max per file) — *p90 and max are supplementary data shown in report details; only average is used in scoring*
- Code duplication percentage (token-based detection)
- Nesting depth (average, max) — *max is supplementary data; only average is used in scoring*
- File size distribution (% of files over 500 LOC)

**Score calculation (0–100):**

| Sub-metric | Weight | Formula | Thresholds |
| --- | --- | --- | --- |
| Complexity | 40% | `max(0, 100 - (avg_complexity - 5) * 4)` | avg 5 → 100, avg 10 → 80, avg 15 → 60 |
| Duplication | 30% | `max(0, 100 - duplication_pct * 5)` | 0% → 100, 5% → 75, 10% → 50, 20% → 0 |
| Nesting | 15% | `max(0, 100 - (avg_nesting - 1.5) * 20)` | avg 1.5 → 100, 2.5 → 80, 4.0 → 50 |
| File size | 15% | `max(0, 100 - pct_files_over_500loc * 2)` | 0% large → 100, 25% → 50, 50% → 0 |

**Why these thresholds:** Cyclomatic complexity of 5 is widely accepted as "simple" (McCabe, 1976). Duplication under 5% is considered healthy across most static analysis tools. Nesting depth of 1.5 average reflects flat, readable code. The 500-LOC file threshold aligns with common linter defaults.

---

### 3.2 Security Posture

**What it measures:** How exposed the codebase is to known vulnerabilities and credential leaks.

**Metrics collected by scanner:**
- Hardcoded secrets count (regex + entropy detection)
- Known CVEs by severity (critical, high, medium, low)
- License compatibility issues (SPDX detection)

**Score calculation (0–100):** *(canonical — all other docs must match these values)*

| Sub-metric | Weight | Formula | Thresholds |
| --- | --- | --- | --- |
| Secrets | 35% | `100 if count == 0 else 0` | Binary: any secret → 0 |
| CVEs | 45% | `max(0, 100 - critical*30 - high*15 - medium*5 - low*1)` | 1 critical → 70, 2 critical → 40 |
| License issues | 20% | `max(0, 100 - issues * 25)` | 0 → 100, 2 → 50, 4 → 0 |

**Why these thresholds:** Secrets scoring is binary because even one leaked credential is a critical finding. CVE severity weights (critical=30, high=15, medium=5, low=1) follow CVSS severity classification conventions. CVEs at 45% weight captures the actual security consequence of outdated dependencies — unmaintained dependency percentage is scored separately under Dependency Health to avoid double-counting. License issues are weighted at 20% because license risk in M&A carries legal exposure for buyers.

---

### 3.3 Handoff Readiness

**What it measures:** How prepared the codebase is for a new owner to take over.

**Metrics collected by scanner:**
- Estimated test coverage % (file-ratio heuristic: test files / total files, LOC-weighted)
- Documentation density (high / medium / low — based on README, inline comments, doc files)
- Environment variable count (from .env.example, .env.template, or config schema)

**Score calculation (0–100):**

| Sub-metric | Weight | Formula | Thresholds |
| --- | --- | --- | --- |
| Est. test coverage | 50% | `min(100, coverage_pct * 1.25)` | 80% → 100, 50% → 62.5, 0% → 0 |
| Documentation | 25% | `high → 90, medium → 60, low → 30` | Categorical mapping |
| Env complexity | 25% | `max(0, 100 - max(0, env_count - 5) * 3)` | ≤5 → 100, 15 → 70, 25 → 40, 38 → 0 |

**Why these thresholds:** Test coverage of 80% is widely considered "strong" for production applications. The file-ratio heuristic is always labeled "Est. Test Coverage" in the UI — it is not execution coverage. Documentation density uses a categorical scale because measuring doc quality precisely requires LLM analysis (reserved for Deep Scan). Environment variable count reflects configuration complexity a new owner must manage.

**Env complexity note:** The multiplier was adjusted from 5 to 3 (floor at 38 env vars instead of 25) because mature SaaS products commonly have 25–30 environment variables. The original threshold penalized well-configured production apps too aggressively. Monitor post-launch and recalibrate if needed.

**Note:** Contributor count is intentionally NOT included in scoring. A solo founder should not be penalized for building alone. Contributor count is shown as raw data in Development Activity.

---

### 3.4 Dependency Health

**What it measures:** How current and well-maintained the project's dependencies are.

**Metrics collected by scanner:**
- Median dependency age (months)
- Unmaintained dependency percentage (no update in 2+ years)
- Total dependency count — *supplementary data shown in report and Codebase Profile; not used in scoring formula*

**Score calculation (0–100):**

| Sub-metric | Weight | Formula | Thresholds |
| --- | --- | --- | --- |
| Median age | 50% | `max(0, 100 - max(0, median_months - 6) * 2.5)` | ≤6mo → 100, 18mo → 70, 30mo → 40, 46mo → 0 |
| Unmaintained % | 50% | `max(0, 100 - unmaintained_pct * 4)` | 0% → 100, 10% → 60, 25% → 0 |

**Why these thresholds:** A 6-month median dependency age means most packages are on recent releases — healthy for a maintained project. Dependencies older than 2 years without updates are widely considered unmaintained. The 25% unmaintained threshold as a floor reflects a point where dependency upgrade effort becomes a material acquisition cost.

**Note:** CVE counts from dependencies are already captured in the Security category. Dependency Health focuses on currency and maintenance status to avoid double-counting.

---

### 3.5 Development Activity

**What it measures:** How actively and consistently the codebase is being developed.

**Metrics collected by scanner:**
- Days since last commit
- Commit velocity (average commits per month, last 12 months)
- Active development months (months with >0 commits in last 12)

**Score calculation (0–100):**

| Sub-metric | Weight | Formula | Thresholds |
| --- | --- | --- | --- |
| Recency | 40% | `max(0, 100 - days_since_last_commit * 0.55)` | 0 days → 100, 30 days → 83, 90 days → 50, 180 days → 0 |
| Velocity | 30% | `min(100, 22 * sqrt(total_commits / min(12, repo_age_months)))` | 5/mo → 49, 7/mo → 58, 10/mo → 70, 20/mo → 98 |
| Consistency | 30% | `(active_months / min(12, repo_age_months)) * 100` | All months active → 100, half → 50 |

**Why these thresholds:** A commit within the last 30 days signals active maintenance. Velocity uses a **diminishing-returns curve** (`22 * sqrt`) so that mature projects with 5–10 commits/month score reasonably (49–70), while still rewarding higher velocity without requiring an unrealistic 20 commits/month for a perfect score. Consistency matters more than bursts — a codebase with 6 idle months raises transfer risk.

**Repo age scaling:** Both velocity and consistency are computed over `min(12, repo_age_months)` rather than a fixed 12-month window. A 2-month-old project with 2 active months gets 100% consistency, not 17% (2/12). This prevents new projects from being penalized for not having existed long enough. Repo age is determined from the first commit date.

**Note:** Contributor count is shown as raw data but NOT scored. A solo founder's codebase can be perfectly healthy.

---

### 3.6 SRE & Infrastructure

**What it measures:** How mature the project's operational practices are.

**Metrics collected by scanner:**
- Infrastructure as Code detected (Docker, Terraform, K8s, etc.)
- CI/CD pipeline detected (GitHub Actions, GitLab CI, etc.)
- Monitoring detected (Datadog, Prometheus, Sentry, etc.)

**Score calculation (0–100):**

| Sub-metric | Weight | Formula | Thresholds |
| --- | --- | --- | --- |
| IaC | 35% | `100 if detected else 0` | Binary |
| CI/CD | 40% | `100 if detected else 0` | Binary |
| Monitoring | 25% | `100 if detected else 0` | Binary |

**Why these weights:** CI/CD is weighted highest because it directly affects deployment safety and velocity post-acquisition. IaC is next because it determines infrastructure reproducibility. Monitoring is weighted lowest because it's the easiest to add post-acquisition.

**Why binary scoring:** These are foundational operational practices. Having Terraform is fundamentally different from not having Terraform — there's no meaningful gradient. The presence check is based on file/config detection (Dockerfile, .github/workflows, terraform files, monitoring SDK imports).

---

## 4. Overall Grade

The overall grade is a weighted average of all 6 scored category scores, mapped to a letter grade using the same scale (Section 2).

| Category | Weight | Rationale |
| --- | --- | --- |
| Security Posture | 25% | Most critical for M&A — vulnerabilities are direct liability |
| Code Maintainability | 20% | Determines ongoing development cost post-acquisition |
| Handoff Readiness | 20% | Directly affects transition timeline and risk |
| Development Activity | 15% | Signals whether the product is alive and maintained |
| Dependency Health | 10% | Reflects upgrade burden and supply chain risk |
| SRE & Infrastructure | 10% | Operational maturity — easiest category to improve post-acquisition |

**Formula:**

```
overall_score = (
    security_score * 0.25 +
    maintainability_score * 0.20 +
    handoff_score * 0.20 +
    activity_score * 0.15 +
    dependency_score * 0.10 +
    infra_score * 0.10
)
overall_grade = grade_from_score(overall_score)
```

**Why these weights:** Security is weighted highest because vulnerabilities create direct legal and financial liability for acquirers. Maintainability and Handoff Readiness are equal because they represent the two biggest post-acquisition cost drivers (development velocity and knowledge transfer). Development Activity is meaningful but less actionable — a buyer can't change the past. Dependency Health and SRE are weighted lowest because they're the most improvable post-acquisition.

---

## 5. Data-Only Categories (Not Scored)

These categories provide structured data without letter grades. They contain valuable information for due diligence but cannot be meaningfully scored because "good" vs "bad" depends entirely on the product's context.

### 5.1 AI Detection

**What it shows:** Whether the codebase uses AI/ML capabilities (LLM APIs, vector databases, RAG pipelines, MCP servers, fine-tuned models, training pipelines, proprietary datasets).

**Why not scored:** The presence of AI features is context-dependent. A buyer acquiring an AI SaaS product expects LLM integration. A buyer acquiring a payroll tool would view it differently. AI moat analysis (defensibility, integration depth) is reserved for Deep Scan where LLM-powered analysis can assess quality, not just presence.

### 5.2 Tech Stack

**What it shows:** Frameworks, runtime versions, databases, and external services detected.

**Why not scored:** Technology choices are not objectively better or worse. Next.js is not inherently superior to Ruby on Rails. What matters is whether the stack fits the buyer's team and strategy — a judgment only the buyer can make.

### 5.3 Codebase Profile

**What it shows:** Total LOC, file count, repository count, and language breakdown.

**Why not scored:** Codebase size is contextual. 42,000 LOC could be lean for an enterprise platform or bloated for a landing page builder. Profile data helps buyers calibrate expectations but doesn't indicate quality.

---

## 6. Red Flags

Red flags are critical findings surfaced prominently at the top of every report, independent of category grades. A codebase can have an A- in security but still trigger a red flag for a single critical CVE.

| Flag | Trigger | Severity |
| --- | --- | --- |
| Secrets Detected | Any hardcoded secrets (count > 0) | Critical |
| Critical/High CVEs | Any critical or high severity CVEs | Critical |
| No Tests Found | 0% estimated test coverage | High |
| Stale Repository | Last commit > 6 months ago | High |
| Unmaintained Dependencies | ≥50% of dependencies unmaintained | High |
| No CI/CD Detected | CI/CD pipeline not detected | Medium |
| No README | No README file found in any repo root | Medium |
| No Git History | No `.git` directory found (Development Activity is N/A) | High |

Red flags use **OR logic** across repos in multi-repo scans — one repo triggering a flag means the flag appears on the report.

---

## 7. Multi-Repo Aggregation

When a scan covers multiple repositories, metrics are aggregated before scoring:

| Metric type | Rule |
| --- | --- |
| Counts (LOC, CVEs, secrets, deps) | Sum across repos |
| Percentages (duplication, coverage, unmaintained) | LOC-weighted average |
| Grades | Computed from aggregated metrics (NOT averaged from per-repo grades) |
| Complexity (avg) | LOC-weighted average |
| Complexity (p90, max) | Global worst |
| Last commit | Most recent across repos |
| Commit velocity | Sum across repos |
| Active months | Union across repos |
| Binary flags (IaC, CI/CD, monitoring) | OR logic (true if any repo) |
| Red flags | OR logic (one repo triggers for all) |

**Why not average per-repo grades:** Averaging masks problems. A repo with F security and a repo with A security would average to C — hiding the critical vulnerability in the F repo.

---

## 7a. Missing Data — N/A Category Handling

When the scanner cannot compute a category score due to missing input data, that category is marked **N/A** rather than scored. N/A categories are **excluded from the overall grade** and their weights are redistributed proportionally across the remaining scored categories.

**Design principle:** A Go CLI tool with zero npm/pip dependencies should not be penalized for having no dependency health data. But missing data that is itself suspicious should be surfaced as a red flag so buyers are not misled by a grade computed from fewer inputs.

### When each category becomes N/A

| Category | N/A Condition | Red Flag? | Rationale |
| --- | --- | --- | --- |
| Development Activity | No `.git` directory present | Yes — **"No Git History"** (High) | Stripped git history is suspicious in an M&A context. Buyers should know. |
| Dependency Health | Zero dependencies detected (no package manager files) | No | Legitimate for single-file tools, scripts, or self-contained binaries. |
| Code Maintainability | No files in supported languages (all Tier 2 / unsupported) | No — but report prominently notes "No supported languages analyzed" | The report is essentially a metadata-only scan. Buyer should understand limited coverage. |
| Security Posture | N/A not possible — secrets detection and license scanning work on all file types regardless of language support. CVEs require dependency files; if none exist, CVE sub-metric scores 100 (no known vulnerabilities). | — | Security always produces a score. |
| Handoff Readiness | N/A not possible — README detection, env var scanning, and test file ratio work on all repos. | — | Handoff always produces a score. |
| SRE & Infrastructure | N/A not possible — file detection (Dockerfile, CI configs, monitoring) works on all repos. | — | SRE always produces a score. |

### Overall grade with N/A categories

When one or more categories are N/A, the overall grade is computed from the remaining categories with **renormalized weights** (weights scaled proportionally so they sum to 100%).

**Example:** If Development Activity (15%) is N/A:

```
remaining_weights = {
    security: 0.25, maintainability: 0.20, handoff: 0.20,
    dependency: 0.10, infra: 0.10
}
# Sum = 0.85 → renormalize by dividing each by 0.85
renormalized = {
    security: 0.294, maintainability: 0.235, handoff: 0.235,
    dependency: 0.118, infra: 0.118
}
overall_score = sum(category_score * renormalized_weight for each scored category)
```

**Example:** If both Development Activity (15%) and Dependency Health (10%) are N/A:

```
remaining_weights = {
    security: 0.25, maintainability: 0.20, handoff: 0.20, infra: 0.10
}
# Sum = 0.75 → renormalize by dividing each by 0.75
```

### Display requirements

- Reports must show which categories are N/A and why (e.g., "Development Activity: N/A — no git history detected")
- The overall grade label must indicate reduced coverage: **"Overall Grade (4 of 6 categories)"**
- N/A categories appear in the category list as greyed-out cards with an explanation, not hidden
- Any N/A-triggered red flags (e.g., "No Git History") appear in the red flags section as normal

### Scanner implementation

When data is missing, the scanner sets the category grade to `null` in the JSON output and includes an `na_reason` field:

```jsonc
"activity": {
    "grade": null,
    "na_reason": "no_git_directory",
    // raw fields omitted or set to null
}
```

The backend and frontend must handle `grade: null` for any scored category.

---

## 8. Limitations & Disclaimers

Displayed on every report:

1. **No comparative benchmarks (V1).** Grades are based on absolute thresholds from industry best practices, not percentile ranking against other codebases. "B+" means the code meets specific quality thresholds, not that it's better than X% of other projects. Comparative benchmarking is planned for V2 when sufficient anonymized data is available.

2. **Estimated test coverage is a heuristic.** The scanner uses a file-ratio approach (test files / total files, LOC-weighted) — not test execution. Actual coverage may differ. Always labeled "Est. Test Coverage" in reports.

3. **AI detection is presence-based, not quality-based.** The static scanner detects whether AI capabilities exist (LLM API calls, vector DB imports) but cannot assess their quality, defensibility, or business value. Deep Scan provides this analysis.

4. **Binary infrastructure checks.** SRE & Infrastructure scoring detects the presence of tools (Docker, CI/CD, monitoring) but does not evaluate their configuration quality. A misconfigured Terraform setup scores the same as a well-architected one.

5. **Snapshot in time.** A VettCode report reflects the codebase at the moment of scanning. Code quality can change rapidly. Report freshness indicators help buyers assess relevance (< 30 days: Recent, 30–90 days: Aging, > 90 days: Stale).

6. **Language support.** V1 supports JavaScript/TypeScript, Python, Go, PHP, Ruby, and Java. Repositories in unsupported languages are listed but not analyzed. Grades reflect only the analyzed portion of the codebase.

---

## 9. Cross-Component Implementation Notes

This document is the **single source of truth** for all static scan scoring logic (formulas, weights, thresholds, grade scale). Deep scan grades are produced by LLM analysis (see [03-deep-scan-design.md](./03-deep-scan-design.md)) and are not formula-based — they are outside the scope of this document. Other component docs must reference this document rather than duplicating formulas.

**Anti-duplication rule:** Docs 01 and 02 may include a quick-reference summary of sub-metric weights (e.g., "Security: Secrets 35%, CVEs 45%…") for readability, but must NOT duplicate formulas or threshold values. If a weight or formula changes, it changes here first — summary tables in other docs are updated to match. Any conflict between this doc and another doc is a bug in the other doc.

**Where scoring is implemented:**

| Component | Responsibility | Reference |
| --- | --- | --- |
| **Scanner (01)** | Computes scores and grades locally during scan | Scanner Section 4.4 references this doc |
| **Backend (02)** | Re-computes scores server-side for report generation | Backend Section 4.5 and 5.1 reference this doc |
| **Frontend (04)** | Displays 6 scored category grades + 3 data-only categories + overall grade | Frontend FR-07 |
| **Technical Overview (00b)** | Summary of scoring approach; defers to this doc for details | 00b Section 7 |

---

## 10. Implementation Tickets

Doc 06 is a methodology reference — it defines *what* to compute, not a deployable service. All scoring logic is implemented in the scanner (client-side) and backend (server-side, authoritative). This section maps each scoring concern to its implementing ticket.

**No standalone milestones.** Doc 06 has no independent implementation schedule. The scoring work is embedded in scanner Epic 5 and backend Epic 3, executed within their respective milestone timelines.

### Implementing Ticket Cross-Reference

| Scoring concern | Scanner (01) | Backend (02) | Frontend (04) |
| --- | --- | --- | --- |
| Per-metric score functions (all 6 categories) | SC-040 (3h) | T3.1 (1 day) | — |
| Grade conversion (score → letter grade) | SC-041 (2h) | T3.1 (1 day) | FE-006 (`grade-badge`) |
| Overall grade (weighted average) | SC-041 (2h) | T3.1 (1 day) | FE-021 (overall grade badge) |
| Red flag evaluation | SC-042 (2h) | T3.1 (1 day) | FE-020 (red flags section) |
| Multi-repo aggregation | SC-043 (4h) | T3.1 (1 day) | — |
| Report assembly (risks, strengths, explanations) | — | T3.3 (1 day) | FE-018–025 (report viewer) |
| Scored category display (6 cards) | — | — | FE-021 (6h) |
| Data-only category display (3 sections) | — | — | FE-021 (included) |
| Scorer unit tests | SC-062 (2h) | T3.1 AC | — |

**Consistency rule:** Scanner scores are advisory — the backend re-computes all scores server-side from raw metrics. Both must produce identical results for the same inputs. SC-062 and T3.1 unit tests use shared fixture data (SC-060) to verify this.
