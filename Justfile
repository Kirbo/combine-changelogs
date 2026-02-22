# combine-changelogs Justfile

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

# Build with version info baked in (pass VERSION=x.y.z)
build-release version="dev":
    GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version={{version}}" -o {{bin_dir}}/{{binary}}-linux-amd64 .
    GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version={{version}}" -o {{bin_dir}}/{{binary}}-linux-arm64 .

# Run go vet and check for issues
lint:
    go vet ./...

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
