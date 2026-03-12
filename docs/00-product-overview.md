# VettCode V1 -- Product Overview (Index)

**Version:** 0.1-draft
**Status:** In Review

---

## Overview

VettCode is a **"Privacy-First" Technical Due Diligence Platform** for the digital asset M&A market. It provides the equivalent of a home appraisal report — but for software businesses. The product consists of a local CLI scanner (Go) and a cloud platform (Python/FastAPI backend + Next.js frontend).

**One-liner:** "Carfax for code — trust without exposure."

This document was split into two companion documents for easier navigation:

- **[Business Overview](./00a-product-overview-business.md)** — Vision, personas, pricing, GTM strategy, revenue projections, milestones, competitive landscape
- **[Technical Overview](./00b-product-overview-technical.md)** — Architecture, workflows, metrics, data contracts, tech decisions, security, infrastructure capacity

---

## Section Reference Map

Original section numbers are preserved in both documents. Use this map to locate any section referenced by component design docs.

| Original Section | Title | Location |
| --- | --- | --- |
| 1 | Product Vision | [Business](./00a-product-overview-business.md#1-product-vision) |
| 2 | Problems to Solve | [Business](./00a-product-overview-business.md#2-problems-to-solve) |
| 3 | User Personas | [Business](./00a-product-overview-business.md#3-user-personas) |
| 4 | Product Architecture — High Level | [Technical](./00b-product-overview-technical.md#4-product-architecture--high-level) |
| 5 | Core Workflows | [Technical](./00b-product-overview-technical.md#5-core-workflows) |
| 6 | Component Summary | [Technical](./00b-product-overview-technical.md#6-component-summary) |
| 7 | Scanner Metrics — What We Measure | [Technical](./00b-product-overview-technical.md#7-scanner-metrics--what-we-measure) |
| 8 | Report Structure (Static Scan) | [Technical](./00b-product-overview-technical.md#8-report-structure-static-scan) |
| 9 | Data Contracts (9a, 9b, 9c) | [Technical](./00b-product-overview-technical.md#9-data-contracts-draft) |
| 10 | Pricing Strategy | [Business](./00a-product-overview-business.md#10-pricing-strategy-draft) |
| 11 | Go-to-Market Strategy | [Business](./00a-product-overview-business.md#11-go-to-market-strategy) |
| 12 | Technical Decisions & Rationale | [Technical](./00b-product-overview-technical.md#12-technical-decisions--rationale) |
| 13 | Security & Privacy Considerations | [Technical](./00b-product-overview-technical.md#13-security--privacy-considerations) |
| 14a | DAU & Revenue Projections | [Business](./00a-product-overview-business.md#14a-dau--revenue-projections-v1--first-12-months) |
| 14b | Infrastructure Capacity & SRE | [Technical](./00b-product-overview-technical.md#14b-infrastructure-capacity--sre) |
| 15 | Open Questions for Discussion | [Business](./00a-product-overview-business.md#15-open-questions-for-discussion) |
| 16 | V2 Roadmap | [Business](./00a-product-overview-business.md#16-v2-roadmap) |
| 17 | Standard Design Document Template | [Technical](./00b-product-overview-technical.md#17-standard-design-document-template) |
| 18 | V1 Milestones | [Business](./00a-product-overview-business.md#18-v1-milestones-ai-accelerated-with-claude-code) |
| Appendix A | Competitive Landscape | [Business](./00a-product-overview-business.md#appendix-a-competitive-landscape) |

---

## Component Design Documents

- [01 — Scanner Design](./components/01-scanner-design.md)
- [02 — Platform Backend Design](./components/02-platform-backend-design.md)
- [03 — Deep Scan Engine Design](./components/03-deep-scan-design.md)
- [04 — Platform Frontend Design](./components/04-platform-frontend-design.md)
- [05 — Infrastructure / SRE Design](./components/05-infrastructure-sre-design.md)
- [06 — Scoring Methodology](./components/06-scoring-methodology.md)
