# gitlab-changelog Justfile

binary := "gitlab-changelog"
install_dir := "/usr/local/bin"

# List available commands
default:
    @just --list

# Build the binary
build:
    go build -o {{binary}} .

# Build with version info baked in (pass VERSION=x.y.z)
build-release version="dev":
    go build -ldflags "-X main.version={{version}}" -o {{binary}} .

# Run go vet and check for issues
lint:
    go vet ./...

# Remove built binary
clean:
    rm -f {{binary}}

# Install binary to /usr/local/bin (may require sudo)
install: build
    install -m 755 {{binary}} {{install_dir}}/{{binary}}
    @echo "Installed to {{install_dir}}/{{binary}}"

# Uninstall binary from /usr/local/bin
uninstall:
    rm -f {{install_dir}}/{{binary}}
    @echo "Removed {{install_dir}}/{{binary}}"

# Generate changelog for a project (usage: just run mygroup/myrepo)
run project output="CHANGELOG.md":
    ./{{binary}} -project "{{project}}" -output "{{output}}"

# Generate changelog for a private project using GITLAB_TOKEN env var
run-private project output="CHANGELOG.md":
    ./{{binary}} -project "{{project}}" -output "{{output}}" -token "$GITLAB_TOKEN"

# Generate changelog against a self-hosted GitLab instance
run-self-hosted url project output="CHANGELOG.md":
    ./{{binary}} -url "{{url}}" -project "{{project}}" -output "{{output}}" -token "$GITLAB_TOKEN"
