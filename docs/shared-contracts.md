# VettCode V1 — Shared Contracts

**Version:** 1.0
**Status:** Reviewed
**Purpose:** Single source of truth for cross-component types, naming conventions, and data flow. Carry this document into every component conversation.

---

## 1. Naming Conventions

| Convention | Rule | Example |
| --- | --- | --- |
| JSON fields | `snake_case` | `scan_id`, `total_loc`, `cve_summary` |
| Percentage fields | `*_pct` suffix | `duplication_pct`, `unmaintained_pct` |
| Count fields | `*_count` suffix or bare noun | `hotspot_count`, `secrets_found`, `suppressed_secrets` |
| Boolean fields | `has_*` prefix or bare adjective | `has_readme`, `cosigned` |
| Monetary values | Cents (integer), `*_cents` suffix for deal values | `price_cents: 29900` = $299.00 |
| Timestamps | ISO-8601 UTC | `2026-03-17T10:30:00Z` |
| UUIDs | v4, used in all API paths and URLs | `550e8400-e29b-41d4-a716-446655440000` |
| Display report ID | `VETT-YYYY-XXXXXX` (display only, never in URLs) | `VETT-2026-000042` |
| Deep scan display ID | `VETT-YYYY-XXXXXX-DEEP` | `VETT-2026-000042-DEEP` |
| Key IDs | `vettcode-{component}-key-YYYY-MM` | `vettcode-scanner-key-2026-03` |
| API base URL | `/api/v1` prefix | `https://api.vettcode.com/api/v1` |

---

## 2. Shared Enums

```
Grade:               A | A- | B+ | B | B- | C+ | C | C- | D+ | D | D- | F  (no A+)
Severity:            critical | high | medium | low
Scan Status:         uploaded | queued | cloning | scanning | completed | paid | report_generated | failed
Report Status:       generating | completed | failed
Payment Status:      pending | completed | failed | expired | refunded
Deep Scan Status:    requested | pending_seller_approval | approved | payment_pending |
                     scanning | analyzing | completed | failed | rejected | cancelled
Verification Level:  self_reported | platform_cosigned | provider_verified
Report Type:         static | deep
Pricing Tier:        starter (≤30K LOC, $99) | standard (≤100K LOC, $299) |
                     professional (≤300K LOC, $599) | enterprise (>300K LOC, $999)
Scan Source:         upload | github | gitlab | gitlab_self_hosted
Repo Status:         analyzed | unsupported | error
Doc Density:         high | medium | low
Trend:               increasing | stable | declining
Git Provider:        github | gitlab | gitlab_self_hosted | bitbucket (V2)
Payment Type:        report | deep_scan
Deal Value Response: confirmed | adjusted
```

---

## 3. Scored Categories & Weights

| Category | Weight | Key Metrics |
| --- | --- | --- |
| Security Posture | 25% | Secrets (35%), CVEs (45%), License issues (20%) |
| Code Maintainability | 20% | Complexity (40%), Duplication (30%), Nesting (15%), File size (15%) |
| Handoff Readiness | 20% | Est. test coverage (50%), Doc density (25%), Env complexity (25%) |
| Development Activity | 15% | Recency (40%), Velocity (30%), Consistency (30%) |
| Dependency Health | 10% | Median age (50%), Unmaintained % (50%) |
| SRE & Infrastructure | 10% | IaC (35%), CI/CD (40%), Monitoring (25%) — all binary |

**Data-only categories (not scored):** AI Detection, Tech Stack, Codebase Profile

**N/A handling:** Exclude from overall grade, renormalize remaining weights.

**Canonical source:** [06-scoring-methodology.md](./components/06-scoring-methodology.md) — all other docs reference, never duplicate formulas.

---

## 4. Cross-Component Data Flow

```
Scanner (Go CLI)                    Platform Backend (FastAPI)           Frontend (Next.js)
─────────────────                   ─────────────────────────           ──────────────────

[1] Local scan
    ├─ Terminal output (ephemeral)
    └─ scan-result.json (9a) ──────► POST /scans/upload ──────────────► Dashboard shows scan
                                       ├─ Validates integrity
                                       ├─ Re-computes scores
[2] Co-sign (optional)                 ├─ Determines pricing tier
    POST /cosign/init ◄────────────►   │
    POST /cosign/complete ◄────────►   │
                                       │
[3] Payment                            │
                                       ├─ POST /payments/checkout ◄─────── Stripe redirect
                                       ├─ Webhook: checkout.completed
                                       │
[4] Report generation                  │
                                       ├─ Assembles report (9b)
                                       ├─ Signs with Ed25519
                                       ├─ Stores JSON + PDF in GCS ────► GET /reports/{uuid}
                                       │                                   Report viewer
[5] Deep Scan                          │
                                       ├─ POST /deep-scan/request ◄────── Buyer requests
                                       ├─ Email → seller consent
                                       ├─ POST /deep-scan/{id}/approve
                                       ├─ Buyer payment
                                       ├─ Cloud Tasks → Deep Scan Engine
                                       │      (clones, LLM analysis,
                                       │       deletes code)
                                       ├─ Receives 9c result
                                       ├─ Signs deep report ─────────────► GET /deep-scan/{id}
                                       │                                    Deep report viewer
[6] Verification (public)             │
                                       └─ GET /reports/{uuid}/verify ───► /verify/{uuid} page
```

---

## 5. Data Contracts Summary

