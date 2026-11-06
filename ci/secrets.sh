# shellcheck shell=sh
# Sourced by CI jobs that need a secret: `. ci/secrets.sh`, then `dockerhub_login`, or
# `secrets_load <profile>…`.
#
# Every CI secret lives in fnox.toml as age ciphertext, encrypted only to the CI tier that
# may read it (docs/SECRETS.md). The job's GitLab environment (release/publish) scopes ONE
# protected File-type variable to it: FNOX_AGE_KEY_FILE, that tier's age identity. fnox
# decrypts with it.
#
# Migration fallback: until the maintainer has set fnox up (scripts/secrets.sh init-keys, then
# the publish values), dockerhub_login uses the legacy GitLab variables DOCKERHUB_USERNAME /
# DOCKERHUB_PASSWORD. fnox wins as soon as this commit's fnox.toml holds the publish profile
# AND the job has the identity; after that the legacy variables can be deleted.
#
# fnox and age come from mise.lock: ci/install-locked.sh installs the exact, checksummed
# artifacts the lock names (the docker:cli image has no mise).
#
# GitLab masks none of the fnox-loaded values in the job log (they are not GitLab
# variables): never `set -x`, never echo them.

export FNOX_NON_INTERACTIVE=true

# fnox_ready — whether this job reads its secrets from fnox: GitLab gave it the tier identity
# and this commit's fnox.toml holds the publish secrets.
fnox_ready() {
  [ -n "${FNOX_AGE_KEY_FILE:-}" ] && grep -q '^\[profiles\.publish\.secrets\]' fnox.toml
}

# secrets_load <profile>… — export the named profiles' secrets into THIS shell. Exits the
# job when the identity cannot decrypt them (a wrong tier is a bug).
secrets_load() {
  if [ -z "${FNOX_AGE_KEY_FILE:-}" ]; then
    echo "secrets: no FNOX_AGE_KEY_FILE in this job: does environment ${CI_ENVIRONMENT_NAME:-none} have one in GitLab (scripts/secrets.sh sync-ci), and is the ref protected?" >&2
    exit 1
  fi
  if ! command -v fnox >/dev/null 2>&1; then
    sh ci/install-locked.sh fnox age || exit 1
    export PATH="${HOME}/.local/bin:${PATH}"
  fi
  _sp=$(printf '%s' "$*" | tr ' ' ',')
  if ! _so=$(fnox export --format shell --no-defaults --if-missing error -P "${_sp}"); then
    echo "secrets: this job's identity cannot decrypt fnox profile(s): $*" >&2
    exit 1
  fi
  eval "${_so}"
  unset _so _sp
  echo "secrets: loaded fnox profile(s): $*"
}

# dockerhub_login — log in to $DOCKERHUB_REGISTRY with the Docker Hub push credentials: from
# fnox (profile publish) once it is set up, else from the legacy GitLab variables.
dockerhub_login() {
  if fnox_ready; then
    secrets_load publish
  elif [ -n "${DOCKERHUB_USERNAME:-}" ] && [ -n "${DOCKERHUB_PASSWORD:-}" ]; then
    echo "secrets: fnox is not set up for this repo yet — using the legacy GitLab variables DOCKERHUB_USERNAME / DOCKERHUB_PASSWORD (migration: docs/SECRETS.md)"
  else
    echo "secrets: no Docker Hub credentials — neither fnox (FNOX_AGE_KEY_FILE for environment release/publish + the publish profile in fnox.toml) nor the legacy GitLab variables DOCKERHUB_USERNAME / DOCKERHUB_PASSWORD" >&2
    exit 1
  fi
  printf '%s' "${DOCKERHUB_PASSWORD}" | docker login -u "${DOCKERHUB_USERNAME}" --password-stdin "${DOCKERHUB_REGISTRY:-docker.io}"
  unset DOCKERHUB_PASSWORD
}
