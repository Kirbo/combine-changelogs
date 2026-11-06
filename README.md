# combine-changelogs

A CLI tool written in Go that combines releases from a GitLab project (via the API) with any number of local or remote changelog files into a single `CHANGELOG.md`.

## Requirements

- To build from source: [Go](https://go.dev/dl/) 1.25+ (`go.mod`), or [mise](https://mise.jdx.dev), which installs the exact toolchain CI and the releases use (`mise install`, see [Development](#development))
- A GitLab account and (for private projects) a [personal access token](https://gitlab.com/-/user_settings/personal_access_tokens)

## Installation

### Install script (recommended)

```bash
# Latest release
curl -fsSL https://gitlab.com/kirbo/combine-changelogs/-/raw/main/install.sh | sh

# Specific version
curl -fsSL https://gitlab.com/kirbo/combine-changelogs/-/raw/main/install.sh | sh -s -- v1.2.0

# Specific version via env var
VERSION=v1.2.0 curl -fsSL https://gitlab.com/kirbo/combine-changelogs/-/raw/main/install.sh | sh

# Custom install directory (default: /usr/local/bin)
INSTALL_DIR=~/.local/bin curl -fsSL https://gitlab.com/kirbo/combine-changelogs/-/raw/main/install.sh | sh
```

### Docker

```bash
docker pull kirbownz/combine-changelogs:latest
```

`:latest` always follows the newest release; every release is also tagged `:<version>` (e.g. `:1.5.0`). For reproducible pipelines, pin a release by digest: each [GitLab release](https://gitlab.com/kirbo/combine-changelogs/-/releases) lists its image as `kirbownz/combine-changelogs:<version>@sha256:…`.

Run it against a project (mounts the current directory so the output `CHANGELOG.md` lands on the host):

```bash
# Public project
docker run --rm -v $(pwd):/work -w /work \
  kirbownz/combine-changelogs:latest \
  combine-changelogs -project "mygroup/myrepo"

# Private project
docker run --rm \
  -e GITLAB_TOKEN="glpat-xxxxxxxxxxxx" \
  -v $(pwd):/work -w /work \
  kirbownz/combine-changelogs:latest \
  combine-changelogs -project "mygroup/myrepo"

# Self-hosted GitLab instance
docker run --rm \
  -e GITLAB_TOKEN="glpat-xxxxxxxxxxxx" \
  -v $(pwd):/work -w /work \
  kirbownz/combine-changelogs:latest \
  combine-changelogs -url "https://gitlab.example.com" -project "mygroup/myrepo"
```

### From source

```bash
git clone https://gitlab.com/kirbo/combine-changelogs.git
cd combine-changelogs
go build -o combine-changelogs .

# Install system-wide
just install
# or: sudo install -m 755 combine-changelogs /usr/local/bin/combine-changelogs
```

## Usage

```
combine-changelogs [flags]

Flags:
  -project       string   GitLab project path (e.g. group/project) or numeric ID
                          (default: $CI_PROJECT_PATH)
  -include       string   Local file path or URL to merge into the output (repeatable)
  -unreleased    string   Like -include, but stamp the section(s) as released "now" so they
                          always sort newest — for not-yet-released notes (repeatable)
  -mode          string   Source mode: api, local, or mixed (default: mixed)
  -url           string   GitLab instance URL
                          (default: $CI_SERVER_URL, then https://gitlab.com)
  -token         string   GitLab private token
                          (default: $GITLAB_TOKEN, then $CI_JOB_TOKEN)
  -output        string   Output file path (default: CHANGELOG.md)
  -ignore-regex  string   Skip releases whose name matches this regex (repeatable;
                          substring match — no anchors needed)
  -version                Print the version and exit
```

At least one of `-project`, `-include`, or `-unreleased` must be supplied.

### Source modes

| Mode | Description |
|---|---|
| `mixed` | Use both the GitLab API and any `-include` sources (default). Uses whichever sources are available. |
| `api` | GitLab API only — `-include` sources are ignored. Requires `-project` or `$CI_PROJECT_PATH`. |
| `local` | `-include` sources only — the API is never called. Useful inside GitLab CI where `CI_PROJECT_PATH` is always set but you only want local files. |

```bash
# API only
combine-changelogs -mode api -project "mygroup/myrepo"

# Local/remote files only (ignores CI_PROJECT_PATH even when set)
combine-changelogs -mode local -include CHANGELOG.md

# Both (default — same as omitting -mode)
combine-changelogs -mode mixed -project "mygroup/myrepo" -include CHANGELOG.md
```

### Public project

```bash
combine-changelogs -project "gitlab-org/gitlab"
```

### Private project

Pass the token via flag or environment variable:

```bash
# Via flag
combine-changelogs -project "mygroup/myrepo" -token "glpat-xxxxxxxxxxxx"

# Via environment variable (recommended — keeps token out of shell history)
export GITLAB_TOKEN="glpat-xxxxxxxxxxxx"
combine-changelogs -project "mygroup/myrepo"
```

### Self-hosted GitLab instance

```bash
export GITLAB_TOKEN="glpat-xxxxxxxxxxxx"
combine-changelogs \
  -url "https://gitlab.example.com" \
  -project "mygroup/myrepo"
```

### Custom output file

```bash
combine-changelogs -project "mygroup/myrepo" -output "docs/CHANGES.md"
```

### Filtering releases by name

Use `-ignore-regex` to drop releases whose name contains a match. The flag is repeatable — each pattern is tested independently and a release is dropped if any pattern matches. Matching is a substring (unanchored) match by default; add `^`/`$` to anchor.

```bash
# Drop pre-release candidates (v1.2.3-rc.1, 2.0.0-rc.2, …)
combine-changelogs -project "mygroup/myrepo" -ignore-regex '-rc\.\d+'

# Drop both release candidates and nightly builds
combine-changelogs -project "mygroup/myrepo" \
  -ignore-regex '-rc\.\d+' \
  -ignore-regex '-nightly'

# Drop anything that is not a stable semver (anchored negative pattern)
combine-changelogs -project "mygroup/myrepo" -ignore-regex '[^0-9]\.'
```

### Merging local and remote changelog files

Use `-include` to fold one or more local file paths or URLs into the output alongside API releases. All entries are sorted newest-first regardless of source.

```bash
# API releases + a local file for the current (not-yet-published) release
combine-changelogs -project "mygroup/myrepo" -include CHANGELOG.md

# A remote changelog URL
combine-changelogs -project "mygroup/myrepo" -include https://example.com/path/to/CHANGELOG.md

# Local files only — no API call (pass -mode local to suppress API even in CI)
combine-changelogs -mode local -include CHANGELOG.md -include path/to/older.md

# Multiple files + URL + API
combine-changelogs -project "mygroup/myrepo" \
  -include CHANGELOG.md \
  -include path/to/older.md \
  -include https://example.com/legacy/CHANGELOG.md
```

`-include` sources are expected to contain one or more sections delimited by markdown version headings. Both `go-semantic-release` (`## 1.2.3 (2024-01-15)`) and Keep a Changelog (`## [1.2.3] - 2024-01-15`) formats are recognised.

### Pinning the unreleased section to the top (`-unreleased`)

A not-yet-released changelog (e.g. the output of `git-cliff` for commits since the last tag) carries a heading with only a **date** — `## 2026.6.12 (2026-06-08)`. That date parses to **midnight**, so if a release was already published **earlier the same day** (with a real timestamp like 13:45), the date-sorted output would put that published release *above* your unreleased section. That's almost never what you want.

`-unreleased` behaves exactly like `-include`, but stamps every section it loads with the **current time**, so it always sorts newest:

```bash
# git-cliff wrote the pending notes to UNRELEASED-CHANGELOG.md; keep it on top:
combine-changelogs -project "mygroup/myrepo" -unreleased UNRELEASED-CHANGELOG.md
```

Use `-include` for historical/dated changelogs whose order must be preserved, and `-unreleased` for the single pending section that should always lead. The two can be combined.

### Inside a GitLab CI pipeline

GitLab automatically provides `CI_JOB_TOKEN`, `CI_PROJECT_PATH`, and `CI_SERVER_URL`, so no flags are needed for the current project.

#### Using the Docker image (recommended)

Use `kirbownz/combine-changelogs` directly as the job image — no installation step needed (pin `:<version>@sha256:…` from the release page instead of `:latest` for a reproducible pipeline):

```yaml
generate-changelog:
  image: kirbownz/combine-changelogs:latest
  stage: docs
  script:
    # API releases only (CI_PROJECT_PATH and CI_JOB_TOKEN are injected automatically)
    - combine-changelogs

    # Merge a local file produced by "go-semantic-release --dry" with past API releases.
    # Use this when the changelog is generated before the GitLab Release is published.
    # - combine-changelogs -include CHANGELOG.md
  artifacts:
    paths:
      - CHANGELOG.md
```

#### Using the install script

Download the pre-built binary for the runner's architecture at job start:

```yaml
generate-changelog:
  stage: docs
  before_script:
    # Install latest release. Pin a version with: | sh -s -- v1.2.0
    - curl -fsSL https://gitlab.com/kirbo/combine-changelogs/-/raw/main/install.sh | sh
  script:
    - combine-changelogs
  artifacts:
    paths:
      - CHANGELOG.md
```

#### Merging a local file with past API releases inside CI

`CI_PROJECT_PATH`, `CI_SERVER_URL`, and `CI_JOB_TOKEN` are injected automatically by GitLab, so no project, URL, or token flags are needed. Only the local file to merge needs to be specified:

```yaml
generate-changelog:
  image: kirbownz/combine-changelogs:latest
  stage: docs
  script:
    - combine-changelogs -include CHANGES.md
  artifacts:
    paths:
      - CHANGELOG.md
```

All entries from `CHANGES.md` and the project's GitLab Releases are merged and sorted newest-first into `CHANGELOG.md`.

> **Note:** `CI_JOB_TOKEN` can only access the **current project's** releases by default. To fetch releases from another project, use a `GITLAB_TOKEN` (personal access token) with `read_api` scope instead, and add it as a CI/CD variable.

Token resolution order (first non-empty value wins):

| Priority | Source | Header sent |
|---|---|---|
| 1 | `-token` flag | `PRIVATE-TOKEN` |
| 2 | `$GITLAB_TOKEN` env | `PRIVATE-TOKEN` |
| 3 | `$CI_JOB_TOKEN` env | `JOB-TOKEN` |

## Getting a GitLab token

1. Go to **GitLab → User Settings → Access Tokens** (or visit `https://gitlab.com/-/user_settings/personal_access_tokens`)
2. Create a token with the **`read_api`** scope
3. Copy the generated token and export it as `GITLAB_TOKEN`

## Output format

Every release becomes one `## <name> - (<YYYY-MM-DD>)` section, newest first, with its description (GitLab release notes or the included section's body) below it. The output is itself a valid `-include` source:

```markdown
# Changelog

All notable changes to this project will be documented in this file.

## v1.2.0 - (2024-03-15)

Release description from GitLab...

## v1.1.0 - (2024-02-01)

...
```

## Just commands

If you have [just](https://github.com/casey/just) installed, the following helper commands are available:

| Command | Description |
|---|---|
| `just build` | Build the binary |
| `just build-release <x.y.z>` | Build the Linux binaries with the version baked in |
| `just check` | Everything CI checks: `gofmt`, `go vet`, tests (with coverage), `shellcheck` |
| `just lint` | Run `go vet` |
| `just test` / `just test-race` | Tests with coverage / with the race detector |
| `just vuln` | `govulncheck` against the module and the Go toolchain |
| `just next-version` | Preview the next release's version and notes |
| `just install` | Build and install to `/usr/local/bin` |
| `just uninstall` | Remove from `/usr/local/bin` |
| `just clean` | Remove the built binary |
| `just run <project>` | Build (if needed) and generate changelog |
| `just run-private <project>` | Same, using `$GITLAB_TOKEN` |
| `just run-self-hosted <url> <project>` | Run against a self-hosted instance |

## Development

The toolchain (Go, just, shellcheck, git-cliff, jq, fnox, age) is pinned exactly in [`mise.toml`](mise.toml) and locked with URLs and checksums in `mise.lock`; CI installs from the lock in locked mode. `go.mod`'s `go` line is only the minimum language version.

```bash
mise install          # the pinned toolchain
mise exec -- just check
```

After changing a pin in `mise.toml`: `mise lock` and commit `mise.lock` (CI's `mise.lock in sync` job fails otherwise).

### Releases

Every push to `main` runs the pipeline on the Mac runners (protected refs on the ref-protected runner). [git-cliff](https://git-cliff.org) ([`cliff.toml`](cliff.toml), [`scripts/next-version.sh`](scripts/next-version.sh)) works out the next version from the [Conventional Commits](https://www.conventionalcommits.org) since the last `v*` tag:

| Commits since the last release | Release |
|---|---|
| a breaking change (`type!:` or a `BREAKING CHANGE:` footer) | major |
| `feat:` | minor |
| `fix:`, `chore(deps):` | patch |
| only `ci:`, `docs:`, `chore:`, `build:`, `refactor:`, `perf:`, `test:`, `style:` | none — the pipeline still builds and passes, and those commits are listed in the next release's notes |

A release pushes the multi-arch image (`:latest` and `:<version>`) to Docker Hub, uploads the Linux binaries to the Generic Packages Registry, and creates the tag and the GitLab release (with `CI_JOB_TOKEN`) whose notes git-cliff renders in the same `## <version> (<date>)` format this tool parses. The version is only stamped into the build (`-version`, image tags, release); CI never commits it back. Docker Hub credentials: [`docs/SECRETS.md`](docs/SECRETS.md).
