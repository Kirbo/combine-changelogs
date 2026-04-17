# CLAUDE.md

## Project

Go CLI. Combines GitLab project releases (REST API) + local/remote changelog files → single `CHANGELOG.md`, newest-first. Single module (`combine-changelogs`), two packages: `main` + `sources/`. **Standard library only** — no third-party deps. Binary distributed via `install.sh` (GitLab Generic Packages Registry) and Docker Hub (`kirbownz/combine-changelogs`). GitLab CI cross-compiles Linux amd64/arm64, `go-semantic-release` drives version bumps + tags + release.

## Commands

| Command | What |
|---|---|
| `just build` / `go build -o bin/combine-changelogs .` | Build for host platform |
| `just build-linux` | Cross-compile amd64 + arm64 Linux binaries into `bin/` |
| `just build-release version=x.y.z` | Linux multi-arch build with `-ldflags "-X main.version=..."` |
| `go test ./...` | All tests (root + `sources/`) |
| `go test -run <Regexp> ./...` | Filter by test name |
| `go test -race ./...` | Race detector (concurrency changes) |
| `go test -cover ./...` | Coverage summary |
| `just lint` / `go vet ./...` | Static analysis |
| `gofmt -s -w .` | Format + simplify, in-place |
| `gofmt -l .` | List unformatted files (exit clean == clean) |
| `go mod tidy` + `go mod verify` | Prune + verify module graph after dep changes |
| `just install` / `just uninstall` | Install binary to `/usr/local/bin` |
| `just run <project> [output]` | Build + run against a public GitLab project |

Optional tooling — install on demand via `go install`, not mandated:

- `staticcheck ./...` — stricter static analysis
- `govulncheck ./...` — known-CVE scan against `go.sum`
- `golangci-lint run` — aggregator

## Architecture

Single process. `main.go` parses flags, resolves source mode (`api` / `local` / `mixed`), fetches + merges + sorts + writes.

- **`main`** (root) — CLI entry (`flag` package), markdown parser for included changelog files (`parseChangelogContent`, `versionHeading`, `parseVersionHeading`), `linkifyCommits`, `writeChangelog`. `resolveSources` is the flag-validation authority. Pure orchestration — no platform logic.
- **`sources/`** — `Release` struct + `Source` interface (`FetchReleases() ([]Release, error)`). `GitLabSource` implements via paginated REST (`X-Next-Page` header drives loop). Token resolution: flag > `$GITLAB_TOKEN` (`PRIVATE-TOKEN`) > `$CI_JOB_TOKEN` (`JOB-TOKEN`). Extension point: new platform → new file implementing `Source`, wire in `main`.

Flow: fetch (API and/or files) → append → `sortReleases` (`ReleasedAt`, fallback `CreatedAt`, newest-first) → `writeChangelog` with optional commit linkification.

## Subagents

`*.go`, `*_test.go`, `docker/`, `Justfile`, `install.sh`, `.gitlab-ci.yml` changes → delegate to `engineer` (TDD). `auditor` for `/audit`. `maintainer` for `/maintain`.

## Rules

### Go code

- `gofmt -s -w .` on every change. Imports in stdlib-then-project-local grouping (match existing files). `goimports -w .` handles the grouping automatically if installed — prefer it when available.
- **Check every error.** Wrap at package boundaries: `fmt.Errorf("context: %w", err)`. Never `_` an error unless the function's contract documents it as infallible (e.g. `strings.Builder.Write*`).
- Use `errors.Is` / `errors.As`, not `err == someErr` or type switches on concrete error types.
- No `panic()` outside `main` (and there, prefer `log.Fatalf`). Library code returns errors — the CLI decides what's fatal.
- HTTP: every `http.Client` has a `Timeout`. HTTP servers (if added) get read/write/idle timeouts. Always `resp.Body.Close()` (defer), always read the body before closing for keep-alive reuse when practical.
- `context.Context` is preferred for new I/O paths that may block. Not mandatory to retrofit existing code — raise in review when a change makes cancellation meaningful.
- Naming: acronyms uppercase (`URL`, `HTTP`, `ID`, `API`, `JSON`). Exported `PascalCase`, unexported `camelCase`. Receiver names short + consistent per type (e.g. `s *GitLabSource`).
- Godoc on every exported identifier. First word matches the identifier, sentence ends with a period. Existing code already follows this — don't regress.
- No global mutable state. Pass dependencies via constructors (`NewXxx`). Package-level `var` OK for compiled regex, sentinel errors, immutable defaults.
- Small interfaces defined by the **consumer**, not the producer. Don't pre-declare interfaces "just in case".
- Concurrency: goroutines need an explicit shutdown path. Hold mutexes for the shortest span. `context.Context` for cancellation.
- Prefer `time.Time` internally; format at the output boundary (`formatDate` in `main.go`).
- Secrets (`GITLAB_TOKEN`, `CI_JOB_TOKEN`) never logged, never echoed, never written to disk.

### Tests

- `testing` package only. No testify / gomega / ginkgo — stdlib keeps the dep graph empty.
- Table-driven tests where variance exists (`tests := []struct{...}{...}` + `t.Run(tc.name, ...)`).
- `t.Helper()` in every assertion helper. `t.Cleanup()` for teardown. `t.Parallel()` where safe.
- HTTP client tests use `httptest.NewServer`. No live network calls.
- Every exported function ships with tests. Coverage not enforced numerically; new code must exercise error paths as thoroughly as happy paths.
- `-race` must pass when touching goroutines / shared state.
- Avoid `TestMain` unless initialization is genuinely required. Avoid `init()` in tests.

### Tooling

