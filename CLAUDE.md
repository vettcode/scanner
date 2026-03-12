# CLAUDE.md

## Project Context
Design docs and specs live in `./docs` — read them before starting any task.

## File Boundaries
- Do not modify files in `./docs` — treat them as read-only specs
- Do not commit directly to `main` — work in feature branches

## Agentic Review Workflow

This project uses a two-agent pattern. Run Claude Code in orchestrator mode so it can spawn subagents.

### Roles
- **Implementer agent** — reads the spec in `./docs`, writes and tests the code
- **Reviewer agent** — independently reads the same spec and reviews the implementation, with no prior context from the implementer

### Workflow
The orchestrator should:

1. Spawn an **Implementer** subagent with this task:
   > "Read the relevant spec in ./docs. Implement the feature. Run tests and lint. Return a summary of what you built and any open questions."

2. Once complete, spawn a **Reviewer** subagent with this task:
   > "Read the relevant spec in ./docs and the diff/code at [path]. Review for: correctness vs spec, edge cases, error handling, and code quality. Do NOT assume the implementer's interpretation is correct — check against the spec yourself. Return a list of issues found."

3. If the reviewer finds issues, spawn a new **Implementer** subagent to address them.

4. Repeat until the reviewer returns no critical issues.

### Notes
- Each subagent should be given a fresh context — do not pass the full conversation history between them
- The goal of two separate agents is independent judgment; don't short-circuit this by summarizing one agent's reasoning to the other
- If a task is ambiguous, check `./docs` and `./references` first before asking
- Prefer small, focused commits over large ones