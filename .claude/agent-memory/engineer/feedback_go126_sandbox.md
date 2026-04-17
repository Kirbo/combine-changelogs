---
name: Go 1.26 stdlib requires network in sandbox
description: Go 1.26 splits stdlib into its own module (module std) requiring golang.org/x deps — needs dangerouslyDisableSandbox to build/test when dep cache is cold
type: feedback
---

Go 1.26 (via mise) moved stdlib to own module (`module std` in `$GOROOT/src/go.mod`) with external deps on `golang.org/x/crypto`, `golang.org/x/net`. When `~/go/pkg/mod/golang.org/x/` uncached, all `go build`/`go test` fail "package X is not in std" — even pure stdlib pkgs like `flag`, `fmt`.

**Why:** proxy.golang.org not in default sandbox allowlist, so cold-cache run needs sandbox disabled.

**How to apply:** `go build`/`go test` fails "is not in std" on Go 1.26+ mise install → retry `dangerouslyDisableSandbox: true`. Once x/ deps cached, subsequent runs work in-sandbox. Root cause: missing `proxy.golang.org` host in sandbox allowlist.