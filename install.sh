#!/usr/bin/env sh
# install.sh — download and install gitlab-changelog
#
# Usage:
#   curl -fsSL https://gitlab.com/kirbo/generate-changelog-from-gitlab-releases/-/raw/main/install.sh | sh
#
# Install a specific version:
#   curl -fsSL ...install.sh | sh -s -- v1.2.0
#   VERSION=v1.2.0 curl -fsSL ...install.sh | sh
#
# Change install directory (default: /usr/local/bin):
#   INSTALL_DIR=~/.local/bin curl -fsSL ...install.sh | sh

set -e

BINARY_NAME="gitlab-changelog"
REPO="kirbo/generate-changelog-from-gitlab-releases"
GITLAB_URL="https://gitlab.com"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Version priority: first argument > $VERSION env var > latest release
VERSION="${1:-${VERSION:-}}"

if [ -z "$VERSION" ]; then
  ENCODED_REPO=$(echo "$REPO" | sed 's|/|%2F|g')
  VERSION=$(curl -fsSL \
    "${GITLAB_URL}/api/v4/projects/${ENCODED_REPO}/releases?per_page=1" \
    | grep -o '"tag_name":"[^"]*"' | cut -d'"' -f4)
fi

if [ -z "$VERSION" ]; then
  echo "Error: could not determine version to install." >&2
  exit 1
fi

# Detect OS
OS=$(uname -s)
case "$OS" in
  Linux) OS=linux ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH=amd64 ;;
  aarch64) ARCH=arm64 ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

URL="${GITLAB_URL}/${REPO}/-/packages/generic/${BINARY_NAME}/${VERSION}/${BINARY_NAME}-${OS}-${ARCH}"
DEST="${INSTALL_DIR}/${BINARY_NAME}"

echo "Installing ${BINARY_NAME} ${VERSION} (${OS}/${ARCH}) to ${DEST} ..."
curl -fsSL -o "$DEST" "$URL"
chmod +x "$DEST"
echo "Done. Run: ${BINARY_NAME} --help"
