# VettCode Platform Frontend — Detailed Design Document

**Component:** `vettcode-platform-fe`
**Version:** 0.1-draft
**Status:** In Review
**Parent Document:** [00b-product-overview-technical.md](../00b-product-overview-technical.md)
**Milestone:** M2 — Platform MVP (Week 1-3 starting March 17, 2026, parallel with Scanner)

---

## Table of Contents

1. [Component Overview](#1-component-overview)
2. [Functional Requirements](#2-functional-requirements)
3. [Technical Requirements](#3-technical-requirements)
4. [Architecture](#4-architecture)
5. [Solution Design](#5-solution-design)
6. [Tech Stack](#6-tech-stack)
7. [Page & Route Design](#7-page--route-design)
8. [Diagrams](#8-diagrams)
9. [Testing Plan](#9-testing-plan)
10. [Capacity & Performance](#10-capacity--performance)
11. [Deployment & Operations](#11-deployment--operations)
12. [Milestones & Tickets](#12-milestones--tickets)

---

## 1. Component Overview

### Purpose

The `vettcode-platform-fe` is the web frontend for VettCode. It serves two audiences: **visitors** who see marketing content, and **authenticated users** (sellers + buyers) who interact with the platform. It communicates exclusively with `vettcode-platform-be` via REST API — no direct database access, no backend logic.

### Scope

The frontend is responsible for:

- **Marketing pages** — Landing page, pricing, how it works, trust/security explanation
- **Authentication UI** — Clerk-hosted sign-in/sign-up (social login: Google, GitHub, Apple)
- **Dashboard** — Unified dashboard for all users showing their scans, reports, and deep scans
- **Scan upload flow** — JSON file upload with company name input, pricing tier display, Stripe Checkout redirect
- **Git provider integration flow** — Connect GitHub or GitLab (including self-hosted), select repos, trigger scan, show scan progress
- **Payment flow** — Redirect to Stripe Checkout, handle success/cancel callbacks
- **Report viewer** — Interactive display of signed report data (the core buyer experience)
- **Report verification** — Public page where anyone can verify report authenticity
- **Deep scan flow** — Request, seller approval, payment, status tracking, deep report viewer

### Boundaries

The frontend does NOT:

- Run any scanner logic
- Store or process source code
- Handle payments directly (Stripe Checkout handles this)
- Generate or sign reports (backend responsibility)
- Manage infrastructure or deployments

The frontend DOES interact with:

- `vettcode-platform-be` REST API (all data operations)
- Clerk (authentication UI components + JWT management)
- Stripe (redirect to Checkout, handle return URLs)
- GitHub (redirect to App installation, handle callback)
- GitLab (OAuth2 flow; self-hosted: instance URL input before redirect)

---

## 2. Functional Requirements

### FR-01: Marketing Pages

**User Story:** As a visitor, I can learn what VettCode does, how it works, and what it costs, so I can decide whether to try it.

**Acceptance Criteria:**

- AC-1.1: Landing page with hero section, value proposition, and CTA ("Download Scanner" + "Sign Up")
- AC-1.2: "How It Works" section showing the 3 paths: free scan → paid report → deep scan
- AC-1.3: Pricing page with tier table and deep scan pricing — values sourced from [00a-product-overview-business.md](../00a-product-overview-business.md) Section 10 (single source of truth for all pricing)
- AC-1.4: Trust/security page explaining privacy-first architecture, what data leaves the machine, and Ed25519 verification
- AC-1.5: Marketing pages are server-side rendered for SEO
- AC-1.6: All marketing pages load without authentication
- AC-1.7: **Cookie consent banner** shown on first visit — explains that VettCode uses only essential cookies (Clerk auth session) and optional analytics (Vercel Analytics). Users can accept or reject analytics cookies. Consent choice stored in a first-party cookie (`vettcode_consent`). No third-party tracking cookies are used — consistent with the "privacy-first" brand.
- AC-1.8: **Privacy Policy** and **Terms of Service** pages linked from footer and cookie banner
- AC-1.9: **Guide page** at `/guide` — a public, SSG reference page that serves as the single source of user-facing documentation. Covers both seller and buyer workflows. Structured as a single long-form page with anchor sections and a sticky sidebar table of contents for quick navigation. Sections:
  1. **Getting Started** (`#getting-started`) — What VettCode is, who it's for (sellers and buyers), the 3-path overview (free scan → signed report → deep scan)
  2. **Scanner** (`#scanner`) — Installation (curl, Homebrew, Docker), first scan walkthrough, multi-repo scanning (pass multiple paths, use `--label`), CLI flags reference, offline mode, Docker usage, output format (terminal + JSON), troubleshooting common errors
  3. **Reports** (`#reports`) — What's in a signed report (6 scored categories, 3 data-only, red flags, risk/strength summaries), how to read grades (A-F scale with plain-English meanings), overall grade, how verification works (Ed25519 signatures, public verify link, QR code), report freshness (what Recent/Aging/Stale mean), downloading (PDF/JSON)
  4. **Uploading & Payment** (`#uploading`) — How to upload scan JSON, company name, pricing tiers (by LOC), what happens after payment, Stripe checkout, re-accessing reports from dashboard
  5. **Git Provider Scans** (`#git-providers`) — Connecting GitHub / GitLab (including self-hosted), selecting repos, scan process, verification level (Provider Verified)
  6. **Deep Scans** (`#deep-scans`) — What deep scans add (7 analysis categories), who pays (buyer), how the request/approval flow works, seller privacy disclosure, deal value and pricing, what the seller sees, partial reports
  7. **For Buyers** (`#for-buyers`) — How to access a report (link or ID lookup), what verification levels mean, how to interpret grades and red flags, requesting a deep scan, viewed reports on dashboard
  8. **Scoring Methodology** (`#scoring`) — Brief overview of the 6 scored categories with weights, link to the full scoring methodology (or inline summary), why certain categories are data-only, limitations and disclaimers
  9. **FAQ** (`#faq`) — Common questions consolidating the pricing FAQ and other Q&A: "Why not a subscription?", "What if I have multiple repos?", "What does the free scan include?", "Can the seller see who viewed their report?", "Is my code safe during a deep scan?", "What languages are supported?", "How do I verify a report?", "Can I compare reports?", etc.
  - The guide page is linked from: marketing navigation bar, marketing footer, authenticated app sidebar ("Guide" link), CLI help output (`https://vettcode.com/guide`), and the empty state messages on dashboard
  - Content is written for a non-technical audience first, with technical details available via expandable sections or inline where necessary

### FR-02: Authentication

**User Story:** As a visitor, I can sign up and log in using my Google, GitHub, or Apple account.

**Acceptance Criteria:**

- AC-2.1: Clerk `<SignIn>` and `<SignUp>` components used — no custom auth UI
- AC-2.2: Social login buttons for Google, GitHub, and Apple
- AC-2.3: After sign-in, redirect to `/dashboard`
- AC-2.4: JWT token is automatically attached to all API requests via Clerk's React hooks
- AC-2.5: Unauthenticated users who access protected routes are redirected to sign-in
- AC-2.6: Sign-out clears session and redirects to landing page

### FR-03: Dashboard

**User Story:** As a user, I can see all my scans, reports, and pending actions in one place.

**Acceptance Criteria:**

- AC-3.1: Unified dashboard — no separate seller/buyer views. Everyone sees the same layout with contextual content. **Progressive disclosure:** sections are only rendered when they contain data. Empty sections are hidden entirely (not shown as empty tables). This prevents a pure buyer from seeing empty "My Scans" and "My Reports" sections, and a pure seller from seeing an empty "Viewed Reports" section.
- AC-3.1a: When a section is hidden because it has no data, a contextual cross-persona prompt is shown at the bottom of the dashboard (below all visible sections) to introduce the other workflow:
  - For users with no scans/reports (pure buyer profile): "Selling a business? VettCode helps you prove code quality to buyers. [Learn how →](/guide#scanner)"
  - For users with no viewed reports (pure seller profile): "Evaluating an acquisition? Look up a report by ID or ask a seller to share one with you. [Learn more →](/guide#for-buyers)"
  - These prompts are dismissed permanently with a "Dismiss" link (stored in localStorage). They are not shown if the user has data in all sections.
- AC-3.2: **My Scans** section shows all scans the user has created (CLI uploads + git provider scans), sorted by date, with status badges (`uploaded`, `paid`, `report_generated`, `failed`, etc.). **Hidden if user has zero scans.**
- AC-3.3: Scans in `uploaded` status show a "Pay for Report" button that navigates to the post-upload value page (FR-04, AC-4.5) with the scan's pricing tier pre-loaded
- AC-3.4: Scans in progress (git provider scans: `queued`, `cloning`, `scanning`) show a progress indicator
- AC-3.4a: Scan and report lists use cursor-based pagination — "Load More" button (or infinite scroll) fetches next page via `?cursor=<opaque>&per_page=20`. No page numbers shown.
- AC-3.5: **My Reports** section shows all reports the user owns (as seller), with links to view each report. **Hidden if user has zero reports.**
- AC-3.6: **Viewed Reports** section shows reports the user has accessed by ID (as buyer), for quick re-access. **Hidden if user has zero viewed reports.**
- AC-3.7: **Deep Scan Requests** section shows incoming (as seller) and outgoing (as buyer) deep scan requests with current status. **Hidden if user has zero deep scan requests.**
- AC-3.8: Quick actions are always shown. The set of actions adapts to context:
  - Always: "Look Up Report by ID"
  - If user has scans or reports (seller context): "Upload Scan JSON", "Connect Git Provider"
  - If user has viewed reports (buyer context): "Request Deep Scan"
  - For brand-new users (no data at all): Show all four quick actions plus the guide link
- AC-3.9: Empty state messages guide new users to download the scanner or connect a git provider (GitHub or GitLab), with a link to the guide page ("New to VettCode? Read the guide →")

### FR-04: Scan Upload Flow

**User Story:** As a seller, I can upload my scan JSON and pay for a signed report.

**Acceptance Criteria:**

- AC-4.1: Upload page at `/upload` with file dropzone accepting `.json` files (max 10 MB)
- AC-4.2: Company name input field (required) — label: "Company or Business Name (appears on signed report)"
- AC-4.3: After file selection, frontend shows a preview: repo count, total LOC, language breakdown, red flag count (parsed from JSON client-side for preview only — backend is authoritative)
- AC-4.4: "Upload & Get Pricing" button sends JSON + company name to `POST /api/v1/scans/upload`
- AC-4.5: After successful upload, display a **post-upload value page** with three sections (in order):
  1. **Scan summary strip** — pricing tier badge, total LOC, repo count, red flag count (from upload response)
  2. **"Here's what your buyer will see" — Buyer Preview card** — a live preview of the seller's own report data rendered in the buyer's report layout. This uses the seller's actual scan data (grades, red flags, top risk, top strength) from the upload response, not a generic sample. The preview includes:
     - **Report header** — generated report ID placeholder, seller's company name, verification level badge (based on scan origin), scan date, freshness indicator (green "Recent")
     - **Overall grade** — letter grade badge with a one-sentence qualitative summary (e.g., "Strong technical health suitable for most M&A transactions.")
     - **Category grade grid** — all 6 scored categories with the seller's actual letter grades, displayed in a 3x2 grid matching the buyer's report layout
     - **Red flags** — if any, shown as buyer would see them (red badges with labels); if none, show "No red flags detected" with a green checkmark
     - **Top risk + top strength preview** — one of each, with "buyer will see this flagged with remediation time estimate" / "buyer will see this as a clean [category] signal" annotation text explaining the buyer's experience
     - **Trust signals block** — Ed25519 digital signature, public verification link ("buyer can check independently — no account needed"), QR code for instant mobile verification
     - **"Plus" summary line** — "Plus: plain-English explanations for each category, buyer impact analysis, and downloadable PDF"
     - **Sample report link** — "View full sample report →" linking to `/sample-report`
  3. **Price + CTA** — pricing tier, price, social proof count (when available), and "Pay for Signed Report" button
- AC-4.5a: The buyer preview card is styled to visually resemble the actual report viewer — same card styling, same grade badge colors, same layout proportions (scaled down). The heading reads **"Here's what your buyer will see"** — not a comparison, not a sales pitch, just a preview of the deal asset the seller is about to create. The card has a subtle elevated shadow to distinguish it as a "preview within a page."
- AC-4.5b: The sample report page at `/sample-report` shows a complete report generated from a public open-source project. This page is publicly accessible (no auth required) and serves as both a conversion aid and a marketing asset. It is linked from the buyer preview card and the marketing/pricing pages.
  - **Project selection criteria:** Choose a project that scores in the **B+ range overall** with a mix of strengths and red flags. This demonstrates both sides of the report — grade badges, strength summaries, *and* red flag alerts, risk summaries with buyer impact. A perfect A report doesn't showcase the risk analysis features; an F report discourages sellers. B+ is the sweet spot: strong enough to be aspirational, imperfect enough to show the report's full diagnostic power.
  - **Sample report banner:** The page must display a persistent banner at the top: "This is a sample report generated from [project name] (open source). Your report will reflect your codebase's actual health." The banner is styled distinctly (e.g., blue info background) so it cannot be confused with a real report. The project name links to the project's public repository.
  - **No download/share/verify actions:** The sample report page omits the Download, Share, and Verify buttons — these are only meaningful on real reports. A "Get your own report →" CTA replaces them.
- AC-4.5c: Below the price in the CTA section, a single line of social proof (when available): "Join X sellers who have generated VettCode reports" (count from backend, hidden if < 10).
- AC-4.5d: **Design rationale:** The previous "Free Scan vs. Signed Report" comparison card used defensive framing — telling sellers what they *don't* have. By this point the seller already ran the scan and saw their terminal output; they know the raw data. The buyer preview reframes the moment from "you need to pay" to "look at the deal asset you're about to create," letting the seller experience their report through the buyer's eyes. This creates ownership over the output rather than anxiety about the purchase.
- AC-4.6: "Pay for Signed Report" calls `POST /api/v1/payments/checkout` and redirects to Stripe Checkout
- AC-4.7: Stripe success URL redirects to `/reports/{id}` (UUID) — report is generated automatically on payment success
- AC-4.8: Stripe cancel URL redirects to `/dashboard` — scan remains in `uploaded` status, user can retry payment later
- AC-4.9: Validation errors (invalid JSON, bad signature) shown as clear error messages
- AC-4.10: Client-side JSON preview uses text-only rendering — all values are escaped before display. No `dangerouslySetInnerHTML` or raw HTML injection. This prevents XSS from malicious JSON payloads containing HTML or `__proto__` pollution.

### FR-05: Git Provider Integration Flow

**User Story:** As a seller, I can connect my GitHub, GitLab (including self-hosted), or Bitbucket (V2) account and scan repos without downloading the CLI.

**Acceptance Criteria:**

- AC-5.1: Dashboard shows a "Connect Git Provider" section with provider cards: **GitHub** (active), **GitLab** (active, with "Includes self-hosted" subtitle), **Bitbucket** ("Coming soon" badge)
- AC-5.1a: Each provider card shows connection status (connected/not connected) and the connected account username if linked
- AC-5.2: Clicking an active provider card initiates the provider's connection flow (GitHub App install page for GitHub, OAuth flow for GitLab)
- AC-5.2a: For GitLab, show an additional step before OAuth: radio choice between "GitLab.com" and "Self-hosted GitLab" — if self-hosted, an input field for the instance URL (e.g., `https://gitlab.mycompany.com`) with URL validation
- AC-5.3: After connection, user returns to `/dashboard` with provider connection confirmed
- AC-5.4: "New Scan" page at `/scan/new` shows repos from **all connected providers**, grouped by provider. User selects 1+ repos (can mix providers in a single scan), optionally adds labels, enters company name, and clicks "Start Scan"
- AC-5.5: Frontend calls the provider-agnostic scan endpoint (`POST /api/v1/git/{provider}/scan`, e.g. `/api/v1/git/github/scan` or `/api/v1/git/gitlab/scan`) and redirects to a scan status page
- AC-5.6: Scan status page polls `GET /api/v1/scans/{scan_id}/status` every 5 seconds, showing progress bar and current step
- AC-5.7: On scan completion (`completed` status), show the same post-upload value page as FR-04 (AC-4.5): buyer preview with seller's actual grades, sample report link, pricing tier, and "Pay for Signed Report" button
- AC-5.8: On scan failure, show error message with retry option
- AC-5.9: User can manage all git connections in settings (view connected accounts per provider, disconnect individual connections)

### FR-06: Payment Flow

**User Story:** As a seller, I can pay for my report via Stripe and receive it immediately after.

**Acceptance Criteria:**

- AC-6.1: Payment is handled entirely by Stripe Checkout (no credit card form on our site)
- AC-6.2: After payment success, user is redirected to the report viewer page with their generated report
- AC-6.3: If report generation is still in progress when user returns, show a loading state with "Generating your report..." and poll until ready
- AC-6.4: Payment receipt email is sent by the backend (via Resend), not the frontend

### FR-07: Report Viewer

**User Story:** As a buyer or seller, I can view a signed VettCode report with clear, actionable information.

**Acceptance Criteria:**

- AC-7.1: Report viewer at `/reports/{id}` (UUID) — requires authentication. The human-readable `VETT-YYYY-XXXXXX` is displayed in the UI but never used in URLs (anti-enumeration).
- AC-7.2: Report header shows: report ID, company name, scan date, verification level badge (`Platform Co-Signed`, `Self-Reported`, or `Provider Verified`), scan origin. The specific provider name (GitHub, GitLab) comes from `scan_origin`, not the badge label.
- AC-7.3: Buyer disclosure section shown prominently at top — different content for CLI vs git provider scans (GitHub, GitLab)
- AC-7.3a: For `self_reported` scans, buyer disclosure prominently shows: "⚠ Self-Reported — Not Co-Signed by VettCode. This scan was run offline and has not been independently verified."
- AC-7.3b: For `platform_cosigned` CLI scans, buyer disclosure shows: "ℹ This scan was run by the seller and co-signed by VettCode's platform."
- AC-7.3c: For `provider_verified` scans, buyer disclosure shows: "ℹ This scan was performed by VettCode's cloud infrastructure using the seller's connected [Provider] account. VettCode verified admin/maintain access to these repositories."
- AC-7.4: **Red Flags** section shown first if any exist — prominently styled with warning colors
- AC-7.5: **Overall Grade** displayed prominently in report header as a large letter grade badge. **Category Grades** displayed as cards: Maintainability, Security, Handoff Readiness, Dependency Health, Development Activity, SRE & Infrastructure (each with letter grade badges)
- AC-7.5a: Test coverage metric is always labeled "Est. Test Coverage" (not "Test Coverage") throughout the report viewer — reflecting the file-ratio heuristic, not execution coverage
- AC-7.6: **Data-Only Categories** displayed below scored categories: AI Detection, Tech Stack, Codebase Profile
- AC-7.6a: For multi-repo scans, repos with `"status": "unsupported"` are shown in a separate "Unsupported Repositories" section listing repo name and detected languages, with a note: "These repositories were not analyzed. Language support may be added in future releases."
- AC-7.7: Each category card is expandable to show detailed metrics and plain-English explanations
- AC-7.7a: Each scored category expanded card shows sub-metric weights. Code Maintainability: Complexity (40%), Duplication (30%), Nesting Depth (15%), File Size (15%). Security Posture: Secrets (35%), CVEs (45%), License Issues (20%). Handoff Readiness: Est. Test Coverage (50%), Documentation Density (25%), Environment Variables (25%). Dependency Health: Median Age (50%), Unmaintained % (50%). Development Activity: Recency (40%), Velocity (30%), Consistency (30%). SRE & Infrastructure: IaC (35%), CI/CD (40%), Monitoring (25%). Contributor count is shown under Development Activity as raw data only — it is not included in the Development Activity grade.
- AC-7.8: **Risk Summary** and **Strength Summary** sections with severity indicators
- AC-7.9: **Deep Scan Upsell** section (if static report) with "Request Deep Scan" CTA
- AC-7.10: **Download** button generates a signed GCS URL via `GET /api/v1/reports/{id}/download` (UUID)
- AC-7.10a: Download dropdown offers both PDF and JSON formats. PDF is the primary/default download.
- AC-7.11: **Verification badge** showing signature status — links to `/verify/{id}` (UUID)
- AC-7.11a: Report header shows a freshness indicator based on scan date: green "Recent" (< 30 days), yellow "Aging" (30-90 days), red "Stale" (> 90 days) — helps buyers assess relevance
- AC-7.12: Report accessed by buyer is recorded server-side via `POST /api/v1/reports/{id}/view` and shown in "Viewed Reports" on their dashboard (synced across devices)
- AC-7.13: **Share Report** button (green, primary) in report header opens a popover with two copy options: (a) the human-readable report ID (`VETT-YYYY-XXXXXX`) for buyers to look up via the homepage, and (b) the direct URL for one-click access. Each has a "Copy" button that writes to clipboard and shows "Copied!" confirmation. Helper text explains both sharing methods. The Share button is the most prominent action in the header — sellers need this to complete the core workflow.

> **Wireframe note (M-9):** Share Report popover is the primary CTA in report header (green filled button). See wireframes/index.html Report tab for implementation.

> **Wireframe note (M-7):** Report header should include a freshness indicator showing time since scan (e.g., "Scanned 3 days ago" in green, "Scanned 45 days ago" in amber, "Scanned 90+ days ago" in red with "Consider rescanning" prompt). See wireframes/index.html Report tab for implementation.

> **Wireframe note (M-8):** Report header "Download" button should be a dropdown with two options: "Download PDF" and "Download JSON (raw data)". See wireframes/index.html Report tab for implementation.

### FR-08: Report Lookup

**User Story:** As a buyer, I can enter a report ID shared by a seller to view the report.

**Acceptance Criteria:**

- AC-8.1: Report lookup input on dashboard and at `/verify` — accepts `VETT-YYYY-XXXXXX` format
- AC-8.2: Frontend calls `GET /api/v1/reports/lookup?report_id=VETT-YYYY-XXXXXX` to resolve the human-readable ID to a UUID, then redirects to `/reports/{uuid}`
- AC-8.3: If not authenticated, prompt to sign in first, then redirect to the report after auth
- AC-8.4: Invalid or non-existent report IDs show a clear "Report not found" message

### FR-09: Report Verification (Public)

**User Story:** As anyone (buyer, third party, marketplace), I can verify that a VettCode report is authentic.

**Acceptance Criteria:**

- AC-9.1: Public verification page at `/verify/{id}` (UUID) — no authentication required
- AC-9.2: Page shows: "This report is verified" or "Verification failed" with details
- AC-9.3: Displays: report ID, report type, signed date, scanner version, key ID
- AC-9.4: For signed-in users, includes a "View Full Report" link
- AC-9.5: For anonymous users, shows verification result and prompts sign-in to view full report
- AC-9.6: Also accessible via the `verify_url` embedded in the report JSON
- AC-9.7: Frontend calls `GET /api/v1/reports/{id}/verify` (UUID) to fetch verification status for the public `/verify/{id}` page

### FR-10: Deep Scan Flow

**User Story:** As a buyer, I can request a deep scan; as a seller, I can approve or reject it.

**Acceptance Criteria:**

- AC-10.1: Deep scan can be requested two ways: (a) "Request Deep Scan" button on a static report viewer, which pre-fills the parent report context, or (b) standalone from the dashboard via "Request Deep Scan" quick action, where the buyer selects a seller. Deep scan does NOT require a prior static scan or report.
- AC-10.1a: When initiated from a report, navigates to `/deep-scan/request?report={uuid}` — form pre-loads company name, report ID, and grades as read-only context
- AC-10.1b: When initiated standalone, navigates to `/deep-scan/request` — form requires buyer to enter seller info and company name
- AC-10.2: Request form collects: declared deal value, optional message to seller, and a **required attestation checkbox**: "I confirm that the declared deal value reflects the actual transaction value per the Letter of Intent or equivalent agreement." Submit button is disabled until the checkbox is checked.
- AC-10.2a: When a parent report exists (`?report={uuid}`), the form shows an informational hint below the deal value input: "Typical deal values for codebases of this size: $X - $Y" (range provided by the backend's LOC-based reasonableness check). If the buyer enters a value below the range, a soft confirmation nudge appears: "The value you entered is below the typical range for this codebase size. Continue?" — not a modal or blocker, just a friction-adding nudge with a proceed button.
- AC-10.3: Frontend calculates and shows the estimated price before submission using the deep scan pricing formula from [00a Section 10](../00a-product-overview-business.md). A note is displayed: "Final price may change if the seller provides a different deal value during approval."
- AC-10.4: Seller receives notification via email with request details, including the buyer's declared deal value shown prominently and a **buyer profile summary** (see AC-10.4a)
- AC-10.4a: **Buyer profile card** shown on both the approval email and the dashboard approval UI. Displays: (a) buyer's display name, (b) email domain only — e.g., "@ acquirefund.com" (full email not shown for privacy), (c) "Member since [month year]" from account creation date, (d) platform activity — "Viewed X reports on VettCode" or "First-time buyer" if zero. This helps sellers assess buyer credibility before granting code access.
- AC-10.5: Seller can approve or reject from the dashboard — approval requires confirming GitHub repo access, **confirming the deal value**, and **accepting the privacy disclosure**. The approval card shows the buyer profile card (AC-10.4a) prominently at the top, followed by the buyer's declared deal value with a radio toggle: "Confirm deal value" / "Adjust deal value". If "Adjust" is selected, a number input appears for the seller to enter their deal value. If the `below_expected_range` flag is set, the approval card includes a note: "The declared deal value is below the typical range for a codebase of this size."
- AC-10.5a: **Privacy disclosure on approval screen.** Above the approve/reject buttons, the approval card displays the full privacy disclosure (see [Deep Scan Design (03), Seller Privacy Disclosure](../components/03-deep-scan-design.md#seller-privacy-disclosure)) explaining that source code will be sent to Anthropic's Claude API. The seller must check an explicit consent checkbox: "I understand that my source code will be sent to Anthropic's Claude API for analysis." The approve button is disabled until this checkbox is checked. This field maps to `privacy_disclosure_accepted` in the backend approval request.
> **Wireframe required (see wireframes/index.html, "Deep Scan Approval" tab):** The seller approval screen must include: (1) deal value display with adjustment input (seller can confirm or adjust buyer's declared value), (2) repo selection checkboxes (which repos to include in deep scan), (3) privacy disclosure text (full text from [03-deep-scan-design.md, Seller Privacy Disclosure](./03-deep-scan-design.md#seller-privacy-disclosure)), (4) explicit consent checkbox ("I understand my source code will be sent to Anthropic's Claude API"), (5) approve/reject buttons (approve disabled until checkbox checked).

- AC-10.5b: **Note to buyer.** The approval screen includes an optional "Note to buyer" text field. Sent with both approve and reject actions. On approval, included in the `deep_scan_approved` email. On rejection, included in the `deep_scan_rejected` email (replaces terse "Seller declined" with actionable context).
- AC-10.6: After seller approval, buyer sees "Pay for Deep Scan" button and is redirected to Stripe Checkout. **If the price changed** (seller adjusted deal value upward), the buyer sees an explanation before payment: "The seller provided a different deal value. Price has been updated from $X to $Y based on the higher value." The buyer can choose to proceed or cancel.
- AC-10.7: Deep scan status page shows progress with categories completed/remaining, similar to git provider scan status
- AC-10.8: Deep scan report viewer extends the static report viewer with additional deep scan sections (AI moat, architecture, code quality, tech debt, security deep, infrastructure, post-acquisition). The report header includes **deal context metadata**: "Deal value: $300,000 (confirmed by seller)" or "Deal value: $300,000 (buyer declared) / $450,000 (seller adjusted — pricing used higher value)".
- AC-10.9: **Deep scan approval error states.** The frontend handles these edge cases gracefully:
  - **Expired request:** If the seller opens the approval screen after the request has expired (backend returns 410 Gone), show: "This deep scan request has expired. The buyer can submit a new request." Disable approve/reject buttons.
  - **Buyer cancellation:** If the buyer cancels the request before the seller acts (backend returns 409 Conflict on approve/reject), show: "The buyer has withdrawn this deep scan request." Disable approve/reject buttons.
  - **Access revocation:** If the seller's git provider connection was disconnected after the request was created, the approval screen shows a warning: "Your [GitHub/GitLab] connection is no longer active. Reconnect before approving." Approve button disabled; link to Settings > Git Provider Connections.
  - **Already actioned:** If the seller (or another team member) already approved/rejected, show the current status instead of the approval form.

### FR-11: Settings

**User Story:** As a user, I can manage my account settings and connections.

> **Wireframe required (see wireframes/index.html, "Settings" tab):** 4 tabs: Profile (name, email, avatar from Clerk), Git Provider Connections (list connected GitHub/GitLab accounts with disconnect), Payment History (table of past payments), Account (delete account with confirmation modal requiring typed "DELETE").

**Acceptance Criteria:**

- AC-11.1: Settings page at `/settings`
- AC-11.2: **Profile** tab: shows name, email, avatar (from Clerk) — edit redirects to Clerk's user profile UI
- AC-11.3: **Git Provider Connections** tab: shows all connected git provider accounts (GitHub orgs/installations, GitLab accounts including self-hosted instances) with connection status and disconnect option per provider
- AC-11.4: **Payment History** tab: shows past payments with amounts, dates, and associated report IDs
- AC-11.5: **Account** tab: shows "Delete Account" section with warning text explaining data deletion consequences
- AC-11.6: Delete account flow: user clicks "Delete Account" → confirmation modal requiring typed "DELETE" → calls `DELETE /api/v1/account` with `{ "confirm": "DELETE" }` → on success, clears session and redirects to landing page
- AC-11.7: Confirmation modal lists what will be deleted: scans, reports, git provider connections (GitHub + GitLab), payment records. Notes that reports already purchased by buyers remain accessible but anonymized.

### FR-12: Notifications

> **DEFERRED TO V2.** In-app notification system: bell icon, dropdown, polling, mark-read UI. V1 uses email-only notifications. See [00a Section 16](../00a-product-overview-business.md#16-v2-roadmap).

---

## 3. Technical Requirements

### Performance

| Metric | Target |
| --- | --- |
| First Contentful Paint (marketing pages) | < 1.5s |
| Largest Contentful Paint (marketing pages) | < 2.5s |
| Time to Interactive (dashboard) | < 3s |
| Report viewer render (full report) | < 2s |
| CLS (Cumulative Layout Shift) | < 0.1 |
| Lighthouse Performance score (marketing) | > 90 |

### SEO

- Marketing pages server-side rendered (SSR/SSG)
- Proper meta tags, Open Graph tags, structured data
- Sitemap and robots.txt generated at build time
- No client-only rendering for marketing content

### Accessibility

- WCAG 2.1 AA compliance
- Keyboard navigation for all interactive elements
- Screen reader support via semantic HTML and ARIA attributes
- Color contrast ratios meet AA standards

### Browser Support

- Chrome, Firefox, Safari, Edge — latest 2 versions
- Mobile responsive (no native app in V1)
- Minimum viewport: 320px width

### Responsive Breakpoints

| Breakpoint | Width | Layout |
| --- | --- | --- |
| Mobile | 320px – 767px | Single column. Sidebar collapses to hamburger menu. Category cards stack vertically. Report header wraps. Tables scroll horizontally. |
| Tablet | 768px – 1023px | Two-column where possible. Sidebar visible as overlay (toggled). Category cards in 2-column grid. |
| Desktop | 1024px+ | Full layout. Persistent sidebar. Category cards in 3-column grid. Side-by-side risk/strength summaries. |

Key mobile behaviors:
- Dashboard scan/report lists: full-width cards, "Load More" button (no infinite scroll on mobile to avoid scroll hijacking)
- Report viewer: category cards collapse to accordion-style (one open at a time) to reduce scrolling
- Upload dropzone: tap to open file picker (no drag-and-drop on mobile)
- Pricing table: horizontally scrollable with sticky first column

---

## 4. Architecture

### 4.1 Application Structure

```
vettcode-platform-fe/
├── src/
│   ├── app/                          # Next.js App Router
│   │   ├── layout.tsx                # Root layout (Clerk provider, theme, fonts)
│   │   ├── page.tsx                  # Landing page (marketing)
│   │   ├── pricing/
│   │   │   └── page.tsx              # Pricing page
│   │   ├── how-it-works/
│   │   │   └── page.tsx              # How It Works page
│   │   ├── security/
│   │   │   └── page.tsx              # Trust & Security page
│   │   ├── privacy/
│   │   │   └── page.tsx          # Privacy Policy (static)
│   │   ├── terms/
│   │   │   └── page.tsx          # Terms of Service (static)
│   │   ├── sign-in/[[...sign-in]]/
│   │   │   └── page.tsx              # Clerk sign-in
│   │   ├── sign-up/[[...sign-up]]/
│   │   │   └── page.tsx              # Clerk sign-up
│   │   ├── verify/
│   │   │   └── [id]/
│   │   │       └── page.tsx          # Public verification (UUID in URL, no auth)
│   │   ├── sample-report/
│   │   │   └── page.tsx              # Sample report from public OSS project (public, SSG)
│   │   ├── (protected)/              # Route group — auth required
│   │   │   ├── layout.tsx            # Protected layout (auth check, sidebar)
│   │   │   ├── dashboard/
│   │   │   │   └── page.tsx          # Dashboard
│   │   │   ├── upload/
│   │   │   │   └── page.tsx          # Scan JSON upload
│   │   │   ├── scan/
│   │   │   │   └── new/
│   │   │   │       └── page.tsx      # New Scan — repo selection from all connected providers + scan trigger
│   │   │   ├── scans/
│   │   │   │   └── [scanId]/
│   │   │   │       └── page.tsx      # Scan status (git provider scans)
│   │   │   ├── reports/
│   │   │   │   └── [id]/
│   │   │   │       └── page.tsx      # Report viewer (UUID in URL)
│   │   │   ├── deep-scan/
│   │   │   │   ├── request/
│   │   │   │   │   └── page.tsx      # Deep scan request form
│   │   │   │   └── [deepScanId]/
│   │   │   │       └── page.tsx      # Deep scan status + report
│   │   │   └── settings/
│   │   │       └── page.tsx          # User settings
│   │   └── api/                      # Next.js API routes (minimal — only for callbacks)
│   │       └── github/
│   │           └── callback/
│   │               └── route.ts      # GitHub App installation callback redirect
│   │
│   ├── components/
│   │   ├── ui/                       # shadcn/ui components (Button, Card, Badge, etc.)
│   │   ├── layout/
│   │   │   ├── header.tsx            # Marketing header (nav + CTA)
│   │   │   ├── footer.tsx            # Marketing footer
│   │   │   ├── sidebar.tsx           # Dashboard sidebar nav
│   │   │   ├── protected-layout.tsx  # Auth-required layout wrapper
│   │   │   └── notification-bell.tsx # **V2** — Notification bell icon + dropdown panel (not built in V1; V1 is email-only)
│   │   ├── marketing/
│   │   │   ├── hero.tsx              # Landing page hero
│   │   │   ├── how-it-works.tsx      # Step-by-step explanation
│   │   │   ├── pricing-table.tsx     # Pricing tier cards
│   │   │   └── trust-section.tsx     # Privacy/security messaging
│   │   ├── dashboard/
│   │   │   ├── scan-list.tsx         # List of user's scans
│   │   │   ├── report-list.tsx       # List of user's reports
│   │   │   ├── deep-scan-list.tsx    # Deep scan requests
│   │   │   ├── quick-actions.tsx     # Upload + git provider connect buttons
│   │   │   └── empty-state.tsx       # Guide for new users
│   │   ├── scan/
│   │   │   ├── upload-dropzone.tsx   # JSON file upload dropzone
│   │   │   ├── scan-preview.tsx      # Client-side JSON preview
│   │   │   ├── buyer-report-preview.tsx  # Post-upload value page: live buyer preview with seller's actual grades, trust signals, sample report link
│   │   │   ├── pricing-display.tsx   # Tier + price + CTA within the value preview
│   │   │   └── scan-status.tsx       # Polling progress display
│   │   ├── report/
│   │   │   ├── report-header.tsx     # Report ID, company, date, badges
│   │   │   ├── buyer-disclosure.tsx  # Trust/verification notice
│   │   │   ├── red-flags.tsx         # Red flag alerts
│   │   │   ├── category-card.tsx     # Grade card (expandable)
│   │   │   ├── data-category.tsx     # Data-only category display
│   │   │   ├── risk-summary.tsx      # Top risks list
│   │   │   ├── strength-summary.tsx  # Top strengths list
│   │   │   ├── deep-scan-upsell.tsx  # CTA for deep scan
│   │   │   └── verification-badge.tsx # Signature verification status
│   │   ├── deep-scan/
│   │   │   ├── request-form.tsx      # Deal value + message form
│   │   │   ├── approval-card.tsx     # Seller approval UI
│   │   │   ├── deep-report-sections.tsx # AI moat, architecture, etc.
│   │   │   └── deep-scan-status.tsx  # Progress with category list
│   │   └── common/
│   │       ├── status-badge.tsx      # Colored status badges
│   │       ├── grade-badge.tsx       # Letter grade (A through F)
│   │       ├── loading-spinner.tsx   # Loading states
│   │       ├── error-message.tsx     # Error display
│   │       ├── error-boundary.tsx    # React error boundary with fallback UI
│   │       └── cookie-consent.tsx    # GDPR cookie consent banner
│   │
│   ├── lib/
│   │   ├── api.ts                    # API client (fetch wrapper with auth headers)
│   │   ├── api-types.ts             # TypeScript types auto-generated from backend OpenAPI schema (see Section 4.6)
│   │   └── utils.ts                  # Formatting helpers (price, date, LOC)
│   │
│   ├── hooks/
│   │   ├── use-api.ts               # SWR/React Query hook for API calls
│   │   ├── use-poll.ts              # Polling hook for scan/deep scan status
│   │   ├── use-notifications.ts    # **V2** — Notification polling hook (30s interval, pauses on hidden tab; not built in V1)
│   │   └── use-report-history.ts    # Hook for viewed report IDs (server-side via API)
│   │
│   └── styles/
│       └── globals.css               # Tailwind imports, custom CSS variables
│
├── public/
│   ├── favicon.ico
│   ├── og-image.png                  # Open Graph image
│   └── robots.txt
│
├── next.config.ts
├── tailwind.config.ts
├── tsconfig.json
├── package.json
└── README.md
```

### 4.2 Data Flow

```
User Browser
    │
    ├── Marketing pages ──> Vercel CDN (SSR/SSG, no API calls)
    │
    ├── Auth ──> Clerk (hosted UI components, JWT management)
    │
    └── App pages ──> API Client (lib/api.ts)
                          │
                          ├── Auth header: Clerk JWT auto-injected
                          │
                          └── REST calls to: https://api.vettcode.com/api/v1/*
                                │
                                └── vettcode-platform-be (FastAPI on Cloud Run)
```

### 4.3 Authentication Architecture

```
┌──────────────────────────────────────────┐
│  Next.js App (Vercel)                     │
│                                           │
│  <ClerkProvider>                          │
│    ├── Public routes (marketing, verify)  │
│    │    No auth required                  │
│    │                                      │
│    ├── Auth routes (/sign-in, /sign-up)   │
│    │    Clerk <SignIn> / <SignUp>          │
│    │                                      │
│    └── Protected routes (/(protected)/*)  │
│         middleware.ts enforces auth        │
│         useAuth() provides JWT            │
│         API calls include Bearer token    │
│                                           │
│  JWT lifecycle:                           │
│    1. Clerk issues JWT after social login  │
│    2. Stored in httpOnly cookie by Clerk  │
│    3. Auto-refreshed by Clerk SDK         │
│    4. Attached to API calls via getToken()│
└──────────────────────────────────────────┘
```

### 4.4 State Management

No global state library (Redux, Zustand) in V1. State is managed via:

- **Server state:** React Query (TanStack Query) for API data fetching, caching, and revalidation
- **Auth state:** Clerk React hooks (`useUser`, `useAuth`)
- **Local UI state:** React `useState` / `useReducer` for form inputs, modals, toggles
- **No client-side persistence needed** — viewed report history is stored server-side and fetched via API

**Why no global store:** V1 pages are mostly independent — dashboard, upload, report viewer have minimal shared state. API data is the primary state, and React Query handles it. Adding a store would be premature abstraction.

### 4.5 Error Boundary Strategy

React error boundaries prevent a crash in one component from taking down the entire page:

- **Root error boundary** wraps the entire app in `layout.tsx` — catches unhandled React errors, shows a "Something went wrong" fallback with a "Reload" button and a link to `/dashboard`
- **Route-level error boundaries** via Next.js `error.tsx` files in each route segment — catches per-page crashes without losing the sidebar/nav
- **Component-level boundaries** wrap the report viewer's category cards and deep scan sections — a crash in one card doesn't break the whole report
- Error boundaries log the error + component stack to the browser console (V2: send to error tracking service like Sentry)
- Fallback UI always shows a recovery action (retry button or navigation link) — never a blank screen

### 4.6 API Type Synchronization

TypeScript types must stay in sync with backend Pydantic schemas to prevent runtime contract drift:

- Backend generates an **OpenAPI 3.1 spec** from its Pydantic models (FastAPI does this automatically at `/openapi.json`)
- Frontend runs `openapi-typescript` as a build step to auto-generate `lib/api-types.ts` from the spec
- CI pipeline fails if the generated types differ from the committed types — forces developers to regenerate after backend schema changes
- The API client (`lib/api.ts`) uses these generated types for request/response typing, ensuring compile-time safety

```bash
# package.json script
"generate:types": "openapi-typescript https://api.vettcode.com/openapi.json -o src/lib/api-types.ts"
```

### 4.7 Security Headers

Vercel deployment includes the following security headers in `next.config.js`:

| Header | Value | Purpose |
| --- | --- | --- |
| `Content-Security-Policy` | `default-src 'self'; script-src 'self' https://clerk.vettcode.com https://js.stripe.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self' https://api.vettcode.com https://clerk.vettcode.com https://checkout.stripe.com https://api.stripe.com; frame-src https://js.stripe.com https://checkout.stripe.com` | Prevents XSS by restricting resource origins |
| `X-Content-Type-Options` | `nosniff` | Prevents MIME-type sniffing |
| `X-Frame-Options` | `DENY` | Prevents clickjacking |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Limits referrer leakage |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=()` | Disables unused browser APIs |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` | Enforces HTTPS |

---

## 5. Solution Design

### 5.1 API Client

A thin fetch wrapper that auto-attaches the Clerk JWT and handles common patterns:

```typescript
// lib/api.ts — server-side API client
// apiClient<T>(path, options?) → Promise<T>
// - Gets JWT via Clerk's auth() (server) or useAuth() (client)
// - Attaches Authorization: Bearer <token>
// - Throws ApiError(status, detail) on non-2xx responses
// Implementation: standard fetch wrapper pattern

// hooks/use-api.ts — client-side React Query wrapper
// useApiQuery<T>(key[], path) → UseQueryResult<T>
// useApiMutation<T>(path, method) → UseMutationResult<T>
// - Uses Clerk's useAuth().getToken() for client-side JWT
// - Wraps TanStack Query with apiClient as queryFn
```

### 5.2 Scan Upload Flow (Detailed)

```
1. User navigates to /upload
2. User drops/selects a .json file (max 10 MB)
3. Client-side: parse JSON, extract preview data
   - repo_count, total_loc, languages, red_flag count
   - This is for UX preview only — backend recalculates everything
4. User enters company name
5. User clicks "Upload & Get Pricing"
6. Frontend: POST /api/v1/scans/upload { company_name, scan_data }
7. Backend: validates, returns { scan_id, pricing_tier, price, report_count }
8. Frontend: renders POST-UPLOAD VALUE PAGE (see wireframe below)
   a. Scan summary strip (tier badge, LOC, repos, red flags)
   b. Buyer preview card — seller's actual grades displayed as the buyer would see them (AC-4.5a)
   c. Trust signals block (Ed25519 signature, public verify link, QR code)
   d. Sample report link + Price + "Pay for Signed Report" CTA
9. User clicks "Pay for Signed Report"
10. Frontend: POST /api/v1/payments/checkout { scan_id, success_url, cancel_url }
11. Backend: returns { checkout_url }
12. Frontend: window.location.href = checkout_url (redirect to Stripe)
13. Stripe handles payment
14. On success: Stripe redirects to /reports/{id} (UUID)
    - Backend has already triggered report generation via webhook
    - Frontend polls until report is ready, then renders
15. On cancel: Stripe redirects to /dashboard
    - Scan stays in "uploaded" status
    - User can retry from dashboard ("Pay for Report" button)
```

#### Post-Upload Value Page Wireframe

This page shows the seller a live preview of their own report as the buyer would see it — using the seller's actual scan data (grades, red flags, risk/strength findings). The seller experiences the report through the buyer's eyes, shifting the moment from "should I pay?" to "look at the deal asset I'm about to create."

```
┌─────────────────────────────────────────────────────────────────┐
│  ✓ Scan uploaded successfully                                    │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ STANDARD REPORT  │  42,600 LOC  │  2 repos  │  0 red flags│  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                   │
│  Here's what your buyer will see                                  │
│  ─────────────────────────────────────────────────────────────    │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │                                                              ││
│  │   VETT-2026-XXXXXX │ Acme SaaS Inc.                         ││
│  │   ✓ Platform Co-Signed  │  Mar 11, 2026  │  🟢 Recent       ││
│  │                                                              ││
│  │   Overall Grade                                              ││
│  │   ┌──────┐                                                   ││
│  │   │  B+  │  Strong technical health suitable for most        ││
│  │   └──────┘  M&A transactions.                                ││
│  │                                                              ││
│  │   ┌─────────────┬──────────────┬─────────────┐              ││
│  │   │ Security A- │ Maintain. B+ │ Handoff B   │              ││
│  │   ├─────────────┼──────────────┼─────────────┤              ││
│  │   │ Dep. Hlth B │ Activity A   │ SRE B-      │              ││
│  │   └─────────────┴──────────────┴─────────────┘              ││
│  │                                                              ││
│  │   ✓ No red flags detected                                   ││
│  │                                                              ││
│  │   ┌──────────────────────┬───────────────────────────┐      ││
│  │   │ Top Risk             │ Top Strength              │      ││
│  │   │ Est. test coverage   │ No hardcoded secrets      │      ││
│  │   │ at 42%               │ detected                  │      ││
│  │   │                      │                           │      ││
│  │   │ ↳ Buyer will see     │ ↳ Buyer will see this    │      ││
│  │   │   this flagged with  │   as a clean security    │      ││
│  │   │   remediation time   │   signal                 │      ││
│  │   │   estimate           │                           │      ││
│  │   └──────────────────────┴───────────────────────────┘      ││
│  │                                                              ││
│  │   ┌──────────────────────────────────────────────────────┐   ││
│  │   │  ✓ Ed25519 digital signature                         │   ││
│  │   │  ✓ Public verification link (buyer can check         │   ││
│  │   │    independently — no account needed)                 │   ││
│  │   │  ✓ QR code for instant mobile verification           │   ││
│  │   └──────────────────────────────────────────────────────┘   ││
│  │                                                              ││
│  │   Plus: plain-English explanations for each category,        ││
│  │   buyer impact analysis, and downloadable PDF                ││
│  │                                                              ││
│  │   [View full sample report →]                                ││
│  │                                                              ││
│  └──────────────────────────────────────────────────────────────┘│
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │                                                              ││
│  │   Standard Report                                  $299      ││
│  │   Up to 100K LOC · Your scan: 42,600 LOC                    ││
│  │                                                              ││
│  │   Join 47 sellers who have generated VettCode reports        ││
│  │                                                              ││
│  │              [ Pay for Signed Report → ]                     ││
│  │                                                              ││
│  └──────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### 5.3 Report Viewer Layout

The report viewer is the core buyer experience. Layout prioritizes scannability:

```
┌─────────────────────────────────────────────────┐
│  REPORT HEADER                                   │
│  VETT-2026-000042 │ Acme SaaS Inc.              │
│  Scanned: 2026-03-06 │ ✓ Provider Verified      │
│  [Download PDF] [Download JSON] [Verify Signature]│
├─────────────────────────────────────────────────┤
│  BUYER DISCLOSURE                                │
│  ┌─────────────────────────────────────────────┐ │
│  │ ℹ This scan was performed by VettCode's     │ │
│  │   cloud infrastructure via GitHub...         │ │
│  └─────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────┤
│  RED FLAGS (if any)                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐        │
│  │ ⚠ Secret │ │ ⚠ No CI  │ │ ⚠ Stale  │        │
│  │ detected │ │ /CD      │ │ repo     │        │
│  └──────────┘ └──────────┘ └──────────┘        │
├─────────────────────────────────────────────────┤
│  SCORED CATEGORIES (6 — each with letter grade)  │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────┐│
│  │Maintainability│ │  Security    │ │ Handoff  ││
│  │    B+         │ │    A-        │ │   C+     ││
│  │  [expand ▼]   │ │  [expand ▼]  │ │[expand ▼]││
│  ├──────────────┤ ├──────────────┤ ├──────────┤│
│  │ Dep. Health  │ │ Dev Activity │ │ SRE &    ││
│  │    B         │ │    A         │ │ Infra B- ││
│  │  [expand ▼]   │ │  [expand ▼]  │ │[expand ▼]││
│  └──────────────┘ └──────────────┘ └──────────┘│
├─────────────────────────────────────────────────┤
│  DATA-ONLY CATEGORIES (3 — no letter grade)      │
│  AI Detection │ Tech Stack │ Codebase Profile    │
│  (each expandable with detailed metrics)         │
├─────────────────────────────────────────────────┤
│  CODEBASE OVERVIEW                               │
│  3 repos │ 42,600 LOC │ TS 62% / Python 31%    │
│  Tech: Next.js 14, FastAPI, PostgreSQL, Redis   │
├─────────────────────────────────────────────────┤
│  RISK SUMMARY                                    │
│  1. Low est. test coverage (42%) — Medium severity│
│  2. 1 medium CVE in lodash — Medium severity     │
├─────────────────────────────────────────────────┤
│  STRENGTH SUMMARY                                │
│  1. Clean secrets posture — Zero hardcoded creds │
│  2. Low code duplication (4.2%)                  │
├─────────────────────────────────────────────────┤
│  DEEP SCAN UPSELL (static reports only)          │
│  Want deeper analysis? AI moat scoring,          │
│  architecture review, security audit...          │
│  [Request Deep Scan]                             │
└─────────────────────────────────────────────────┘
```

### 5.4 Polling Strategy for Async Operations

Git provider scans (GitHub, GitLab) and deep scans are async. Frontend polls for status:

```typescript
// hooks/use-poll.ts
// usePollScanStatus(scanId, enabled) → UseQueryResult
// - Polls GET /scans/{id}/status every 5s via React Query refetchInterval
// - Stops on terminal states: completed, failed, report_generated
// - Pauses when browser tab is hidden (refetchIntervalInBackground: false)
// Same pattern reused for deep scan status polling (GET /deep-scan/{id}/status)
```

### 5.5 Viewed Reports (Buyer Quick-Access)

When a buyer views a report, the frontend calls `POST /api/v1/reports/{id}/view` to record the view server-side. The dashboard fetches the buyer's viewed reports via `GET /api/v1/reports/viewed` (paginated, sorted by last viewed). This ensures viewed report history is consistent across devices and also powers the `reports_viewed` count in the buyer profile card (shown to sellers during deep scan approval).

### 5.6 Deep Scan Report Viewer

Extends the static report viewer with additional collapsible sections:

> **Wireframe required (see wireframes/index.html, "Deep Scan Report" tab):** 7 collapsible sections matching the analysis categories: AI Moat (wrapper score gauge, component table), Architecture (pattern badge, API surface stats, dependency map), Code Quality (grade, anti-pattern list, error handling assessment), Technical Debt (total effort estimate, prioritized breakdown table), Security Deep Dive (grade, vulnerability list, compliance readiness, remediation plan), Infrastructure (detected resources table, scaling readiness), Post-Acquisition Risk (migration effort, key-person risk, 90-day roadmap timeline). Each section has a letter grade badge and expandable details.

| Section | Content | Visualization |
| --- | --- | --- |
| AI Moat Analysis | Wrapper score, integration depth, component breakdown, narrative | Score gauge, component cards |
| Architecture | Pattern classification, API surface, DB schema, service dependencies | Text + optional diagram |
| Code Quality | Grade, critical path assessment, anti-patterns, error handling | Grade badge, findings list |
| Technical Debt | Total effort estimate, prioritized breakdown | Effort bar chart, priority list |
| Security Deep Dive | Grade, auth review, compliance readiness, remediation plan | Compliance checklist, priority list |
| Infrastructure | Detected resources with pricing links, scaling readiness | Resource table, grade |
| Post-Acquisition | Migration effort, key-person risk, onboarding estimate, 90-day roadmap | Timeline, risk cards |

---

## 6. Tech Stack

| Layer | Technology | Version | Rationale |
| --- | --- | --- | --- |
| Framework | Next.js | 14 (App Router) | SSR for marketing + SEO, React for dashboard, Vercel-native |
| Language | TypeScript | 5.3+ | Type safety, matches backend API schemas |
| Styling | Tailwind CSS | 3.4+ | Utility-first, fast iteration, consistent design |
| Component Library | shadcn/ui | Latest | Copy-paste components built on Radix UI — own the code, no dependency lock-in |
| Auth | Clerk (React SDK) | Latest | Pre-built `<SignIn>`, `<SignUp>`, `useAuth` — zero custom auth UI |
| Data Fetching | TanStack Query (React Query) | 5+ | Caching, background refetch, polling, mutation management |
| Charts (if needed) | Recharts | Latest | Lightweight, React-native, for any visual metrics (V1: minimal charts) |
| Icons | Lucide React | Latest | Included with shadcn/ui, consistent icon set |
| Hosting | Vercel | N/A | Native Next.js, global CDN, automatic HTTPS, preview deployments |
| Testing | Vitest + React Testing Library | Latest | Fast unit tests, component testing |
| E2E Testing | Playwright | Latest | Cross-browser E2E tests |
| Linting | ESLint + Prettier | Latest | Code quality, consistent formatting |

---

## 7. Page & Route Design

### 7.1 Route Map

| Route | Auth | Rendering | Purpose |
| --- | --- | --- | --- |
| `/` | Public | SSG | Landing page |
| `/pricing` | Public | SSG | Pricing tiers + deep scan pricing |
| `/how-it-works` | Public | SSG | Step-by-step explanation |
| `/security` | Public | SSG | Trust, privacy, verification explanation |
| `/privacy` | Public | SSR | Privacy Policy — static legal page, required for CAN-SPAM and GDPR compliance |
| `/terms` | Public | SSR | Terms of Service — static legal page, required before accepting payments |
| `/sign-in` | Public | CSR | Clerk sign-in |
| `/sign-up` | Public | CSR | Clerk sign-up |
| `/guide` | Public | SSG | User guide — single-page reference covering scanner, reports, deep scans, FAQ. Anchor sections with sticky sidebar TOC. Linked from marketing nav, footer, app sidebar, and CLI help output. |
| `/sample-report` | Public | SSG | Sample report from a B+ range OSS project (mix of strengths + red flags). Banner: "This is a sample report generated from [project]. Your report will reflect your codebase's actual health." No download/share/verify actions — replaced with "Get your own report →" CTA. |
| `/verify/[id]` | Public | SSR | Report verification (public, UUID in URL) |
| `/dashboard` | Protected | CSR | User dashboard |
| `/upload` | Protected | CSR | Scan JSON upload |
| `/scan/new` | Protected | CSR | New Scan — select repos from all connected providers (GitHub, GitLab), trigger scan. |
| `/scans/[scanId]` | Protected | CSR | Scan status (git provider scans) |
| `/reports/[id]` | Protected | CSR | Report viewer (UUID in URL) |
| `/deep-scan/request` | Protected | CSR | Deep scan request form (optional `?report={uuid}` for report-linked requests) |
| `/deep-scan/[deepScanId]` | Protected | CSR | Deep scan status + report |
| `/settings` | Protected | CSR | User settings |

**Rendering strategy:**
- **SSG** (Static Site Generation) for marketing pages — built at deploy time, served from CDN, fastest possible load
- **SSR** (Server-Side Rendering) for verification page — needs to fetch verification data server-side for SEO/sharing (report ID in meta tags)
- **CSR** (Client-Side Rendering) for app pages — protected by auth, dynamic data, no SEO needed

### 7.2 Navigation

**Marketing (unauthenticated):**
```
[Logo] [How It Works] [Pricing] [Security] [Guide]     [Sign In] [Get Started →]
```

**App (authenticated):**
```
Sidebar:                                    Top-right:
├── Dashboard                               [Avatar ▼]
│                                            (V2: add 🔔 bell icon here)
├── Upload Scan
├── New Scan
├── Settings
├── Guide ↗ (opens /guide in new tab)
└── [Sign Out]
```

### 7.3 Key Pages Detail

**Landing Page (`/`):**
- Hero: "Carfax for code — trust without exposure"
- Sub-hero: "Privacy-first technical due diligence for software M&A"
- Two CTAs: "Download Scanner (Free)" linking to GitHub releases, "View Sample Report" linking to `/sample-report` (public page showing a complete report generated from a public open-source project)
- How It Works (3 steps): Scan → Upload → Report
- Pricing preview (link to full pricing page)
- Trust signals: "No source code ever leaves your machine", "Ed25519 signed reports"
- Footer: links, legal

**Pricing Page (`/pricing`):**
- "Free Scan" card: unlimited local scans, terminal output, raw JSON
- Tier cards: LOC-based tiers with prices — sourced from [00a Section 10](../00a-product-overview-business.md) (single source of truth for pricing values)
- Deep Scan section: deal-value-based pricing with floor/cap per 00a Section 10
- FAQ: "Why not a subscription?", "What if I have multiple repos?", "What does the free scan include?"

**Dashboard (`/dashboard`):**
- Welcome message with user name
- Quick actions row (adapts to context — see AC-3.8): [Upload Scan JSON] [Connect Git Provider] [Look Up Report by ID] [Request Deep Scan]
- **Progressive disclosure** — sections only render when they contain data:
  - My Scans table: scan date, source (CLI/GitHub/GitLab), company name, status, LOC, tier, action button. *Hidden if zero scans.*
  - My Reports table: report ID, company name, date, type (static/deep), grades summary, view button. *Hidden if zero reports.*
  - Viewed Reports table: report ID, company name, last viewed, view button. *Hidden if zero viewed reports.*
  - Deep Scan Requests: incoming (seller) + outgoing (buyer), status, action buttons. *Hidden if zero requests.*
- Cross-persona prompt at bottom (dismissible, stored in localStorage):
  - For pure buyers: "Selling a business? VettCode helps you prove code quality to buyers. [Learn how →]"
  - For pure sellers: "Evaluating an acquisition? Look up a report by ID or ask a seller to share one. [Learn more →]"

---

## 8. Diagrams

### 8.1 User Flow — Seller (CLI Path)

```
Download Scanner (from vettcode.com)
        │
        ▼
Run: vettcode scan ./myrepo
        │
        ▼
Review terminal output (free)
        │
        ├── Fix issues + rescan (loop, free)
        │
        ▼
Ready to list? Upload JSON to platform
        │
        ▼
Sign up / Sign in (Clerk — Google/GitHub/Apple)
        │
        ▼
/upload → Drop JSON + enter company name
        │
        ▼
Post-upload value page: buyer preview with seller's
actual grades, trust signals, sample report link
        │
        ▼
See pricing tier → Click "Pay for Report"
        │
        ▼
Stripe Checkout → Pay
        │
        ▼
Redirect to /reports/{uuid}
        │
        ▼
View + download signed report
        │
        ▼
Click "Share Report" → copy report ID or direct link → send to buyer
```

### 8.2 User Flow — Seller (Git Provider Path)

```
Sign up / Sign in
        │
        ▼
Dashboard → "Connect Git Provider" (GitHub or GitLab)
        │
        ▼
Complete provider connection (GitHub: install App; GitLab: OAuth2, self-hosted: enter instance URL first)
        │
        ▼
Return to platform → /scan/new
        │
        ▼
Select repos from connected providers + enter company name → "Start Scan"
        │
        ▼
/scans/{scanId} — watch progress (polling)
        │
        ▼
Scan complete → Post-upload value page (same as CLI upload)
→ buyer preview with seller's grades, sample report link → "Pay for Report"
        │
        ▼
Stripe Checkout → Pay
        │
        ▼
Redirect to /reports/{uuid}
        │
        ▼
View + download signed report
        │
        ▼
Click "Share Report" → copy report ID or direct link → send to buyer
```

### 8.3 User Flow — Buyer

```
Receive report ID from seller
        │
        ▼
Navigate to platform.vettcode.com
        │
        ▼
Sign up / Sign in
        │
        ▼
Dashboard → Enter report ID in lookup box
        │
        ▼
/reports/{uuid} — view full report
        │
        ▼
Review grades, risks, strengths, explanations
        │
        ├── Download report (PDF or JSON)
        │
        ├── Verify signature at /verify/{uuid}
        │
        └── Request Deep Scan (optional)
                │
                ▼
            /deep-scan/request (optional ?report={uuid}) → Enter deal value → Submit
                │
                ▼
            Wait for seller approval
                │
                ▼
            Pay for deep scan (Stripe)
                │
                ▼
            /deep-scan/{deepScanId} — watch progress
                │
                ▼
            View deep scan report
```

### 8.4 Frontend ↔ Backend Sequence — Scan Upload

```
Browser                    Vercel (Next.js)          Backend API
  │                              │                        │
  │  Navigate to /upload         │                        │
  │─────────────────────────────>│                        │
  │  Render upload page          │                        │
  │<─────────────────────────────│                        │
  │                              │                        │
  │  Drop JSON file              │                        │
  │  (client-side parse          │                        │
  │   for preview only)          │                        │
  │                              │                        │
  │  Click "Upload"              │                        │
  │──────────────────────────────┼──────────────────────>│
  │                              │   POST /scans/upload   │
  │                              │                        │
  │                              │  { scan_id, tier,      │
  │                              │    price }             │
  │<─────────────────────────────┼────────────────────────│
  │                              │                        │
  │  Click "Pay"                 │                        │
  │──────────────────────────────┼──────────────────────>│
  │                              │  POST /payments/       │
  │                              │       checkout         │
  │                              │                        │
  │                              │  { checkout_url }      │
  │<─────────────────────────────┼────────────────────────│
  │                              │                        │
  │  Redirect to Stripe          │                        │
  │─────────────────────>  Stripe Checkout                │
  │                              │                        │
  │  Payment complete            │                        │
  │  Redirect to                 │                        │
  │  /reports/{uuid}              │                        │
  │─────────────────────────────>│                        │
  │                              │  GET /reports/         │
  │                              │       {uuid}           │
  │                              │──────────────────────>│
  │                              │                        │
  │  Render report               │  { report data }      │
  │<─────────────────────────────│<───────────────────────│
```

---

## 9. Testing Plan

### 9.1 Unit Tests (Vitest + React Testing Library)

| Area | What to Test |
| --- | --- |
| `lib/api.ts` | Auth header injection, error handling, response parsing |
| `lib/utils.ts` | Price formatting, date formatting, LOC formatting |
| `grade-badge.tsx` | Correct color for each grade (A=green, B=blue, C=yellow, D/F=red) |
| `status-badge.tsx` | Correct label and color for each scan/payment status |
| `pricing-display.tsx` | Correct tier name, price, and LOC range for each tier |
| `scan-preview.tsx` | Parses sample JSON and displays correct preview metrics |
| `red-flags.tsx` | Renders flag list, handles zero flags, handles multiple flags |

### 9.2 Integration Tests (Vitest + MSW)

Mock the backend API using MSW (Mock Service Worker):

| Flow | What to Test |
| --- | --- |
| Upload flow | File selection → upload → pricing display → checkout redirect |
| Dashboard | Fetches and displays scans, reports, deep scans correctly |
| Report viewer | Fetches report, renders all sections, handles loading/error states |
| Scan status polling | Starts polling, updates UI on status change, stops on terminal state |
| Auth redirect | Unauthenticated access to protected route → redirect to sign-in |

### 9.3 E2E Tests (Playwright)

| Scenario | Steps |
| --- | --- |
| Marketing pages load | Visit /, /pricing, /how-it-works, /security — assert content renders |
| Sign-in flow | Click sign-in → complete auth → land on dashboard |
| Upload + pay | Upload JSON → see pricing → redirect to Stripe (mock) → see report |
| Report viewer | Navigate to report → assert grades, risks, strengths render |
| Report verification | Visit /verify/{id} → assert verification result displays |

### 9.4 Visual Testing

- Storybook for component development and visual review (optional in V1, add if team grows)
- Responsive testing at 320px, 768px, 1024px, 1440px breakpoints

---

## 10. Capacity & Performance

### Bundle Size Targets

| Metric | Target |
| --- | --- |
| Initial JS bundle (marketing) | < 100 KB gzipped |
| Initial JS bundle (app, per-route) | < 120 KB gzipped (excluding async-loaded Clerk ~45 KB and React Query ~15 KB) |
| Total JS loaded (app, first page) | < 300 KB gzipped (framework + Clerk + React Query + route chunk) |
| Total page weight (marketing) | < 500 KB |

### Vercel Deployment

| Period | Traffic | Vercel Plan |
| --- | --- | --- |
| Month 1-3 | 50-200 MAU, ~1K page views/day | Free tier (sufficient) |
| Month 4-6 | 300-1,000 MAU, ~5K page views/day | Free tier (may need Pro for analytics) |
| Month 7-12 | 800-2,500 MAU, ~15K page views/day | Pro ($20/mo) |

### API Call Patterns

| Page | API Calls | Frequency |
| --- | --- | --- |
| Dashboard | GET scans, GET reports | On mount + refetch on focus |
| Upload | POST scans/upload, POST payments/checkout | On user action |
| Scan status | GET scans/{id}/status | Every 5s while active |
| Report viewer | GET reports/{id} | On mount (cached by React Query) |
| Deep scan status | GET deep-scan/{id}/status | Every 5s while active |
| **(V2)** All app pages | GET notifications?unread_only=true | Every 30s (paused when tab hidden) |
| **(V2)** Notification panel | POST notifications/{id}/read, POST notifications/read-all | On user action |

---

## 11. Deployment & Operations

> Component-specific deployment configuration. Shared infrastructure (GCP project, DNS registrar, monitoring framework) is in [05-infrastructure-sre-design.md](./05-infrastructure-sre-design.md).

### 11.1 Vercel Deployment

The frontend is deployed on Vercel using its native Next.js integration. No custom CI/CD pipeline is needed — Vercel's GitHub integration handles everything.

| Setting | Value |
| --- | --- |
| Platform | Vercel |
| Framework | Next.js 14 (App Router) |
| Build command | `next build` (default) |
| Output directory | `.next` (default) |
| Node.js version | 20.x |
| Install command | `npm ci` |

### 11.2 CI/CD Pipeline

> Vercel native — no GitHub Actions workflow needed.

**Trigger:** Push to `main`, PR to `main`

**Vercel build checks (run automatically):**
1. Install dependencies (`npm ci`)
2. Lint (`eslint .`)
3. Type check (`tsc --noEmit`)
4. Unit tests (`vitest run`)
5. Build (`next build`)

**Deployment behavior:**
- **PR:** Preview deployment on a unique URL (e.g., `vettcode-fe-git-feature-xxx.vercel.app`)
- **Merge to main:** Production deployment to `platform.vettcode.com` and `vettcode.com`
- **Rollback:** Click "Promote" on any previous deployment in Vercel dashboard (instant, atomic)

### 11.3 DNS

| Record | Target | Purpose |
| --- | --- | --- |
| `vettcode.com` | Vercel | Marketing pages + frontend app |
| `platform.vettcode.com` | Vercel (alias) | Same Next.js app, alternative URL |

SSL certificates are auto-provisioned by Vercel.

### 11.4 Environment Variables

| Variable | Source | Description |
| --- | --- | --- |
| `NEXT_PUBLIC_API_URL` | Vercel env settings | Backend API URL (`https://api.vettcode.com/api/v1`) |
| `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` | Vercel env settings | Clerk frontend key |
| `CLERK_SECRET_KEY` | Vercel env settings | Clerk server-side key (for SSR auth checks) |
| `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` | Vercel env settings | Stripe frontend key |
| `NEXT_PUBLIC_VERCEL_ANALYTICS_ID` | Vercel env settings | Vercel Analytics (optional, respects cookie consent) |

**Per-environment values:**
- Preview deployments use staging API (`https://api.staging.vettcode.com/api/v1`)
- Production deployments use production API (`https://api.vettcode.com/api/v1`)

### 11.5 Health Check

| Check | URL | Interval | Alert |
| --- | --- | --- | --- |
| Frontend uptime | `GET https://platform.vettcode.com` | Every 5 minutes | 3 consecutive failures |

Monitored via GCP Cloud Monitoring uptime check (configured in infra repo).

### 11.6 Cost Projection

| Period | Traffic | Vercel Plan | Monthly Cost |
| --- | --- | --- | --- |
| Month 1-3 | 50-200 MAU, ~1K page views/day | Free tier | $0 |
| Month 4-6 | 300-1,000 MAU, ~5K page views/day | Free tier (may need Pro for analytics) | $0 |
| Month 7-12 | 800-2,500 MAU, ~15K page views/day | Pro | $20/mo |

### 11.7 Local Development

```bash
cd vettcode-platform-fe
cp .env.example .env.local
# Set: NEXT_PUBLIC_API_URL=http://localhost:8000/api/v1
# Set: NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=pk_test_xxx (Clerk dev instance)
npm install
npm run dev
```

No GCP services or Vercel CLI needed for local development. The frontend connects to the locally running backend.

### 11.8 Deployment Strategy

| Aspect | Detail |
| --- | --- |
| Strategy | Vercel atomic deploys (immutable, instant switch) |
| Zero downtime | Yes — Vercel routes traffic atomically to new deployment |
| Rollback | "Promote" any previous deployment in Vercel dashboard (instant) |
| Preview environments | Every PR gets a unique preview URL |
| Branch deploys | Configurable per branch in Vercel settings |

---

## 12. Milestones & Tickets

### M2.1: Foundation (3 days)

| # | Task | Est. |
| --- | --- | --- |
| FE-001 | Next.js 14 project setup with App Router, TypeScript, Tailwind, shadcn/ui | 2h |
| FE-002 | Clerk integration — sign-in, sign-up, auth middleware, protected routes | 4h |
| FE-003 | API client (`lib/api.ts`) with Clerk JWT injection + error handling | 2h |
| FE-003a | OpenAPI type generation setup (`openapi-typescript` + CI check) | 1h |
| FE-004 | React Query setup, polling hook (with `refetchIntervalInBackground: false`), common types | 2h |
| FE-005 | Layout components — marketing header/footer, app sidebar, protected layout, error boundaries (root + route-level) | 5h |
| FE-006 | Common components — status-badge, grade-badge, loading-spinner, error-message, error-boundary, cookie-consent | 4h |
| FE-007 | Vercel deployment — env vars, domain config, preview deployments | 2h |

### M2.2: Marketing Pages (2 days)

| # | Task | Est. |
| --- | --- | --- |
| FE-008 | Landing page — hero, how it works, trust signals, CTAs | 4h |
| FE-009 | Pricing page — tier cards, deep scan pricing, FAQ | 3h |
| FE-010 | How It Works page — 3-step flow, diagrams | 2h |
| FE-011 | Security/Trust page — privacy architecture, verification explanation | 2h |
| FE-012 | SEO — meta tags, Open Graph, sitemap, robots.txt | 2h |
| FE-012a | Security headers in `next.config.js` (CSP, X-Frame-Options, HSTS, etc.) | 1h |
| FE-012b | Cookie consent banner (essential cookies only + optional analytics). Privacy policy and terms of service static pages with routes (/privacy, /terms) and footer links. | 3h |
| FE-012c | Guide page (`/guide`) — single-page SSG reference with anchor sections and sticky sidebar TOC. Covers: getting started, scanner (install + multi-repo + CLI flags), reports (grades, verification, freshness), uploading & payment, git provider scans, deep scans (request flow, privacy, pricing), for buyers (access, interpret, request), scoring methodology, FAQ. Linked from marketing nav, footer, app sidebar, CLI help output, and dashboard empty states. **Estimate note:** 9 content sections + sticky TOC + responsive layout + cross-linking from 5+ surfaces. 6h assumes content is pre-written; if content authoring is included, estimate 10-12h. | 6-12h |

### M2.3: Dashboard + Upload (3 days)

| # | Task | Est. |
| --- | --- | --- |
| FE-013 | Dashboard page — scan list, report list, quick actions, empty states | 6h |
| FE-014 | Upload page — dropzone, JSON preview, company name input | 4h |
| FE-015 | Post-upload value page — buyer preview card using seller's actual scan data (grades, red flags, top risk/strength), trust signals block, sample report link, pricing display, "Pay for Signed Report" CTA, social proof count. Reused for both CLI upload and git provider scan completion. | 6h |
| FE-015a | Sample report page (`/sample-report`) — full report from a B+ range OSS project with a mix of strengths and red flags. Info banner at top ("This is a sample report generated from [project]. Your report will reflect your codebase's actual health."). No download/share/verify actions — replaced with "Get your own report →" CTA. Linked from buyer preview card, marketing pages, and pricing page. **Prerequisite:** Sample OSS project must be selected and scanned before this ticket starts — selection criteria: public repo, B+ grade range, mix of strengths and red flags across categories, permissive license (MIT/Apache). | 3h |
| FE-016 | Stripe success/cancel return handling | 2h |
| FE-017 | Viewed reports list (server-side via API) + report ID lookup input | 3h |

### M2.4: Report Viewer (3 days)

| # | Task | Est. |
| --- | --- | --- |
| FE-018 | Report header — ID, company, date, verification badge, download button | 3h |
| FE-019 | Buyer disclosure section | 2h |
| FE-020 | Red flags section | 2h |
| FE-021 | Overall grade badge + scored category cards (maintainability, security, handoff, dependency health, activity, SRE) — expandable with details | 6h |
| FE-022 | Data-only category sections (AI detection, tech stack, codebase profile) | 2h |
| FE-023 | Risk summary + strength summary sections | 2h |
| FE-024 | Deep scan upsell section | 1h |
| FE-025 | Report download (signed GCS URL) | 1h |

### M2.5: Git Provider Integration (2 days)

| # | Task | Est. |
| --- | --- | --- |
| FE-026 | Git provider connect button + callback handling (GitHub + GitLab). Note: GitLab self-hosted requires an instance URL input field (validated before OAuth2 redirect) before the OAuth2 redirect. | 4h |
| FE-027 | New Scan page (`/scan/new`) — repo list from all connected providers grouped by provider (GitHub and GitLab), multi-select, labels, company name | 5h |
| FE-028 | Scan status page — polling progress bar, current step, completion/failure states | 4h |
| FE-029 | Settings page — Git provider connections list (GitHub + GitLab), disconnect per provider, account deletion. Note: GitLab self-hosted instances show the stored instance URL alongside the account name. | 4h |
| FE-029a | **DEFERRED TO V2.** Notification bell component + dropdown panel + polling hook (in-app notifications are out of V1 scope; V1 notifications are email-only) | 4h |

### M2.6: Report Verification + Deep Scan (2 days)

| # | Task | Est. |
| --- | --- | --- |
| FE-030 | Public verification page (`/verify/[id]`) — UUID in URL, no auth, shows verification result | 3h |
| FE-031 | Deep scan request form — deal value input, price calculator, message field | 3h |
| FE-032 | Deep scan seller approval UI — approve/reject from dashboard | 2h |
| FE-033 | Deep scan status page — polling, category progress | 2h |
| FE-034 | Deep scan report sections (extends report viewer) | 4h |

### M2.7: Polish + Testing (2 days)

| # | Task | Est. |
| --- | --- | --- |
| FE-035 | Responsive design pass — mobile, tablet, desktop breakpoints. **Estimate note:** Covers 15+ pages/flows (marketing, dashboard, upload, report viewer, deep scan, settings, guide). Report viewer and deep scan approval are complex at mobile widths. 4h is realistic only if Tailwind responsive utilities are used throughout development; if responsive is deferred to this ticket, estimate 8-10h. | 4-10h |
| FE-036 | Loading states and error states for all pages | 3h |
| FE-037 | Unit tests for utils, components, API client | 4h |
| FE-038 | Integration tests (MSW) for upload flow, dashboard, report viewer | 4h |
| FE-039 | E2E tests (Playwright) for critical paths | 4h |
| FE-040 | Wireframe: Deep scan seller approval screen (deal value adjustment, repo selection, privacy disclosure, consent checkbox) | 0.5 day |
| FE-041 | Wireframe: Settings page (4 tabs: profile, git connections, payment history, account deletion) | 0.25 day |
| FE-042 | **DEFERRED TO V2.** Wireframe: Notification bell + dropdown (unread badge, notification list, mark-read) (in-app notifications are out of V1 scope) | 0.25 day |
| FE-043 | Wireframe: Deep scan report viewer (7 collapsible analysis sections with grades) | 0.5 day |
| ~~FE-044~~ | ~~Privacy Policy + Terms of Service pages~~ — **MERGED INTO FE-012b** | — |
| FE-045 | **Post-MVP.** UTM Parameter Capture — Capture UTM parameters (utm_source, utm_medium, utm_campaign) from landing page URLs. Persist through signup flow and send to backend on user registration. Required for marketing attribution feedback loop. Paired with backend T10.6. | 3h |

**Total estimated time: ~18 working days (~3.5 weeks)**

### Dependencies

| Frontend Task | Depends On |
| --- | --- |
| FE-015 (upload flow) | Backend: `POST /scans/upload` + `POST /payments/checkout` |
| FE-016 (Stripe return) | Backend: Stripe webhook + report generation |
| FE-018-025 (report viewer) | Backend: `GET /reports/{id}` + report generation pipeline |
| FE-026-028 (git provider flow) | Backend: GitHub App setup + GitLab OAuth setup + `POST /api/v1/git/{provider}/scan` + scan workers |
| FE-030 (verification) | Backend: `GET /reports/{id}/verify` |
| FE-031-034 (deep scan) | Backend: deep scan orchestration |
| FE-029a (notifications) **(V2)** | Backend: `GET /notifications`, `POST /notifications/{id}/read`, `POST /notifications/read-all` |

**Parallelization:** FE-001 through FE-012 (foundation + marketing) can proceed without any backend APIs. FE-013 (dashboard shell) can use mock data. Backend API integration starts from FE-014 onward.