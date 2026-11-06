#!/bin/sh
# install-locked.sh <tool>… — install tools EXACTLY as mise.lock names them, in a CI job
# whose image has no mise (the Alpine-based docker:cli image of build-and-push-docker).
#
# For each tool, reads this platform's musl entry (linux-x64-musl / linux-arm64-musl — the
# image is Alpine) `url` and `checksum` from mise.lock, downloads the release tarball, refuses it unless its sha256 matches, and puts
# the tool's binaries in $BIN (default $HOME/.local/bin). The lock is the only source of
# versions and checksums, the same one `mise install --locked` uses in every other job —
# bump a tool in mise.toml, run `mise lock`, and this follows.
#
# Supported tools (their binaries): fnox (fnox), age (age, age-plugin-pq — through which fnox
# reaches the post-quantum keys).
# Needs: awk, tar, sha256sum and curl or wget (busybox is enough).
set -eu

LOCK="$(cd "$(dirname "$0")/.." && pwd)/mise.lock"
BIN="${BIN:-${HOME}/.local/bin}"

die() {
  echo "install-locked: $*" >&2
  exit 1
}

[ $# -gt 0 ] || die "usage: $0 <tool>…"
[ -f "${LOCK}" ] || die "no ${LOCK}"

case "$(uname -m)" in
  x86_64 | amd64) platform=linux-x64-musl ;;
  aarch64 | arm64) platform=linux-arm64-musl ;;
  *) die "unsupported architecture $(uname -m)" ;;
esac

# lock_field <tool> <field> — the field of the tool's [tools.<tool>."platforms.<platform>"] table.
lock_field() {
  awk -v hdr="[tools.$1.\"platforms.${platform}\"]" -v key="$2" '
    $0 == hdr { inside = 1; found++; next }
    /^\[/ { inside = 0 }
    inside && index($0, key " = \"") == 1 {
      v = substr($0, length(key) + 5); sub(/"$/, "", v); print v
    }
    END { if (found != 1) exit 1 }
  ' "${LOCK}"
}

fetch() { # url dest
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 3 -o "$2" "$1"
  else
    wget -q -O "$2" "$1"
  fi
}

mkdir -p "${BIN}"
tmp=$(mktemp -d)
trap 'rm -rf "${tmp}"' EXIT

for tool in "$@"; do
  case "${tool}" in
    fnox) bins="fnox" ;;
    age) bins="age age-plugin-pq" ;;
    *) die "don't know which binaries '${tool}' ships — add it to the case in $0" ;;
  esac
  url=$(lock_field "${tool}" url) || die "mise.lock has no single ${tool} entry for ${platform} (run 'mise lock')"
  checksum=$(lock_field "${tool}" checksum) || die "mise.lock has no ${tool} checksum for ${platform}"
  sha="${checksum#sha256:}"
  printf '%s' "${sha}" | grep -Eqx '[0-9a-f]{64}' || die "${tool} ${platform} checksum '${checksum}' is not a sha256"
  [ -n "${url}" ] || die "mise.lock has no ${tool} url for ${platform}"

  fetch "${url}" "${tmp}/${tool}.tgz"
  printf '%s  %s\n' "${sha}" "${tmp}/${tool}.tgz" | sha256sum -c - >/dev/null ||
    die "${tool}: sha256 of ${url} does not match mise.lock — refusing to install"
  mkdir "${tmp}/${tool}"
  tar -xzf "${tmp}/${tool}.tgz" -C "${tmp}/${tool}"
  for b in ${bins}; do
    path=$(find "${tmp}/${tool}" -type f -name "${b}" | head -n 1)
    [ -n "${path}" ] || die "${tool}: no '${b}' in ${url}"
    mv "${path}" "${BIN}/${b}"
    chmod 0755 "${BIN}/${b}"
  done
  echo "install-locked: ${tool} (${platform}) from mise.lock → ${BIN}"
done
