# combine-changelogs Justfile — the tools come from mise.toml (`mise install`, then run `just` inside
# the activated mise environment or as `mise exec -- just <recipe>`).

binary := "combine-changelogs"
bin_dir := "bin"
install_dir := "/usr/local/bin"

# List available commands
default:
    @just --list

# Build the binary for the host platform
build:
    go build -o {{bin_dir}}/{{binary}} .

# Build Linux amd64 binary
build-linux-amd64:
    GOOS=linux GOARCH=amd64 go build -o {{bin_dir}}/{{binary}}-linux-amd64 .

# Build Linux arm64 binary
build-linux-arm64:
    GOOS=linux GOARCH=arm64 go build -o {{bin_dir}}/{{binary}}-linux-arm64 .

# Build both Linux binaries
build-linux: build-linux-amd64 build-linux-arm64

# Build the Linux binaries with the version baked in (just build-release x.y.z; -version prints it)
build-release version="dev":
    GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version={{version}}" -o {{bin_dir}}/{{binary}}-linux-amd64 .
    GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version={{version}}" -o {{bin_dir}}/{{binary}}-linux-arm64 .

# Run go vet and check for issues
lint:
    go vet ./...

# Fail when a tracked Go file is not gofmt -s formatted (only tracked files: CI keeps the Go
# toolchain under .mise/ inside the project dir)
fmt-check:
    test -z "$(gofmt -s -l $(git ls-files '*.go'))" || { gofmt -s -l $(git ls-files '*.go'); echo "run: gofmt -s -w ." >&2; exit 1; }

# Run the tests, printing the total coverage (CI parses the `total:` line)
test:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | tail -1

# Run the tests with the race detector (needs cgo and a C compiler)
test-race:
    CGO_ENABLED=1 go test -race ./...

# Lint the shell scripts
lint-shell:
    shellcheck $(git ls-files '*.sh')

# Check the module and the Go toolchain for known vulnerabilities
vuln:
    go run golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION} ./...

# Everything CI checks: format, vet, tests, shell scripts
check: fmt-check lint test lint-shell

# Preview the next release: its notes on stderr, the version (empty if nothing to release) on stdout
next-version:
    scripts/next-version.sh --notes /dev/stderr

# Remove built binaries
clean:
    rm -rf {{bin_dir}}

# Install binary to /usr/local/bin (may require sudo)
install: build
    install -m 755 {{bin_dir}}/{{binary}} {{install_dir}}/{{binary}}
    @echo "Installed to {{install_dir}}/{{binary}}"

# Uninstall binary from /usr/local/bin
uninstall:
    rm -f {{install_dir}}/{{binary}}
    @echo "Removed {{install_dir}}/{{binary}}"

# Generate changelog for a project (usage: just run mygroup/myrepo)
run project output="CHANGELOG.md": build
    {{bin_dir}}/{{binary}} -project "{{project}}" -output "{{output}}"

# Generate changelog for a private project using GITLAB_TOKEN env var
run-private project output="CHANGELOG.md": build
    {{bin_dir}}/{{binary}} -project "{{project}}" -output "{{output}}" -token "$GITLAB_TOKEN"

# Generate changelog against a self-hosted GitLab instance
run-self-hosted url project output="CHANGELOG.md": build
    {{bin_dir}}/{{binary}} -url "{{url}}" -project "{{project}}" -output "{{output}}" -token "$GITLAB_TOKEN"
