# CLAUDE.md

## Project Structure

- **Design docs path:** `./docs/`
- **Shared contracts:** `./docs/shared-contracts.md`
- **Reference docs path:** `./references/`
- **Tickets:** `./docs/01-scanner-tickets.md` 
- **Ticket Statuses:** `todo` → `developed` → `waiting-for-input` → `reviewed` → `resolved` →`merged`

## Initializing

1. Read the component's design documents under `./docs/`.
2. Read `./docs/shared-contracts.md` for shared types, interfaces, and cross-component dependencies.
3. Confirm your understanding of the scope before writing any code. If anything is unclear, check `./references/` or ask me.
4. You may create sub-tickets or update ticket descriptions to clarify scope.

## Workflow

### Implementation

1. Create a feature branch: `feat/<epic-slug>`.
2. Implement tickets one by one. After each ticket:
  - Run lint, type checks, and write unit tests covering core paths and edge cases.
  - Mark the ticket as `developed` and update `updated_at`.
  - Simulate two independent review agents, each with zero context from implementation:
    - **Reviewer 1 — Spec compliance:** Read the relevant spec in `./docs/` and the diff. Check correctness against the spec. Flag any deviation or misinterpretation.
    - **Reviewer 2 — Robustness:** Read the same spec and diff. Focus on edge cases, error handling, failure modes, and defensive coding.
     Neither reviewer should assume the implementer's interpretation is correct — check the spec directly. Each reviewer returns a list of issues. For each issue:
    - **Clear fix:** Apply it. Mark the ticket as `reviewed`. Update `updated_at`.
    - **Debatable or unsure:** Document the disagreement in the ticket's `notes` field. Mark the ticket as `waiting-for-input`. Do not proceed on that ticket until I weigh in.
  - After My Input
  1. Apply the agreed-upon fixes.
  2. Spin up a fresh review agent (no prior context) to review only the changed code, using the same two-lens criteria above.
  3. If the fresh review surfaces new issues, follow the same triage: fix what's clear, escalate what's debatable.
  4. Mark resolved tickets as `resolved`. Update `updated_at`.

### Merging

Once every ticket in the epic is`resolved`:

1. Merge the feature branch into `main`.
2. Mark all tickets in the epic as `merged`.
3. Move to the next epic.

## Ground Rules

- Never mark a ticket as `reviewed` without running both reviewers.
- Keep commits atomic — one ticket per commit where possible.
- If you spot a contradiction between a design doc and the shared contracts, stop and ask me.

