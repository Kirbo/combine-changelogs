---
name: audit-fix
description: Full audit-fix-commit pipeline — audit → save report → fix approved findings → verify → commit batches
disable-model-invocation: true
---

Orchestrator for full audit-fix-commit pipeline. Execute steps in order. No skipping. No commit without explicit user approval.

## Step 1: Audit (sub-agent, phases 1-3 only)

Spawn `auditor` agent via Agent tool with the exact instruction:

> Run phases 1-3 of your audit workflow (Automated Checks, Manual Review, Report). Do NOT proceed to Phase 4 or beyond — stop after producing the report. Save the full structured report to `audit-report.md` in the project root, then return a brief summary of findings (count by severity: BLOCKER / MAJOR / MINOR / NIT).

Wait for agent return before continuing.

## Step 2: Auto-triage

Read `audit-report.md`. Mark ALL findings (BLOCKER, MAJOR, MINOR, NIT) for engineer delegation. No user interaction. Zero findings → jump to Step 5.

## Step 3: Fix (engineer sub-agent)

Spawn `engineer` agent via Agent tool. Pass ALL approved findings in one bundle:

- Finding title, severity
- File path + line number (from `audit-report.md`)
- Specific fix description
- Relevant verification commands from CLAUDE.md § Verification

Wait for engineer return. Read changed files to confirm fixes applied.

## Step 4: Verify

Run full verification pipeline per CLAUDE.md § Verification:

1. `gofmt -l .` — output empty
2. `go vet ./...`
3. `go build ./...`
4. `go test ./...` (add `-race` if concurrency changed)
5. If `go.mod` / `go.sum` changed: `go mod tidy` (no diff) + `go mod verify`

Any step fails: **STOP**. Report failure. No commit. User resolves and re-invokes.

## Step 5: Commit plan (user approval gate)

Run `git diff --stat HEAD` and `git diff HEAD`. Nothing changed → report "no changes to commit" and stop.

Group changes into logical conventional commit batches:

- One commit per coherent change (fix, refactor, chore, docs, etc.)
- Scopes where appropriate (`fix(sources):`, `refactor(cli):`, `chore(deps):`, `docs:`)
- Never bundle unrelated changes

Present proposed batches with:

- Proposed commit message per batch
- Files per batch

**Wait for explicit user approval** before committing. User may adjust grouping, edit messages, or drop batches.

## Step 6: Commit approved batches

Per approved batch, in order:

1. Stage specific files
2. Commit with approved message

Never `git push`. Leave to user.

After all commits: one-line summary (N commits, N findings fixed).
