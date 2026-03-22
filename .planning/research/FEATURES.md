# Feature Landscape: Go Module Reorganization

**Domain:** Go module subtree refactor — moving implementation from repo root into `go/` subtree while preserving public import path
**Researched:** 2026-03-20
**Confidence:** HIGH (all claims verified against official Go docs, primary sources)

---

## Table Stakes

Features users (and CI) expect. Missing any of these = the refactor has failed.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Import path unchanged | Root cause for doing this as a facade; any path change is a breaking change | Low (architectural) | `github.com/amikos-tech/chroma-go-local` must remain valid; Go does not have package-level redirects |
| All exported types re-exported at root | Users call `chroma.Server`, `chroma.Embedded`, etc. — these must resolve at root | Medium | Use `type X = impl.X` (type alias, not new type). Aliases preserve type identity; conversions are not needed |
| All exported functions re-exported at root | `chroma.Init`, `chroma.NewServer`, `chroma.NewEmbedded`, `chroma.StartServer`, `chroma.StartEmbedded` must be callable from root package | Medium | Thin wrapper functions that delegate; or `var F = impl.F` for function-typed vars |
| All exported variables/constants re-exported at root | `chroma.Success` and any other exported constants/vars must stay accessible | Low | `const X = impl.X` for constants; `var X = impl.X` for vars |
| `go test ./...` passes from repo root | CI runs `go test -v ./...` from repo root with `CHROMA_LIB_PATH` set; this must continue to work | Medium | After move, tests either live at root (facade tests) or under `go/`; Makefile `./...` must cover both |
| `make test` and `make test-release` pass | Makefile wires `CHROMA_LIB_PATH` and runs Go tests; must still work post-move | Medium | Makefile `RUN_GO_TEST_DEBUG` expands to `go test -v ./...` — must reach all Go test files |
| `make lint` passes | `golangci-lint run ./...` must cover both root facade and subtree; `.golangci.yml` at root governs | Low-Medium | golangci-lint searches config up from analyzed path; single root config covers `./...` |
| GitHub Actions CI stays green on all 3 OS | `ci.yml` runs build-test-lint on ubuntu/macos/windows matrix; nothing breaks | Medium | CI uses `CHROMA_LIB_PATH` pointing to `shim/target/debug/`; this path is OS-independent of Go layout |
| No circular imports | `go/` subtree must not import the root facade package | Low (design) | Root imports subtree; subtree must not import root. Go compiler enforces this at build time |
| `go.mod` stays at repo root | Single-module repo — no second `go.mod` in `go/`. Adding a nested `go.mod` creates a separate module requiring `replace` directives and breaks `./...` | Low (decision) | Confirmed: subtree packages are in same module; import paths just gain the directory segment |
| Public API surface unchanged | Every exported symbol (types, functions, methods, constants, variables) must remain available with identical signatures | High | This is the hardest part; verify with `golang.org/x/exp/apidiff` or `go-apidiff` before/after |

---

## Differentiators

Features that raise the quality of the refactor beyond "just works." Not expected, but clearly distinguish a well-executed reorganization.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| API compatibility gate in CI | `go-apidiff` or `golang.org/x/exp/apidiff` compares exported symbols before and after; catches accidental regressions automatically | Medium | `joelanford/go-apidiff` has a GitHub Action; can gate on incompatible-changes = 0 |
| Explicit API surface test file | A `compat_test.go` at root that assigns every public type/function to typed variables — compile-time proof the re-export surface is complete | Medium | Pattern: `var _ chroma.Server = chroma.Server{}` or `var _ = chroma.NewServer`. Fails at compile time if a symbol is missing |
| `internal/` guards on subtree packages | Place all implementation packages under `go/internal/` so they cannot be accidentally imported by external consumers | Low | `go/internal/runtime/`, `go/internal/library/` etc. External import → compile error. Free to refactor internally |
| Import path documented in facade files | Root `.go` files contain `// Package chroma provides ...` godoc comment explaining the root is a facade; where the implementation lives | Low | Discoverability for future maintainers |
| `make fmt` covers subtree | `fmt-go` target runs `gofmt -w .` and `goimports -w .` from root; these recurse into `go/` automatically so no Makefile change needed — verify this holds | Low | Confirm `gofmt ./...` vs `gofmt .` behavior after move |
| Test helper / setup code consolidated | Test setup helpers (temp dirs, `CHROMA_LIB_PATH` guards, etc.) extracted to a shared `testutil` or `go/internal/testutil` to reduce per-file duplication | Medium | Reduces risk of test drift post-move |
| `go vet ./...` clean | `go vet` runs as part of `golangci-lint`; all re-exports and aliases must pass vet checks (no shadow, no unused params in wrappers) | Low | Aliases are vet-transparent; thin wrapper functions add negligible surface |
| Examples updated and verified | `examples/go/basic/main.go` imports `github.com/amikos-tech/chroma-go-local`; must still compile and run correctly post-refactor | Low | Example is an external-consumer simulation; verifies the facade from a user's perspective |