| Contract | Producer | Consumer | Schema Location |
| --- | --- | --- | --- |
| **9a** — Static Scan JSON | Scanner | Backend, Frontend (via Backend) | [00b Section 9a](./00b-product-overview-technical.md#9-data-contracts-draft) |
| **9b** — Signed Static Report | Backend | Frontend, PDF | [02 Section 5.3a](./components/02-platform-backend-design.md) |
| **9c** — Deep Scan Report | Deep Scan Engine | Backend, Frontend, PDF | [03 Section 7.2](./components/03-deep-scan-design.md) |

**Key contract rules:**
- **9a** never contains source code, file paths, or secrets — only hashes and aggregates
- **9b** wraps 9a verbatim + adds risk/strength summaries, explanations, Ed25519 signature
- **9c** extends with 7 LLM analysis categories + deal context + provenance metadata
- Scanner scores are advisory — backend re-computes authoritatively from raw metrics
- Both must produce identical scores for the same inputs (shared test fixtures)

---

## 6. Key API Endpoints (Quick Reference)

| Endpoint | Method | Auth | Purpose |
| --- | --- | --- | --- |
| `/cosign/init` | POST | No | Scanner co-sign session |
| `/cosign/complete` | POST | No | Complete co-signing |
| `/scans/upload` | POST | Yes | Upload scan JSON (max 10 MB) |
| `/scans/{id}/status` | GET | Yes | Poll scan progress |
| `/scans` | GET | Yes | List user's scans (cursor pagination) |
| `/payments/checkout` | POST | Yes | Create Stripe checkout session |
| `/reports/{id}` | GET | Yes | Retrieve full report |
| `/reports/{id}/download` | GET | Yes | Get signed download URL (PDF/JSON) |
| `/reports/{id}/verify` | GET | No | Public signature verification |
| `/reports/lookup` | GET | Yes | Resolve `VETT-YYYY-XXXXXX` → UUID |
| `/reports/{id}/view` | POST | Yes | Record buyer view (idempotent) |
| `/reports/viewed` | GET | Yes | List buyer's viewed reports |
| `/git/{provider}/connect` | POST | Yes | Start OAuth/App install |
| `/git/{provider}/repos` | GET | Yes | List connected repos |
| `/git/{provider}/scan` | POST | Yes | Start connected scan |
| `/deep-scan/request` | POST | Yes | Buyer requests deep scan |
| `/deep-scan/{id}/approve` | POST | Yes | Seller approves |
| `/deep-scan/{id}/reject` | POST | Yes | Seller rejects |
| `/deep-scan/{id}/status` | GET | Yes | Poll deep scan progress |
| `/webhooks/stripe` | POST | Stripe sig | Payment events |
| `/webhooks/auth` | POST | Clerk sig | User sync events |
| `/health` | GET | No | Health check |
| `/scanner/latest-version` | GET | No | Scanner version check |
| `/account` | DELETE | Yes | GDPR account deletion |

**Error response format (all endpoints):**
```json
{ "error": "error_code", "message": "Human-readable description" }
```

---

## 7. Authentication

- **Provider:** Clerk (Google, GitHub, Apple social login)
- **Transport:** `Authorization: Bearer <JWT>` on all protected endpoints
- **Frontend:** Clerk React SDK, auto-refresh, httpOnly cookies
- **Backend:** JWT validation on every protected request
- **Webhooks:** Clerk signature (auth), Stripe signature (payments)
- **OAuth flows:** Signed JWT in `state` parameter (contains `user_id` + CSRF token)
- **Public endpoints:** `/health`, `/scanner/latest-version`, `/cosign/*`, `/reports/{id}/verify`

---

## 8. Integrity & Signatures

| Layer | Algorithm | What's Signed | Key ID Format |
| --- | --- | --- | --- |
| Scanner → JSON | Ed25519 | SHA-256 of full scan JSON (excl. integrity block) | `vettcode-scanner-key-YYYY-MM` |
| Platform co-sign | Ed25519 | Scanner checksum + nonce | `vettcode-platform-key-YYYY-MM` |
| Report signature | Ed25519 | SHA-256 of full report payload | `vettcode-platform-key-YYYY-MM` |

**Canonicalization:** RFC 8785 (JCS) — lexicographic key sort, no whitespace, explicit nulls.

**Version compatibility:** Platform accepts current + previous major version for 6+ months. Minor versions are additive only.

---

## 9. Deep Scan — LLM Analysis Categories

| Category | Model | Critical? | Key Output Fields |
| --- | --- | --- | --- |
| AI Moat | Opus | Yes | `wrapper_score` (0-100), `moat_grade`, `integration_depth`, `defensibility` |
| Security Deep | Opus | Yes | `grade`, `business_logic_vulnerabilities`, `auth_review`, `compliance` (GDPR/SOC2) |
| Architecture | Sonnet | No | `pattern`, `api_surface`, `database`, `external_dependencies` |
| Code Quality | Sonnet | No | `grade`, `anti_patterns`, `error_handling` |
| Technical Debt | Sonnet | No | `total_estimated_effort` (person-weeks), `breakdown` |
| Infrastructure | Sonnet | No | `detected_resources`, `scaling_readiness` |
| Post-Acquisition | Sonnet | No | `migration_effort_total`, `key_person_risk`, `onboarding_estimate`, `first_90_days_roadmap` |

**Failure policy:** Both critical categories must succeed. 5+ of 7 total → partial report. <5 → full failure.

**Deep Scan pricing:** 0.5% of declared deal value, floor $499, cap $4,999. `effective_deal_value = max(buyer_declared, seller_confirmed)`.