- Go version in `go.mod` is the source of truth. README, Docker base images, CI all track it.
- Task runner: `just` (not `make`). Linter baseline: `go vet` via `just lint`.
- Conventional Commits. Semantic versioning via `go-semantic-release` (CI pipeline).
- **Version-bumping types** — only when change touches `*.go`, `go.mod`, `go.sum`, `docker/Dockerfile`, `install.sh`:
  - `feat:` → minor
  - `fix:` → patch
  - `chore(deps):` → patch (go.mod updates)
  - `<type>!:` → major. `!` mandatory for breaking changes. `BREAKING CHANGE:` footer is nice-to-have.
- **Non-bumping types** — CI-only, docs, `.claude/`, `Justfile`, `.gitignore`, `.gitlab-ci.yml`: `ci:`, `docs:`, `refactor:`, `test:`, `style:`, `perf:`, `chore:` (without `(deps)`), `build:`.
- Never `git push` — leave to human.
- Never `git checkout` / switch branches — commit on the branch currently checked out.

### Sync

- Docs, examples, config files sync with code.
- **Go version**: `go.mod` `go X.Y[.Z]` directive is canonical. Must match:
  - `README.md` § Requirements
  - `.gitlab-ci.yml` `golang:alpine` tag (pin explicitly if reproducibility required)
- **Binary name** (`combine-changelogs`) appears in **five places** — a rename touches all of them:
  - `Justfile` (`binary :=`)
  - `install.sh`
  - `docker/Dockerfile` (labels + `COPY`)
  - `.gitlab-ci.yml` (`BINARY_NAME` variable + artifact paths)
  - `README.md` (title, install instructions, usage examples)
- **CLI flag changes**: update README § Usage (flags table + examples). `flag.Parse()` auto-generates `-h` output.
- **CI/CD behavior changes** (`.gitlab-ci.yml` stages, jobs, artifacts): update README § GitLab CI pipeline sections if consumer-visible.
- **Docker base image version** (`ARG ALPINE_VERSION`): bump in `docker/Dockerfile` only — no other copies.
- **`.gitignore`** covers `bin/`, `coverage.out`, `*.exe`, local `CHANGELOG.md` if it's generated. Keep generated artifacts out.

### Docker

- `docker/Dockerfile` is a **runtime-only** image. It `COPY`s a pre-built binary from CI build context (`combine-changelogs-linux-${TARGETARCH}`). **Don't add `go build` inside** — breaks reproducible multi-arch buildx flow from CI.
- OCI labels populated via build args (`VERSION`, `BUILD_DATE`, `VCS_REF`). Required when changing Dockerfile.
- Alpine base only — keep image surface small. No toolchain, no package manager leftovers.

### Domain

- `sources.Source` is the extension point. New platform → new file (e.g. `sources/github.go`), implement `FetchReleases() ([]Release, error)`, add constructor (`NewGitHubSource` / `NewGitHubSourceFromEnv`), wire into `main`.
- `Release` struct JSON tags must stay compatible with GitLab REST (`tag_name`, `released_at`, etc.) — that's what `json.Unmarshal` binds against. Changes there break API parsing.
- Merging: everything goes through `sortReleases`. `ReleasedAt` primary, `CreatedAt` fallback. Newest-first output.
- Markdown version heading regex (`versionHeading`) is the sole section delimiter for included files. Both `go-semantic-release` (`## 1.2.3 (2024-01-15)`) and Keep a Changelog (`## [1.2.3] - 2024-01-15`) formats are supported. Change the regex → update both tests + README § Output format.
- Commit linkification (`linkifyCommits` + `commitRef`) requires 7–40 lowercase hex inside parens. Don't loosen — risk of false positives on hex colours / numeric strings. Empty `commitBaseURL` disables linkification (local-only mode).
- Three source modes: `api` requires `-project`, `local` requires `-include`, `mixed` (default) uses whichever sources are available. `resolveSources` is the single validation authority — don't duplicate the logic.

## Verification

Run after every change. **Must pass before commit** — never commit with failing checks.

1. `gofmt -l .` — output empty. Run `gofmt -s -w .` to auto-fix.
2. `go vet ./...` — no warnings.
3. `go build ./...` — compiles clean.
4. `go test ./...` — all tests pass. Add `-race` for concurrency changes.
5. If `go.mod` or `go.sum` changed: `go mod tidy` (no diff), `go mod verify` (clean).
6. Skip steps 2–5 for doc/config-only changes (`.md`, `.claude/`, `.gitlab-ci.yml`, `.gitignore`, `LICENSE`).

Optional (run on demand, not required pre-commit):

- `staticcheck ./...`
- `govulncheck ./...` — run after `go.mod` changes to catch known CVEs pre-commit

## Commits

Two triggers, both need user initiation:

1. User explicitly asks ("commit", "commit this", "make a commit") → run verification (§ Verification) if any code changed, generate message, commit.
2. You think it's a good time to commit → ask, include proposed message, wait for explicit approval.

Never commit without trigger — not after task, not to save progress, not after fix. If no trigger, stop. User may have more changes in mind.

A commit instruction covers only what was explicitly asked. "Commit X, then do Y" authorizes committing X — Y requires its own instruction.

Split into logical chunks — each commit = one coherent change (feature, fix, refactor, docs). No unrelated bundles. No split single logical change. If diff spans multiple concerns, stage + commit separately.

Before `git commit --amend` or `git rebase`: `git log origin/HEAD..HEAD` — empty = all pushed. If pushed, stop + ask. Otherwise proceed.

## Self-maintenance

Suggest updates to `.claude/CLAUDE.md`, `.claude/agents/`, `.claude/skills/` when gaps noticed. Ask before updating.