---

## Anti-Features

Features to explicitly NOT build during this refactor.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Second `go.mod` in `go/` subtree | Creates a separate Go module; breaks `go test ./...`, requires `replace` directives, complicates CI caching, and makes `go-apidiff` harder to run | Keep single `go.mod` at repo root; all packages in `go/` are within the same module |
| `replace` directive for internal path | `replace` directives in published modules cause downstream breakage; banned for non-main modules | Not needed: same-module packages are resolved by directory path automatically |
| New exported API surface | This milestone is purely structural; any new types, functions, or methods belong in a separate feature issue | File separate issues for new API work; keep this PR to zero-delta on public API |
| Renaming exported symbols | Even during a refactor, renaming an exported symbol (e.g., `Server` to `ChromaServer`) is a breaking change | Keep all names identical; use `type X = impl.X` to preserve the original name |
| Deleting old test files before confirming pass | Moving test files without verifying they still pass under the new layout risks silent test loss | Move, run `make test`, confirm green, then clean up |
| Splitting into two separate modules (Go/Java) | Would require module federation, `go.work` or explicit versioned requires; adds long-term maintenance burden | Single-module repo stays simpler; language separation is a directory concern, not a module boundary |
| `go.work` (workspace) file | Go workspaces are for local multi-module development; adding one would be checked in accidentally and cause confusion for contributors | Unnecessary when keeping a single `go.mod` |
| Over-layering internal packages | Creating deep hierarchies like `go/internal/runtime/server/http/handler/` for the current codebase size violates the CLAUDE.md "radically simple" directive | Keep subtree shallow: `go/internal/` with a flat set of files or one level of grouping |

---

## Feature Dependencies

```
Import path unchanged
  └─► type aliases at root  (types must alias impl types, not copy them)
      └─► function wrappers at root  (can reference aliased types in signatures)
          └─► constant/var re-exports at root  (simple; no dependency ordering)

go.mod at root (single module)
  └─► go/internal/ packages within same module  (no replace needed)
      └─► go test ./... covers both root + go/  (single invocation)
          └─► make test passes  (Makefile expands ./...)
              └─► CI green  (CI calls make test via pwsh CHROMA_LIB_PATH pattern)

API surface unchanged
  └─► explicit compile-time compat test  (compat_test.go assigns every symbol)
      └─► go-apidiff gate in CI  (machine-verified, not human-reviewed)
```

---

## MVP Recommendation

The minimum viable refactor (what must land for v0.4.0 to be correct):

1. Move implementation Go files to `go/internal/` — the actual migration
2. Create root facade with type aliases, function wrappers, and constant re-exports — preserves API
3. Update `go test ./...` invocation in Makefile and CI to reach both root and `go/` — keeps CI green
4. Update `.golangci.yml` if path exclusions reference root-level filenames — keeps lint clean
5. Verify `make test`, `make test-release`, `make lint`, `make test-all` all pass
6. Update `examples/go/basic/main.go` if any imports changed (they should not)
7. Update docs (`GO_API_SURFACE.md`, `README.md`) to reflect new layout

Defer (valuable but not blocking v0.4.0):
- `go-apidiff` CI gate: useful signal, but manual API surface test file gives same protection at lower setup cost
- `testutil` consolidation: worthwhile cleanup but orthogonal to structural refactor
- godoc comments on facade files: nice-to-have, not correctness-critical

---

## Sources

- [Organizing a Go module — official layout guidance](https://go.dev/doc/modules/layout) — HIGH confidence
- [What's in an (Alias) Name? — Go blog on type aliases](https://go.dev/blog/alias-names) — HIGH confidence
- [Codebase Refactoring (with help from Go) — Russ Cox 2016](https://go.dev/talks/2016/refactor.article) — HIGH confidence
- [Keeping Your Modules Compatible — Go blog](https://go.dev/blog/module-compatibility) — HIGH confidence
- [Use internal packages to reduce your public API surface — Dave Cheney](https://dave.cheney.net/2019/10/06/use-internal-packages-to-reduce-your-public-api-surface) — HIGH confidence
- [Backward Compatibility, Go 1.21, and Go 2 — Go blog](https://go.dev/blog/compat) — HIGH confidence
- [go-apidiff tool](https://github.com/joelanford/go-apidiff) — MEDIUM confidence (community tool, widely used)
- [golang.org/x/exp/apidiff — official experimental apidiff](https://pkg.go.dev/golang.org/x/exp/apidiff) — HIGH confidence
- [Alias declarations proposal — Go issue #16339](https://github.com/golang/go/issues/16339) — HIGH confidence (accepted, shipped Go 1.9)
- Project-specific: Makefile, `ci.yml`, `go.mod`, `chroma.go` — read directly from codebase
