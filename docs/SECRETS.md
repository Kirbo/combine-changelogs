# Secrets

Only the release's Docker Hub push needs a secret: `build-and-push-docker` logs in to Docker Hub to push `kirbownz/combine-changelogs`. Its credentials live in **`fnox.toml`** as [age](https://age-encryption.org) ciphertext and are read with [fnox](https://fnox.jdx.dev), following the same tier model as Switchboard, electrolux-to-mqtt and gitlab-fleeting-plugin-upcloud: GitLab holds only one age identity per CI tier. fnox and age are pinned in [`mise.toml`](../mise.toml) and locked in `mise.lock`.

Contributors need none of this. Building, testing and the GitLab release (`CI_JOB_TOKEN`) use no secrets.

> **Status: done (2026-11-07).** The Docker Hub credentials come only from `fnox.toml`. The publish profile was verified against Docker Hub before the legacy GitLab variables were deleted (decrypted with the publish identity, the registry granted `push,pull` on `kirbownz/combine-changelogs`); a job without its identity now fails closed.

## Who can read what

A fnox **provider** encrypts to a fixed set of recipients (public keys), so the provider a secret uses decides who can read it. A **profile** groups the secrets of one job group.

| Profile | Secrets | Provider | Readers |
|---|---|---|---|
| `publish` | `DOCKERHUB_USERNAME`, `DOCKERHUB_PASSWORD` (a Docker Hub access token with push rights to `kirbownz/combine-changelogs`) | `age_publish` | maintainer, publish |

All keys are post-quantum (`age-keygen -pq`). fnox reaches them through `age-plugin-pq`, which ships with mise's age, so fnox needs each identity in its plugin form (`AGE-PLUGIN-PQ-1…`).

## Identities

| Identity | Where it lives |
|---|---|
| maintainer | 1Password item "combine-changelogs fnox keys" (field `maintainer identity`). On each maintainer machine it also sits in the OS keychain (service `combine-changelogs.fnox`), put there by `scripts/secrets.sh setup`. |
| publish | The same 1Password item (field `publish identity`), and GitLab: File variable `FNOX_AGE_KEY_FILE`, environment scope `release/publish`, protected and masked. |

`fnox.toml` names no identity, so it works unchanged everywhere. The gitignored `fnox.local.toml` is **generated** by `scripts/secrets.sh` and points every age provider at the keychain. fnox merges a provider whole, so that file repeats each provider's recipients: after pulling a recipient change, run `scripts/secrets.sh refresh-local`.

## Setting it up (once)

Needs an activated mise (`mise install`), the 1Password CLI (`op`) and an authenticated `glab`.

1. `scripts/secrets.sh init-keys` generates both identities, stores them in 1Password ("combine-changelogs fnox keys"), writes their public keys into `fnox.toml`, puts the maintainer identity in the keychain and the publish identity in GitLab (`FNOX_AGE_KEY_FILE`, scope `release/publish`, protected, masked).
2. Add the values, each read from stdin:

   ```sh
   printf '%s' '<docker hub user>'  | scripts/secrets.sh set publish DOCKERHUB_USERNAME
   printf '%s' '<push access token>' | scripts/secrets.sh set publish DOCKERHUB_PASSWORD
   ```

   The Docker Hub account is the one electrolux-to-mqtt pushes with, so you can copy its values from there without them touching the screen or disk (if that token may push this repository too):

   ```sh
   (cd ../electrolux-to-mqtt && fnox get --no-defaults -P read DOCKERHUB_USERNAME) | scripts/secrets.sh set publish DOCKERHUB_USERNAME
   (cd ../electrolux-to-mqtt && fnox get --no-defaults -P publish DOCKERHUB_PASSWORD) | scripts/secrets.sh set publish DOCKERHUB_PASSWORD
   ```

3. `scripts/secrets.sh check`, then commit `fnox.toml` and push. From that commit on, the Docker job reads fnox.

On another maintainer machine, only `scripts/secrets.sh setup` is needed.

## Migrating from the GitLab variables

`ci/secrets.sh` `dockerhub_login` decides per job:

1. The job has `FNOX_AGE_KEY_FILE` **and** this commit's `fnox.toml` holds `[profiles.publish.secrets]` → fnox (a decryption failure fails the job).
2. Otherwise the job fails — there is no fallback (the legacy GitLab variables were deleted 2026-11-07).

(Done 2026-11-07.) Once a release pipeline logged `secrets: loaded fnox profile(s): publish`, delete the GitLab variables `DOCKERHUB_PASSWORD`, `DOCKERHUB_USERNAME`, `DOCKERHUB_PASSWORD_READ` (no job uses it) and `DOCKERHUB_REGISTRY` (now `docker.io` in `.gitlab-ci.yml`).

## Commands

| Command | What |
|---|---|
| `scripts/secrets.sh init-keys` | One-time: new maintainer + publish keys (1Password, `fnox.toml`, keychain, GitLab) |
| `scripts/secrets.sh setup` | Once per machine: maintainer identity 1Password → keychain, generate `fnox.local.toml`, check |
| `scripts/secrets.sh check` | Every profile that holds secrets decrypts |
| `scripts/secrets.sh set <profile> <NAME>` | Encrypt one value (stdin) into that profile |
| `scripts/secrets.sh sync-ci` | Push the publish identity 1Password → GitLab again |
| `scripts/secrets.sh rotate maintainer\|publish` | New key: 1Password first, then recipients and re-encryption, then keychain / GitLab. Commit `fnox.toml` afterwards. |
| `fnox exec --no-defaults -P <profile> -- <cmd>` | Anything else |

## CI

`build-and-push-docker` runs in environment `release/publish` (`action: prepare`), so GitLab gives it the publish identity as `FNOX_AGE_KEY_FILE`. Because the variable is protected, only `main` pipelines get it. The job runs in the Alpine-based `docker:cli` image, which has no mise: [`ci/install-locked.sh`](../ci/install-locked.sh) installs fnox and age there from the `linux-*-musl` entries of `mise.lock` (URL + sha256, the same lock every other job installs from with `mise install --locked`). [`ci/secrets.sh`](../ci/secrets.sh) provides `dockerhub_login` and `secrets_load <profile>…`.

- The password is piped straight into `docker login --password-stdin` and unset.
- GitLab masks none of the fnox-loaded values, so jobs never `set -x` or echo them.
- **BuildKit:** the job builds on the Mac Mini's protected BuildKit server over mutual TLS, with the client certificate in the protected GitLab File variables `BUILDKIT_PROTECTED_CA` / `_CERT` / `_KEY` (the unprotected `BUILDKIT_CA` / `_CERT` / `_KEY` are for branch builds, unused here). They are credentials of the runner machine, shared by every project on it and kept in 1Password ("Kirbo Mini BuildKit protected mTLS"), not secrets of this repo, so they stay GitLab variables. They expire **2029-02-08**.

## Rotation and leaks

Re-encrypting does not revoke ciphertext already in git history. If an identity leaks, rotate it **and** change the values it could read (a new Docker Hub token, `scripts/secrets.sh set`), then commit. 1Password is the backup of record for both identities.
