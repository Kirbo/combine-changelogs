# combine-changelogs

A CLI tool written in Go that combines releases from a GitLab project (via the API) with any number of local or remote changelog files into a single `CHANGELOG.md`.

## Requirements

- [Go](https://go.dev/dl/) 1.21+
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
  -project string   GitLab project path (e.g. group/project) or numeric ID
                    (default: $CI_PROJECT_PATH)
  -include string   Local file path or URL to merge into the output (repeatable)
  -mode   string    Source mode: api, local, or mixed (default: mixed)
  -url    string    GitLab instance URL
                    (default: $CI_SERVER_URL, then https://gitlab.com)
  -token  string    GitLab private token
                    (default: $GITLAB_TOKEN, then $CI_JOB_TOKEN)
  -output string    Output file path (default: CHANGELOG.md)
```

At least one of `-project` or `-include` must be supplied.

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

### Inside a GitLab CI pipeline

Download the pre-built binary for the runner's architecture, then run it. GitLab automatically provides `CI_JOB_TOKEN`, `CI_PROJECT_PATH`, and `CI_SERVER_URL`, so no flags are needed for the current project.

```yaml
generate-changelog:
  stage: docs
  before_script:
    # Install latest release. Pin a version with: | sh -s -- v1.2.0
    - curl -fsSL https://gitlab.com/kirbo/combine-changelogs/-/raw/main/install.sh | sh
  script:
    # API releases only (CI_PROJECT_PATH and CI_JOB_TOKEN are injected automatically)
    - combine-changelogs

    # Merge a local file produced by "go-semantic-release --dry" with past API releases.
    # Use this when the Docker image is built before the GitLab Release is created.
    # - combine-changelogs -include CHANGELOG.md
  artifacts:
    paths:
      - CHANGELOG.md
```

#### Merging a local file with past API releases inside CI

`CI_PROJECT_PATH`, `CI_SERVER_URL`, and `CI_JOB_TOKEN` are injected automatically by GitLab, so no project, URL, or token flags are needed. Only the local file to merge needs to be specified:

```yaml
generate-changelog:
  stage: docs
  before_script:
    - curl -fsSL https://gitlab.com/kirbo/combine-changelogs/-/raw/main/install.sh | sh
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

The generated file follows the [Keep a Changelog](https://keepachangelog.com) convention:

```markdown
# Changelog

All notable changes to this project will be documented in this file.

## [v1.2.0] - 2024-03-15

Release description from GitLab...

## [v1.1.0] - 2024-02-01

...
```

## Just commands

If you have [just](https://github.com/casey/just) installed, the following helper commands are available:

| Command | Description |
|---|---|
| `just build` | Build the binary |
| `just lint` | Run `go vet` |
| `just install` | Build and install to `/usr/local/bin` |
| `just uninstall` | Remove from `/usr/local/bin` |
| `just clean` | Remove the built binary |
| `just run <project>` | Build (if needed) and generate changelog |
| `just run-private <project>` | Same, using `$GITLAB_TOKEN` |
| `just run-self-hosted <url> <project>` | Run against a self-hosted instance |
