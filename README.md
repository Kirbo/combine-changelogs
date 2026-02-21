# gitlab-changelog

A CLI tool written in Go that fetches all releases from a GitLab repository and generates a `CHANGELOG.md` file from their descriptions.

## Requirements

- [Go](https://go.dev/dl/) 1.21+
- A GitLab account and (for private projects) a [personal access token](https://gitlab.com/-/user_settings/personal_access_tokens)

## Installation

### Install script (recommended)

```bash
# Latest release
curl -fsSL https://gitlab.com/kirbo/generate-changelog-from-gitlab-releases/-/raw/main/install.sh | sh

# Specific version
curl -fsSL https://gitlab.com/kirbo/generate-changelog-from-gitlab-releases/-/raw/main/install.sh | sh -s -- v1.2.0

# Specific version via env var
VERSION=v1.2.0 curl -fsSL https://gitlab.com/kirbo/generate-changelog-from-gitlab-releases/-/raw/main/install.sh | sh

# Custom install directory (default: /usr/local/bin)
INSTALL_DIR=~/.local/bin curl -fsSL https://gitlab.com/kirbo/generate-changelog-from-gitlab-releases/-/raw/main/install.sh | sh
```

### From source

```bash
git clone https://gitlab.com/kirbo/generate-changelog-from-gitlab-releases.git
cd generate-changelog-from-gitlab-releases
go build -o gitlab-changelog .

# Install system-wide
just install
# or: sudo install -m 755 gitlab-changelog /usr/local/bin/gitlab-changelog
```

## Usage

```
gitlab-changelog [flags]

Flags:
  -project string   GitLab project path (e.g. group/project) or numeric ID
                    (default: $CI_PROJECT_PATH)
  -url    string    GitLab instance URL
                    (default: $CI_SERVER_URL, then https://gitlab.com)
  -token  string    GitLab private token
                    (default: $GITLAB_TOKEN, then $CI_JOB_TOKEN)
  -output string    Output file path (default: CHANGELOG.md)
```

### Public project

```bash
gitlab-changelog -project "gitlab-org/gitlab"
```

### Private project

Pass the token via flag or environment variable:

```bash
# Via flag
gitlab-changelog -project "mygroup/myrepo" -token "glpat-xxxxxxxxxxxx"

# Via environment variable (recommended — keeps token out of shell history)
export GITLAB_TOKEN="glpat-xxxxxxxxxxxx"
gitlab-changelog -project "mygroup/myrepo"
```

### Self-hosted GitLab instance

```bash
export GITLAB_TOKEN="glpat-xxxxxxxxxxxx"
gitlab-changelog \
  -url "https://gitlab.example.com" \
  -project "mygroup/myrepo"
```

### Custom output file

```bash
gitlab-changelog -project "mygroup/myrepo" -output "docs/CHANGES.md"
```

### Inside a GitLab CI pipeline

Download the pre-built binary for the runner's architecture, then run it. GitLab automatically provides `CI_JOB_TOKEN`, `CI_PROJECT_PATH`, and `CI_SERVER_URL`, so no flags are needed for the current project.

```yaml
generate-changelog:
  stage: docs
  before_script:
    # Install latest release. Pin a version with: | sh -s -- v1.2.0
    - curl -fsSL https://gitlab.com/kirbo/generate-changelog-from-gitlab-releases/-/raw/main/install.sh | sh
  script:
    - gitlab-changelog
    # equivalent to:
    # gitlab-changelog -url "$CI_SERVER_URL" -project "$CI_PROJECT_PATH" -token "$CI_JOB_TOKEN"
  artifacts:
    paths:
      - CHANGELOG.md
```

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
