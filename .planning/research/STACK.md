# Technology Stack: Go Module Subtree Reorganization

**Project:** chroma-go-local v0.4.0 — Go Subtree Reorganization
**Researched:** 2026-03-20
**Scope:** Tools, patterns, and Go module features needed to reorganize Go implementation from repo root into `go/` subtree while preserving `github.com/amikos-tech/chroma-go-local` as the public import path.

---

## Go Module Mechanics (No new go.mod needed)

**Recommendation: Single-module approach — keep the existing go.mod at repo root.**

Do NOT create a separate `go/go.mod`. The single-module pattern places `go.mod` at the repo root, with package subdirectories beneath it. Any package under `go/internal/` becomes importable only from within the same module (enforced by the compiler, not just convention).

| Concern | Answer |
|---------|--------|
| Does `go/` subtree need its own go.mod? | No. All Go packages share the root `go.mod`. |
| Does the module path change? | No. `module github.com/amikos-tech/chroma-go-local` stays in root `go.mod`. |
| Do internal packages stay private? | Yes. `go/internal/...` is compiler-enforced: nothing outside the module can import it. |
| Does `go test ./...` from repo root cover `go/`? | Yes. `./...` recurses into all subdirectories automatically. |

**Confidence: HIGH** — Documented in [Organizing a Go module (go.dev/doc/modules/layout)](https://go.dev/doc/modules/layout). Single-module layout is the explicit official recommendation for this scenario.

---

## Root Package: Thin Facade (The Core Mechanism)

The root package (`package chroma` at `github.com/amikos-tech/chroma-go-local`) becomes a thin facade that forwards every exported symbol to the implementation in `go/internal/runtime/` (or equivalent internal package).

### Forwarding Rules by Symbol Kind

| Kind | Mechanism | Limitation |
|------|-----------|------------|
| `type T` | `type T = impl.T` (type alias) | None since Go 1.9. Generic aliases need Go 1.24. |
| `func F()` | `func F() { impl.F() }` | None. Return type and signature must match exactly. |
| `var V` | `var V = impl.V` (copy, not pointer) | `&V != &impl.V` — different addresses. Acceptable for exported config vars not used by address. |
| `const C` | `const C = impl.C` | None. |
| `error` vars | `var ErrFoo = impl.ErrFoo` | Address differs. `errors.Is` works because value is copied, but `err == impl.ErrFoo` works; `err == ErrFoo` also works if same value. |

**Type alias is the critical tool.** `type Server = impl.Server` means callers assigning `chroma.Server` to `impl.Server` work without conversion — they are the same type. A regular type definition (`type Server impl.Server`) would break interface satisfaction, struct field assignment, and type assertions.

**Confidence: HIGH** — Documented in [What's in an (Alias) Name? (go.dev/blog/alias-names)](https://go.dev/blog/alias-names) and [Codebase Refactoring with Go (go.dev/talks/2016/refactor.article)](https://go.dev/talks/2016/refactor.article).

### Circular Import Constraint

The root facade package MUST NOT be imported by the internal implementation. If `go/internal/runtime/` imports `github.com/amikos-tech/chroma-go-local` (the root), there is a circular dependency and the build breaks. The dependency direction is strictly:

```
root (facade) → go/internal/runtime/ → go/internal/library/
```

Never reverse. This rules out any design where internal packages share helpers with the root facade via imports.

**Confidence: HIGH** — Go compiler enforces this at build time.

---

## Internal Package Layout (Recommended)

```
github.com/amikos-tech/chroma-go-local/          ← root go.mod, package chroma (thin facade)
  chroma.go         ← type aliases + func forwarders for Server, Embedded, options
  config.go         ← type aliases + const/var forwarders for ServerConfig, EmbeddedConfig
  errors.go         ← var forwarders for sentinel errors
  library.go        ← func forwarder for Init, Version
  backup.go         ← type aliases + func forwarders for BackupOption, BackupManifest
  rebuild.go        ← type aliases + func forwarders
  compaction.go     ← type aliases + func forwarders
  wal_prune.go      ← type aliases + func forwarders
  go/
    internal/
      runtime/
        chroma.go       ← Server, Embedded implementation (moved from root)
        config.go
        errors.go
        ...
      library/
        library.go      ← purego FFI wiring, Init(), Version()
        library_unix.go
        library_windows.go
```

The `go/internal/` path means: packages under `go/internal/` can be imported only by code whose import path starts with `github.com/amikos-tech/chroma-go-local/go/`. Since the root facade is at the root (not under `go/`), it can still import `go/internal/runtime/` because the root is part of the same module. The compiler allows modules to import their own internal packages without restriction on the subtree prefix — internal enforcement is per-module, not per-directory-prefix.

Wait — this needs clarification. The Go spec on `internal` says: a package `a/b/c/internal/d/e/f` may be imported only by code in the tree rooted at `a/b/c/`. If the implementation lives at `go/internal/runtime/`, the full import path is `github.com/amikos-tech/chroma-go-local/go/internal/runtime/`. The tree root for this internal package is `github.com/amikos-tech/chroma-go-local/go/`. The root package at `github.com/amikos-tech/chroma-go-local` is OUTSIDE that subtree and therefore CANNOT import it.

**This is a critical layout constraint.** The correct approach is to put `internal/` directly under the module root, not under `go/`:

```
github.com/amikos-tech/chroma-go-local/
  go.mod
  chroma.go         ← thin facade (package chroma)
  config.go
  errors.go
  library.go
  backup.go
  rebuild.go
  compaction.go
  wal_prune.go
  internal/
    runtime/
      chroma.go     ← implementation (package runtime)
      config.go
      embedded.go
      errors.go
      backup.go
      rebuild.go
      compaction.go
      wal_prune.go
    library/
      library.go
      library_unix.go
      library_windows.go
```

This layout means:
- `internal/runtime/` import path: `github.com/amikos-tech/chroma-go-local/internal/runtime`
- Tree root: `github.com/amikos-tech/chroma-go-local/` — which includes the root facade. Root facade CAN import it.
- External modules CANNOT import `internal/runtime` (enforced by compiler).

**Confidence: HIGH** — Go specification on [internal packages](https://pkg.go.dev/cmd/go#hdr-Internal_Directories): "An import of a path containing the element 'internal' is disallowed if the importing code is outside the tree rooted at the parent of the 'internal' directory."

If the project still wants `go/` subdirectory for non-Go-module organizational purposes (README segregation, etc.), the internal packages must live at `internal/` under the module root, not under `go/internal/`.

---

## Platform-Specific File Handling

The current codebase has `library_unix.go` and `library_windows.go` at root using filename-based build constraints (Go's automatic OS-based file selection: `_windows.go` suffix compiles only on Windows, `_unix.go` needs a `//go:build` tag).

When moved to `internal/library/`, the same filename suffixes continue to work. No changes to build tag syntax required. File suffix `_windows.go` and `//go:build !windows` (or `_unix.go` with explicit tag) will be auto-selected by `go build` and `go test` correctly within the internal package.

**Confidence: HIGH** — This is standard Go toolchain behavior, unchanged since Go 1.17 when `//go:build` replaced `// +build`.

---

## API Compatibility Verification Tools

### Primary: `golang.org/x/exp/apidiff` + `joelanford/go-apidiff`

`go-apidiff` wraps `golang.org/x/exp/apidiff` and compares exported API between two git commits. Use it to verify the root facade exposes exactly the same surface before and after the reorganization.

**Install:**
```bash
go install github.com/joelanford/go-apidiff@latest
```

**Usage (compare current HEAD against a baseline tag):**
```bash
# Compare v0.3.4 (pre-refactor) against HEAD
go-apidiff v0.3.4 HEAD --print-compatible --repo-path .
```

If zero incompatible changes are reported, the public API surface is preserved.

**What it checks:** Exported type definitions, function signatures, method sets, constant and variable types. It does NOT check behavioral changes — only compile-time compatibility.

**What it misses:** Sentinel error identity via address comparison (`&err`). Verify that separately if any exported `var ErrFoo = errors.New(...)` is moved — after a `var ErrFoo = impl.ErrFoo` forward, callers using `errors.Is` will still work, but callers using `== &chroma.ErrFoo` (extremely rare) would see a different address.

**Confidence: MEDIUM** — Tool is actively maintained and used in the Go ecosystem; the `golang.org/x/exp/apidiff` package that underlies it is official, but the CLI wrapper's handling of cross-commit comparisons with mixed module support has a documented edge case caveat.

### Secondary: `go build ./...` and `go vet ./...`

Run after each phase:
```bash
go build ./...
go vet ./...
```

`go vet` catches shadowed imports, unreachable code in forwarders, and misused type assertions. These are zero-dependency gates that should be in CI for each phase.

**Confidence: HIGH** — Standard Go toolchain.

### Tertiary: Compile the existing test suite unchanged

The most reliable compatibility signal for this project: the existing `*_test.go` files import `github.com/amikos-tech/chroma-go-local` and exercise every exported symbol. If all tests compile and pass after the reorganization, the public API is compatible. No test source changes should be needed.

**Why this matters more than apidiff here:** The test suite already covers FFI calls, builder pattern options, server/embedded lifecycle, and error handling. A compile-time API diff is a weaker signal than "the actual test suite runs."

---

## Build System Changes

### Makefile `go test ./...` pattern

Currently:
```makefile
RUN_GO_TEST_DEBUG := CHROMA_LIB_PATH=$(abspath $(SHIM_TARGET_DEBUG)) go test -v ./...
```

After reorganization: `./...` continues to work from the repo root and will discover all packages including those in `internal/`. No Makefile change needed for test discovery.

The only possible change is if tests move into `internal/` packages. Tests for internal packages use `package runtime_test` or `package runtime` (white-box) in the same directory — this is idiomatic and `go test ./...` picks them up correctly.

**Confidence: HIGH.**

### `golangci-lint run ./...`

`./...` also covers `internal/` packages. The existing `.golangci.yml` will need one update: the `gci` formatter's `prefix()` section currently references `github.com/chaoslabs-bg/tclr-v2/` which appears to be a copy-paste from another project. This should be updated to `github.com/amikos-tech/chroma-go-local/` regardless of the reorganization. No other linter config changes are required by the subtree move.

**Confidence: HIGH.**

---

## Go Version Compatibility

The current `go.mod` declares `go 1.21`. All features needed for this reorganization are available in Go 1.21:

| Feature | Available Since | Notes |
|---------|----------------|-------|
| Type aliases (`type T = pkg.T`) | Go 1.9 | Full support for non-generic types |
| Generic type aliases | Go 1.24 | Only needed if exported types are generic — current API has no generic types |
| `internal/` package enforcement | Before Go 1.9 | Stable, unchanged |
| `go test ./...` recursive discovery | All Go module versions | No changes |
| `//go:build` constraint syntax | Go 1.17 | Current codebase already uses this |

No `go.mod` version bump is required by the reorganization. The minimum `go 1.21` in `go.mod` is sufficient.

**Confidence: HIGH** — All features verified against official release notes.

---

## What NOT to Do

### Do NOT create a second go.mod under `go/`

A second `go.mod` would create a separate module (`github.com/amikos-tech/chroma-go-local/go`). The root facade would then need a `replace` directive to reference it during development, and consumers would need to import a different module path. This defeats the entire goal of preserving the import path and adds ongoing maintenance burden (two modules, two version tags, two go.sum files).

**Why teams reach for this wrongly:** They conflate "directory subtree" with "Go module". A module is defined by its `go.mod`, not by directory structure. Subdirectories without their own `go.mod` are packages of the parent module.

### Do NOT use `go mod replace` for the internal split

Module replace directives are for substituting an external module with a local one (e.g., local development of a dependency). They are not needed when moving packages within a single module.

### Do NOT put `internal/` under `go/`

As explained in the Internal Package Layout section above, `go/internal/runtime/` cannot be imported by the root package because the root is outside the `go/` subtree. This would require either removing `internal/` from the path (losing the protection) or not having a root facade at all.

### Do NOT inline implementation into the root facade files

The facade files at root should contain only forwarding declarations (`type X = impl.X`, `func F(...) { return impl.F(...) }`). Mixing implementation logic into facade files defeats the organizational goal and makes the internal package the wrong place for tests.

### Do NOT change the public API surface during this refactor

The PROJECT.md constraint is correct: this is a purely structural change. Any API addition or change should be a separate PR after v0.4.0 lands. Mixing features into this refactor makes `go-apidiff` output harder to interpret and increases rollback risk.

---

## Alternatives Considered

| Approach | Why Not |
|----------|---------|
| Separate `go/go.mod` module | Changes import path for internal users; replace directives required in dev; two modules to version |
| `go mod replace` for split | Unnecessary and confusing; doesn't apply to intra-module package moves |
| Keep flat root layout | Fails the stated goal of separating Go/Rust/Java ownership boundaries |
| Move Go files to `go/` without internal protection | Implementation becomes importable by external modules; no boundary enforcement |
| v2 major version bump | Would break all existing importers; the goal explicitly forbids this |

---

## Installation Reference

All tools below are already present in the repo or are one-time dev installs; none change `go.mod`:

```bash
# API compatibility checker (run before/after each phase)
go install github.com/joelanford/go-apidiff@latest

# Already in use — no change
golangci-lint run ./...
go vet ./...
go build ./...
```

---

## Sources

- [Organizing a Go module — go.dev/doc/modules/layout](https://go.dev/doc/modules/layout) — HIGH confidence, official
- [Go Modules Reference — go.dev/ref/mod](https://go.dev/ref/mod) — HIGH confidence, official
- [What's in an (Alias) Name? — go.dev/blog/alias-names](https://go.dev/blog/alias-names) — HIGH confidence, official Go blog
- [Codebase Refactoring (with help from Go) — go.dev/talks/2016/refactor.article](https://go.dev/talks/2016/refactor.article) — HIGH confidence, official Go talk; forwarding patterns
- [golang.org/x/exp/apidiff — pkg.go.dev](https://pkg.go.dev/golang.org/x/exp/apidiff) — HIGH confidence, official x/ package
- [joelanford/go-apidiff — github.com](https://github.com/joelanford/go-apidiff) — MEDIUM confidence, community tool wrapping official package
- [Go 1.24 Release Notes — go.dev/doc/go1.24](https://go.dev/doc/go1.24) — HIGH confidence, official; generic type aliases GA
- [Go internal package spec — pkg.go.dev/cmd/go](https://pkg.go.dev/cmd/go#hdr-Internal_Directories) — HIGH confidence, official spec
- [Keeping Your Modules Compatible — go.dev/blog/module-compatibility](https://go.dev/blog/module-compatibility) — HIGH confidence, official
