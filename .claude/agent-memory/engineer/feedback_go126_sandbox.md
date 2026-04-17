---
name: Go 1.26 stdlib requires network in sandbox
description: Go 1.26 splits stdlib into its own module (module std) requiring golang.org/x deps — needs dangerouslyDisableSandbox to build/test when dep cache is cold
type: feedback
---

Go 1.26 (via mise) moved the stdlib into its own module (`module std` in `$GOROOT/src/go.mod`) with external deps on `golang.org/x/crypto` and `golang.org/x/net`. When `~/go/pkg/mod/golang.org/x/` is not cached, all `go build`/`go test` commands fail with "package X is not in std" — even for purely stdlib packages like `flag` and `fmt`.

**Why:** proxy.golang.org is not in the default sandbox allowlist, so the first cold-cache run needs sandbox disabled.

**How to apply:** When `go build`/`go test` fails with "is not in std" on a Go 1.26+ mise install, immediately retry with `dangerouslyDisableSandbox: true`. Once the x/ deps are cached, subsequent runs should work in-sandbox. The sandbox restriction that causes this is the missing `proxy.golang.org` network host in the allowlist.
