---
name: maintain
description: Update mise tools, CI images and deps, fix vulns, resolve breakage
disable-model-invocation: true
context: fork
agent: maintainer
model: sonnet
effort: medium
---

Run maintainer. Rules: `.claude/agents/maintainer.md`.

Pass, in order (one Conventional Commit per logical bump; never push — the human does):

1. **Survey**: `git status` clean; `mise outdated --bump`; `go list -m -u all`; `just vuln`;
   newest CI images (mise `-debian`, `docker:*-cli`, `gitlab-org/cli`) vs the digests in
   `.gitlab-ci.yml`; last `main` pipelines green (`glab api "projects/kirbo%2Fcombine-changelogs/pipelines?ref=main&per_page=3"`).
2. **mise tools** (`mise.toml`, exact pins): bump to latest releases **≥ 1 day old**
   (`mise ls-remote <tool> --json` → `created_at`), then `mise lock` + `mise install`.
   Go → `chore(deps):` (releases a patch); others → `build:`.
3. **CI images**: re-pin each to a release ≥ 7 days old, `tag@sha256:…` → `ci:`.
4. **Alpine** (`docker/Dockerfile` `ARG ALPINE_VERSION`) → `chore(deps):`.
5. **Verify** (zero warnings): `just check`, `go build ./...`, `MISE_LOCKED=1 mise install --locked --dry-run`,
   `just next-version` (version + notes render), `glab ci lint`.
6. **Secrets health** (read-only): which login path the last release's `build-and-push-docker` logged
   (`secrets: loaded fnox profile(s): publish` vs the legacy-variables fallback); if still legacy,
   remind the user of `docs/SECRETS.md` § Setting it up. Never call `op`, never create keys/variables.
7. **Docs**: README (Requirements, Just commands, Development), `.claude/CLAUDE.md` Sync section,
   `docs/SECRETS.md` match what changed.
8. **Report**: before → after per tool/image, vulns, verification output, commits, open questions.