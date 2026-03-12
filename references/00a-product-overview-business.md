# VettCode V1 -- Product Overview (Business)

**Version:** 0.1-draft
**Status:** In Review
**Companion Document:** [Technical Overview](./00b-product-overview-technical.md)
**Index:** [Product Overview Index](./00-product-overview.md)

> Section numbers are preserved from the original unified document for cross-reference compatibility with component design docs.

---

## Table of Contents

1. [Product Vision](#1-product-vision)
2. [Problems to Solve](#2-problems-to-solve)
3. [User Personas](#3-user-personas)
10. [Pricing Strategy](#10-pricing-strategy-draft)
11. [Go-to-Market Strategy](#11-go-to-market-strategy)
14a. [DAU & Revenue Projections](#14a-dau--revenue-projections-v1--first-12-months)
15. [Open Questions for Discussion](#15-open-questions-for-discussion)
16. [V2 Roadmap](#16-v2-roadmap)
18. [V1 Milestones](#18-v1-milestones-ai-accelerated-with-claude-code)
- [Appendix A: Competitive Landscape](#appendix-a-competitive-landscape)

---

## 1. Product Vision

VettCode is a **"Privacy-First" Technical Due Diligence Platform** for the digital asset M&A market.
It provides the equivalent of a home appraisal report — but for software businesses listed on marketplaces like Acquire.com, Flippa, MicroAcquire, etc.

**One-liner:** "Carfax for code — trust without exposure."

---

## 2. Problems to Solve


| #   | Problem                                                                                                        | Who Feels It | Severity |
| --- | -------------------------------------------------------------------------------------------------------------- | ------------ | -------- |
| 1   | **Trust Gap** — Sellers fear exposing source code pre-close; Buyers fear hidden tech debt                      | Both         | Critical |
| 2   | **Manual Friction** — Professional tech DD is $5K-$50K+ and takes weeks, priced out for sub-$1M deals          | Both         | High     |
| 3   | **"AI Wrapper" Risk** — Buyers can't distinguish a GPT-wrapper from a real product                             | Buyers       | High     |
| 4   | **Information Asymmetry** — Sellers with genuinely good code have no way to credibly signal quality            | Sellers      | Medium   |
| 5   | **Post-Acquisition Surprise** — Hidden infra costs, single-contributor risk, zero tests discovered after close | Buyers       | High     |
| 6   | **No Standard** — Unlike financial DD, there is no standardized technical health assessment format             | Industry     | Medium   |


---

## 3. User Personas

### Seller (Primary)

- Indie hacker or small team selling a SaaS on Acquire.com/Flippa
- Revenue: $5K-$500K ARR (sweet spot for V1)
- Technically capable (can run CLI/Docker)
- Privacy-sensitive — will NOT share source code before LOI
- Wants to maximize sale price by proving code quality

### Buyer (Primary)

- Solo acquirer or small PE/search fund
- Evaluating 5-20 deals simultaneously
- Needs fast signal on technical quality to filter deals
- May not be deeply technical themselves
- Willing to pay for deep DD on serious deals

### Marketplace (Secondary - Future)

- Acquire.com, Flippa, MicroAcquire
- Could integrate VettCode reports as a listing feature
- Potential distribution partner

---

## 10. Pricing Strategy (Draft)

> For the data contracts that implement pricing tier detection, see Section 9a in the [Technical Overview](./00b-product-overview-technical.md).

### Market Context

- Manual technical DD for small companies: **$3,500 - $20,000** (India-based firms at low end)
- Manual technical DD for mid-sized: **$25,000 - $125,000** (UK/US firms)
- Small business DD services (e.g., CapForge): **$2,500 - $12,500** (avg $5,500)
- VettCode's value prop is **replacing $5K-$20K manual DD**, not competing with $30/mo dev tools like SonarQube

### Signed Report Pricing (Primary Revenue — Tiered by Scan Size)

The Standard Report via local scan is the **core product and primary revenue driver**. Most sellers prefer the privacy-first local scan path and will not grant code access for deep scans. Pricing is tiered by total LOC — the simplest metric we can objectively verify at scan time, and a reasonable proxy for deal value (larger codebases generally correlate with higher-value businesses). Repo count does not affect pricing — a 20K LOC app split across 2 repos costs the same as a 20K LOC monorepo.

| Tier                      | Criteria                        | Price  | Typical Deal Size  |
| ------------------------- | ------------------------------- | ------ | ------------------ |
| **Free Scan**             | Unlimited                       | $0     | Any — CLI + terminal output + raw JSON, no signed report |
| **Starter Report**        | Up to 30K LOC                   | $99    | $10K-$75K deals    |
| **Standard Report**       | Up to 100K LOC                  | $299   | $75K-$300K deals   |
| **Professional Report**   | Up to 300K LOC                  | $599   | $300K-$1M deals    |
| **Enterprise Report**     | 300K+ LOC                       | $999   | $1M+ deals         |

### Deep Scan Pricing (Optional Add-on — Percentage of Deal Value)

Deep Scan requires seller consent to grant temporary code access (post-LOI). Expect lower adoption due to privacy friction. Priced as a percentage of deal value since it provides an affordable alternative to manual technical DD ($5K-$20K for human consultants). For deals under ~$100K, the Deep Scan may be the only DD buyers can justify — manual DD often costs more than the deal itself at that scale.

| | |
| --- | --- |
| **Model** | 0.5% of declared deal value |
| **Floor** | $499 |
| **Cap** | $4,999 |
| **When** | Post-LOI only, requires seller approval |

| Deal Value | 0.5% | Actual Price | % of Deal |
| ---------- | ---- | ------------ | --------- |
| $75K       | $375 | $499 (floor) | 0.67%     |
| $150K      | $750 | $750         | 0.50%     |
| $300K      | $1,500 | $1,500     | 0.50%     |
| $600K      | $3,000 | $3,000     | 0.50%     |
| $1.5M      | $7,500 | $4,999 (cap) | 0.33%   |

**Deal value verification:** The buyer declares deal value at request time and attests to its accuracy (checkbox). During the mandatory seller approval step, the seller **confirms or adjusts** the declared value. If the seller provides a higher value, the price is recalculated using the higher of the two — the buyer sees the updated price before payment. The declared (and seller-confirmed) deal value is embedded in the signed deep scan report as a permanent record. When a parent static report exists, the backend performs a soft reasonableness check against codebase size (LOC), flagging gross outliers without blocking the request. See [02-platform-backend-design.md](./components/02-platform-backend-design.md) Section 7.13 and 7.20 for implementation details.

### Other Services

| Service             | Price      | Notes                                    |
| ------------------- | ---------- | ---------------------------------------- |
| **Marketplace API** | Negotiated | Bulk/rev-share for Acquire.com, Flippa   |

### Positioning: What VettCode Replaces (and What It Doesn't)

| | Manual Tech DD ($5K-$20K) | VettCode Static Report ($99-$999) | VettCode Deep Scan ($499-$4,999) |
| --- | --- | --- | --- |
| **Best for** | $500K+ deals | Pre-screening any deal | <$100K deals (only affordable DD), or pre-DD screening for larger deals |
| **What it answers** | "Should I close this deal?" | "Should I pursue this deal?" | "What am I inheriting technically?" |
| **Code access** | Full (buyer reviews code) | None (privacy-first) | Temporary (LLM analysis, no human sees code) |
| **Timeline** | 2-4 weeks | Minutes | < 10 minutes |
| **Limitations** | Expensive, slow | No code-level analysis | Can't assess team, business logic correctness, legal/contractual risk, product-market fit |

**VettCode does not replace manual DD for large deals.** For $1M+ acquisitions, buyers should still engage human advisors for team interviews, business logic review, and legal/contractual analysis. VettCode's Deep Scan covers the *technical* dimension — architecture, code quality, security, tech debt, AI assets, infrastructure — at a fraction of the cost and time. For deals under ~$100K, where manual DD is cost-prohibitive, VettCode's Deep Scan may be the only technical DD buyers perform.

### Pricing Rationale

- **Standard Report is the volume driver**: at $99-$999 it's an impulse buy relative to deal size (0.03-0.67% of typical deal value)
- **Deep Scan is high-value upsell, not the primary revenue driver**: most sellers won't grant code access, but those who do pay $499-$4,999 per scan — significant per-transaction revenue. For <$100K deals, the Deep Scan fills a gap where manual DD is too expensive.
- **Free scan is the funnel**: sellers try VettCode risk-free, see the terminal output, and pay for the signed report with explanations and market context
- **No bundle discount in V1**: The static report is paid by the *seller* and the deep scan is paid by the *buyer* — different payers at different times make bundling impractical. V2 consideration: if the same party (e.g., buyer) pays for both, a bundle discount could be introduced.

### Free-to-Paid Funnel Strategy

**Expected seller behavior:** Sellers will download the free scanner, run it, see a mediocre score, fix issues, and rescan — potentially multiple times before ever paying for a signed report. Many sellers will use VettCode purely as a free dev tool and never upload their JSON. This is by design.

**Why this is a feature, not a revenue leak:**

1. **The Carfax dynamic** — Home sellers fix the leaky roof before the inspection too, but the buyer's mortgage bank still requires the official appraisal from a licensed inspector. A seller saying "my code is clean" is worthless. A signed VettCode report saying it is worth $99-$999. The value isn't the information — it's the **third-party certification**.

2. **Selection bias builds trust** — Only good-scoring codebases get signed reports → VettCode reports carry a quality signal → buyers learn to trust VettCode reports → buyers start *demanding* VettCode reports → sellers *must* get one to be competitive. This is the flywheel.

3. **Free usage = free marketing** — Every seller who runs the scanner (even without paying) learns the brand, tells other sellers, and validates our scoring model. They are future paying customers when they list.

4. **"Fix and rescan" raises the industry bar** — VettCode literally improves the quality of code being sold in M&A marketplaces. This is a narrative investors and marketplace partners will value.

**Why the raw JSON can't replace the paid report:**
A seller cannot hand the raw JSON to a buyer as a substitute. The buyer has no way to verify:
- That the JSON came from VettCode's scanner (not hand-edited to inflate scores)
- That it was generated from the actual codebase being sold (not a different repo)
- That it's current (not generated 6 months ago before major regressions)

The signed report solves all three: a cryptographic signature proves authenticity, the timestamp proves recency, and the report is verifiable on the platform via its unique verify link. The scanner signature in the raw JSON proves the data came from an official scanner, but only the platform can issue the full signed report with risk/strength analysis and buyer-readable formatting.

**Conversion funnel:**
```
Free scanner downloads (awareness)
        |
    100% free usage
        |
    Sellers fix code + rescan (engagement, brand loyalty)
        |
    ~50-60% upload JSON when ready to list and pay for signed report(conversion to platform)
        |
    ~5-10% of report buyers add Deep Scan (upsell)
```

**Key metric to track:** Scanner downloads → JSON uploads → Paid reports. The upload-to-paid conversion should be high (>50%) because sellers only upload when they're ready to sell and want the certification. The bottleneck is downloads-to-uploads, which grows as buyer demand for VettCode reports increases.

---

## 11. Go-to-Market Strategy

### Phase 1: Community-Led Growth (Month 1-3)

- **Publish verification spec** as open documentation to build trust and credibility
- **Content marketing** on r/SaaS, r/SideProject, r/Entrepreneur, Twitter/X, Hacker News, IndieHackers, Acquire.com Facebook group
- **Free scanner** drives awareness; signed reports drive revenue
- **Launch on Product Hunt** with privacy-first messaging

### Phase 2: Marketplace Partnerships (Month 3-6)

- Approach Acquire.com, Flippa for integration partnerships
- **Flippa V&A bundle opportunity:** Flippa's Verification & Assessment service covers financial DD but not technical DD — propose a bundled offering where Flippa V&A + VettCode report gives buyers complete coverage
- "VettCode Verified" badge on listings
- Revenue share or API licensing model

### Phase 3: Enterprise & Platform (Month 6-12)

- Self-serve platform for recurring scans (CI/CD integration)
- PE firm subscriptions for portfolio monitoring
- White-label reports for brokers

### Target Channels


| Channel                                        | Tactic                                      |
| ---------------------------------------------- | ------------------------------------------- |
| Reddit (r/SaaS, r/SideProject, r/Entrepreneur) | Value-first posts about tech DD             |
| Twitter/X                                      | Thread content on tech DD horror stories    |
| Hacker News                                    | Show HN, technical deep dives              |
| IndieHackers                                   | Founder story, milestones, building in public |
| Acquire.com Facebook Group                     | Helpful answers, case studies               |
| Email                                          | Nurture campaigns to platform users         |
| Product Hunt                                   | Launch campaign (one-time)                  |

> **Implementation:** GTM content generation and distribution is handled by the Drumbeat service (separate product). See [drumbeat-design.md](./components/drumbeat-design.md).


---

## 14a. DAU & Revenue Projections (V1 — First 12 Months)

> For infrastructure capacity sized to these projections, see Section 14b in the [Technical Overview](./00b-product-overview-technical.md).

### Market Sizing (Verified March 2026)

**Flippa** (largest marketplace):
- ~3,000 new listings/month (100/day)
- ~5,000 active listings at any time
- 353K active buyers (growing 10K/month)
- 400K weekly active users
- ~12K deals closed/year (~1,000/month)
- SaaS transactions surged 73.5% YoY in 2025

**Acquire.com** (SaaS-focused):
- 500K+ registered buyers
- 2,000-4,000 active vetted listings (estimated)
- ~300-600 new listings/month (estimated)
- Average deal size ~$500K
- 55.7% SaaS, 31.8% ecommerce

**Other platforms** (Empire Flippers, Microns.io, Motion Invest, etc.):
- Estimated ~500-1,000 additional listings/month combined

**Broader market context:**
- 2,698 SaaS M&A transactions in 2025 (record high, ~225/month)
- Cross-border deals increased from 65% to 85% on Flippa

**VettCode's addressable slice** (SaaS/software listings only, ~45% of total):

| Platform | Raw New Listings/mo | SaaS/Software (~45%) |
| --- | --- | --- |
| Flippa | ~3,000 | ~1,350 |
| Acquire.com | ~300-600 | ~200-350 |
| Other platforms | ~500-1,000 | ~300-500 |
| **Total addressable sellers/mo** | | **~1,850-2,200** |

### Projected Usage (Conservative)

Adoption rates reflect reality: we're an unknown startup competing for trust in a market that doesn't yet have a standard for automated tech DD. Growth inflection points are marketplace partnerships (Month 4-6) and brand recognition via content marketing (Month 7+).

| Period | Adoption Rate | Scans/mo | Paid Reports/mo | Deep Scans/mo | DAU | MAU |
| --- | --- | --- | --- | --- | --- | --- |
| Month 1-3 | 1-2% | 20-45 | 10-25 | 1-3 | 5-20 | 50-200 |
| Month 4-6 | 3-8% | 55-175 | 30-100 | 5-15 | 30-100 | 300-1,000 |
| Month 7-12 | 8-15% | 150-330 | 80-200 | 15-40 | 80-250 | 800-2,500 |

**Assumptions:**
- "Adoption rate" = % of addressable SaaS sellers/month who run a VettCode scan
- ~50-60% of scans convert to paid reports (value is clear after seeing terminal output)
- Deep Scan adoption is low (~5-10% of paid report buyers) due to seller privacy friction
- DAU is low because usage is transactional (scan once, view report), not daily-habit
- MAU includes buyers who log in to view/verify reports

### Revenue Projection (Standard Report is Primary Driver)

| Period | Paid Reports/mo | Avg Price | Deep Scan Revenue | **Monthly Revenue** |
| --- | --- | --- | --- | --- |
| Month 1-3 | 10-25 | ~$150 (mostly Starter) | $500-1,500 | **$2,000-$5,250** |
| Month 4-6 | 30-100 | ~$280 (mix of tiers) | $2,500-7,500 | **$11,000-$35,500** |
| Month 7-12 | 80-200 | ~$300 (more Standard/Professional) | $7,500-20,000 | **$31,500-$80,000** |

---

## 15. Open Questions for Discussion

| #   | Question                          | Options                                                     | Status                              |
| --- | --------------------------------- | ----------------------------------------------------------- | ----------------------------------- |
| 6   | Marketplace integration model     | A) API partner / B) Browser extension / C) Embed widget     | Deferred to Drumbeat GTM service    |


---

## 16. V2 Roadmap

Items deferred from V1. Each is documented inline in the relevant V1 section with context; this section consolidates the full V2 backlog.

### LLM-Powered Customer Service Agent

AI support agent that handles inbound calls and emails from customers. Requires platform API access to scan data and report details.

- **Report explanation:** Answers buyer questions about scan results in plain English ("What does my maintainability score mean?", "Why was this flagged as a security risk?", "How should I interpret the handoff readiness grade?")
- **Scan troubleshooting:** Guides sellers through scanner setup, offline mode, Docker usage, and upload issues
- **Common support flows:** Report access, payment/billing questions, pricing tier clarification
- **Escalation:** Routes edge cases to human support when confidence is low or the request involves account/billing changes
- **Cost rationale:** A lean founding team cannot staff a support desk. An LLM agent provides 24/7 coverage at marginal cost while maintaining quality for the majority of queries.

### Other V2 Features

| Feature | Where Documented | Summary |
| --- | --- | --- |
| Bitbucket integration | 02 FR-08 / Section 4.7 | Bitbucket OAuth2 connected scans + deep scans (provider abstraction already in place from V1 GitLab work) |
| Bundle discount pricing | Section 9 (Pricing) | Discount when the same party pays for both static + deep scan |
| Open-source verify CLI (`vettcode-verify`) | 00b Section 6 | Standalone tool for buyers to verify report signatures offline |
| Per-buyer access grants & audit trail | 00b Workflow 1 | Sellers grant/revoke report access per buyer, with audit log |
| Market benchmarking in reports | 00b Section 8 (PDF) | Compare scores against similar-sized SaaS products (requires historical data) |
| Repo deduplication & OSS fingerprinting | 00b Section 6 (Privacy) | Detect duplicate repos across sellers and known open-source projects |
| Scan comparison | 04 FR-04 | "Your score improved from C+ to B+" — show metric diffs between scans for the same company/repos. Sellers can already see current grades in terminal; this adds historical tracking and a comparison UI |
| In-app notification system | 02 Section 7B, 04 FR-12 | Bell icon, notification dropdown, polling, mark-read. V1 uses email-only with reminder emails for time-sensitive flows (deep scan consent). ~2.5 days of effort across frontend and backend |
| SOC2 readiness | 00b Section 6 | Platform compliance certification |
| CLI direct upload (`vettcode upload`) | 01 (Scanner) | One-command upload from CLI: `vettcode upload --company "MyApp"` generates a pre-authenticated upload URL (short-lived token embedded in the CLI session) and opens the browser directly to the payment/review page. Reduces the seller journey from 8 steps (download → install → scan → get JSON → create account → upload JSON → enter company name → pay) to 4 steps (install → scan → upload → pay). Requires: CLI auth flow (OAuth device code grant or one-time browser handshake), backend endpoint for pre-authenticated upload URLs, and browser auto-open with session handoff. |
| Mobile-optimized deep scan approval | 03 (Deep Scan), 04 (Frontend) | Deep scan approval email includes "Approve" and "Reject" action buttons that deep-link to a mobile-optimized approval page with the privacy disclosure pre-loaded. The primary mobile use case is not reading full reports on a phone — it's a seller receiving an approval notification and wanting to approve/reject in one tap. Requires: dedicated mobile-first approval route (e.g., `/deep-scan/{id}/approve`), email template with action buttons, and the privacy disclosure + buyer profile card rendered for small screens. |

---

## 18. V1 Milestones (AI-Accelerated with Claude Code)

> **Timeline anchor:** Week 1 begins March 17, 2026. All week references below are relative to this start date.

| Milestone                    | Scope                                                | Target                      | Bottleneck                         |
| ---------------------------- | ---------------------------------------------------- | --------------------------- | ---------------------------------- |
| **M1: Scanner MVP**          | Core analyzers (JS/TS, Python, Go, PHP, Ruby, Java), CLI, JSON output | Week 1-3                    | Analyzer accuracy tuning           |
| **M2: Platform MVP**         | Upload JSON, generate report, dashboard, Stripe      | Week 1-3 (parallel with M1) | UX decisions, Stripe integration   |
| **M3: Signed Reports**       | Report engine, digital signatures, report viewer     | Week 3 (3-4 days)           | Report template design             |
| **M4: Git Provider Integration** | Connect repos (GitHub + GitLab incl. self-hosted), ephemeral scanning, status tracking | Week 3-5 | OAuth flow, container setup, provider abstraction, GitLab self-hosted |
| **M5: Deep Scan Beta**       | LLM-powered analysis engine (see 03 design doc)     | Week 4-5                    | Prompt engineering, output quality |
| **M6: GTM Launch**           | Marketing site, Product Hunt, content marketing      | Week 4-5 (parallel with M5) | Content/copy decisions             |
| **M7: Marketplace Outreach** | Acquire.com/Flippa partnership discussions           | Week 6+                     | Business development               |


**Critical path planning:**  
- **Static report V1 (scan -> upload -> payment -> signed report): ~5-6 weeks (target: late April 2026)**
- **Full V1 scope including Deep Scan Beta + GTM launch: ~8-10 weeks (target: mid-May 2026)**

Key acceleration factors: Claude Code handles boilerplate, API integrations, and test writing. Real bottlenecks are design decisions (your input), third-party API testing, and analyzer accuracy validation.

---

## Appendix A: Competitive Landscape


| Competitor                          | Overlap                 | VettCode Differentiation                                                       |
| ----------------------------------- | ----------------------- | ------------------------------------------------------------------------------ |
| SonarQube / SonarCloud              | Code quality scanning   | VettCode is M&A-focused, not CI/CD-focused; includes AI moat, business context |
| Snyk                                | Security scanning       | VettCode adds business/DD context; Snyk is dev-tool, not buyer-facing          |
| CodeClimate                         | Maintainability metrics | VettCode adds privacy-first model, signed reports, M&A workflow                |
| Manual DD firms (e.g., DD advisors) | Full-service DD         | VettCode is 10-50x cheaper, instant; complements (not replaces) manual DD for $1M+ deals; for <$100K deals, VettCode may be the only DD buyers can justify |
| Flippa V&A (Verification & Assessment) | Due diligence for digital assets | **Complementary, not competitive.** Flippa V&A covers *financial* DD (revenue verification, traffic validation, P&L analysis) starting at ~$1,500. VettCode covers *technical* DD (code quality, security, tech debt, AI assets). Zero overlap — a buyer could purchase both. Potential partnership: Flippa could bundle or recommend VettCode reports for the technical dimension they don't cover. |
| GitHub Copilot / AI code review     | AI code analysis        | Different use case; VettCode is external validation, not dev tooling           |


**Key moat:** No one combines privacy-first scanning + signed reports + M&A workflow + AI moat detection in one product.

**Partnership opportunity:** Flippa's V&A service validates financials but explicitly does *not* assess code quality or technical architecture. VettCode fills that gap. A bundled offering (Flippa V&A + VettCode report) would give buyers complete DD coverage — financial *and* technical — at a fraction of what manual DD firms charge. This is a natural Phase 2 GTM conversation (see Section 11).
