# Project Research Summary

**Project:** chroma-go-local v0.4.0 — Go Subtree Reorganization
**Domain:** Go module structural refactoring — implementation subtree move with import-path preservation
**Researched:** 2026-03-20
**Confidence:** HIGH

## Executive Summary

This milestone is a pure structural refactor: move all Go implementation files from the repo root into an `internal/` subtree while keeping `github.com/amikos-tech/chroma-go-local` as the unchanged public import path. The standard Go approach is a single-module layout with `go.mod` at the repo root and a thin facade at the root package that re-exports every public symbol via type aliases, function vars, and const/var forwarders. No second `go.mod`, no `replace` directives, and no version bump are needed. This pattern is explicitly documented in Go's official module layout guide and is the recommended approach for this class of reorganization.

The recommended target layout places implementation under `internal/runtime/` and `internal/library/` anchored at the module root (not under `go/`). This is a critical constraint: `internal/` must be at the module root level so the root facade package can legally import it. Moving `internal/` under a `go/` subdirectory would make the root facade unable to import the implementation packages — a compile-time error. The facade itself must contain zero logic and zero state: only `type X = impl.X` aliases, direct `var` assignments for function references and sentinel errors, and `const` forwarding. Methods on types are inherited automatically through aliases; no method wrapping is needed.

The primary risks are mechanical and detectable early. The most dangerous is accidentally duplicating global FFI state (`libHandle`, `libOnce`, `ffiMu`, and ~40 function pointers) between the root and implementation packages — this must all live in `internal/runtime`, with the root containing zero `var` declarations. The second risk is test file misclassification: white-box tests (`package chroma`) must move alongside implementation and become `package runtime` tests; a new `compat_test.go` at root in `package chroma_test` provides the public API regression gate. Both risks are caught immediately by `go test -race ./...` and a drop in test count, making them detectable before merging.

## Key Findings

### Recommended Stack

The refactor uses only the existing Go toolchain with no new dependencies. The critical mechanism is Go's type alias syntax (`type A = B`, available since Go 1.9), which preserves full type identity — the facade's `chroma.Server` and the implementation's `runtime.Server` are the same type, requiring no conversion. Function vars (`var F = impl.F`) delegate functions without interface indirection overhead. The current `go.mod` declaring `go 1.21` is sufficient for all required features.

**Core technologies:**
- Go type aliases (`type A = B`): facade type re-export — preserves type identity, method sets, and interface satisfaction without wrapping
- Go `internal/` package enforcement: implementation protection — compiler-enforced; external modules cannot import `internal/` packages, providing a firm API boundary
- `go-apidiff` / `golang.org/x/exp/apidiff`: API compatibility verification — compare exported symbols before and after; confirms zero regressions
- `go test -race ./...`: concurrency correctness gate — detects duplicated global state across packages during migration
- `golangci-lint run ./...`: lint coverage — single root config covers entire module; requires one config fix (stale `gci` prefix)

### Expected Features

The refactor is defined by what must remain unchanged, not what is added.

**Must have (table stakes):**
- Import path `github.com/amikos-tech/chroma-go-local` unchanged — Go has no package-level redirects; any change is immediately breaking
- All exported types re-exported as aliases at root — `chroma.Server`, `chroma.Embedded`, all option/config/request/response types
- All exported functions delegated at root — `Init`, `NewServer`, `StartServer`, `NewEmbedded`, `StartEmbedded`, all `With*` option constructors
- All exported constants and sentinel error vars forwarded at root — `Success`, `ErrNullPointer`, etc., using direct assignment (not wrapping)
- `make test`, `make test-release`, `make lint`, `make test-all` all pass — CI matrix (linux/mac/windows) green
- Zero new public API surface — purely structural change; any API additions belong in a separate issue

**Should have (quality indicators):**
- `compat_test.go` at root compiling every exported symbol — compile-time regression gate, no tool dependency
- `go test -race ./...` passes at every phase — detects duplicated globals during partial migration
- `.golangci.yml` `gci` prefix fixed (`github.com/chaoslabs-bg/tclr-v2/` is a stale leftover; replace with `github.com/amikos-tech/chroma-go-local/`)
- Cross-compile gate: `GOOS=linux/darwin/windows go build ./...` — verifies OS-specific files (`library_unix.go`, `library_windows.go`) still apply correctly
- `go-apidiff` run as a pre-merge sanity check — machine-verifiable API surface comparison vs. v0.3.4 tag

**Defer (post-v0.4.0):**
- `go-apidiff` as a permanent CI gate — useful but the compile-time compat test covers the same ground at lower setup cost
- `testutil` package consolidation — worthwhile cleanup, orthogonal to structural move
- godoc comments on facade files — nice-to-have discoverability improvement

### Architecture Approach

The target architecture has three Go layers: a thin root facade (`package chroma`), an implementation package (`internal/runtime`), and a platform-specific loader (`internal/library`). Rust (`shim/`) and Java (`java/`) subtrees are completely unchanged — they have no dependency on any Go package and are out of scope. The dependency direction is strictly one-way and must never be reversed.

**Major components:**
1. Root facade (`github.com/amikos-tech/chroma-go-local`, `package chroma`) — zero logic; only type aliases, function vars, and const/var forwards pointing to `internal/runtime`
2. Runtime implementation (`internal/runtime`, `package runtime`) — all Go logic: FFI bindings, `Server`/`Embedded` lifecycle, all operation types and functions; imports `internal/library`
3. Library loader (`internal/library`, `package library`) — dynamic `.so`/`.dylib`/`.dll` resolution; platform-specific files (`library_unix.go`, `library_windows.go`); no dependency on `runtime`
4. Rust shim (`shim/`, not a Go package) — C FFI symbols called by `runtime` via purego; unchanged
5. Java scaffold (`java/`, not a Go package) — JNA + Panama bindings to Rust shim; unchanged

### Critical Pitfalls

1. **`internal/` anchor at wrong level** — placing implementation under `go/internal/` rather than `internal/` at module root makes the root facade unable to import it (compile error: "use of internal package not allowed"). Decision must be made before any files move.

2. **Duplicated global FFI state** — `libHandle`, `libOnce`, `ffiMu`, and all ~40 FFI function pointers must live entirely in `internal/runtime`; the root facade must have zero `var` state declarations. Any duplication causes double-initialization and race conditions. Detect with `go test -race ./...` and by auditing root-level `var` declarations post-migration.

3. **Type definitions instead of type aliases in facade** — `type Server runtime.Server` creates a new type with no methods, breaking all callers. Always use `type Server = runtime.Server`. Catch by running `go build ./...` with the existing test suite; method call failures surface immediately.

4. **White-box test file misclassification** — tests using `package chroma` test unexported symbols that move to `internal/runtime`. Move test files alongside implementation, rename their package declarations to `package runtime`, and add a root-level `compat_test.go` in `package chroma_test` as the compatibility gate. Detect by monitoring per-package test count: a drop indicates orphaned or missing tests.

5. **OS-specific build-tagged files silently dropped** — after moving `library_unix.go` and `library_windows.go`, verify they still apply with `go list -f '{{.GoFiles}} {{.IgnoredGoFiles}}' ./...` on all platforms and cross-compile checks (`GOOS=windows go build ./...`). Silent failure means platform-specific code is excluded with no compile error.

## Implications for Roadmap

Based on research, the refactor has a strict sequential dependency chain. Each phase must be green before the next begins.

### Phase 1: Layout Design and Decision Lock

**Rationale:** Three critical decisions must be made before touching any files — where `internal/` is anchored, what the implementation package name is, and whether the facade is pure-alias (zero-logic). All three are irreversible mid-migration without significant rework. Getting these wrong invalidates subsequent phases.

**Delivers:** A documented layout plan and a skeleton directory structure (empty packages, `package` declarations confirmed). Compile with no files moved yet.

**Addresses:** Table stakes — import path unchanged, `go.mod` stays at root, no circular imports by design.

**Avoids:** Pitfall 3 (`internal/` anchor at wrong level), Pitfall 8 (package name conflict), Pitfall 1 (type alias method limitation — decide facade strategy upfront).

### Phase 2: Core File Migration

**Rationale:** Move library loader first (leaf node, no Go dependencies), then runtime package. This order ensures `internal/runtime` can import `internal/library` as soon as runtime files are moved. Moving all FFI globals in a single atomic step prevents the duplicated-state window.

**Delivers:** All implementation Go files in `internal/library/` and `internal/runtime/`; tests co-located and converted to `package library` / `package runtime`; `go test ./internal/...` passes.

**Uses:** Standard `go test ./...`, `go test -race ./...` as phase exit gates.

**Avoids:** Pitfall 2 (duplicated FFI globals), Pitfall 4 (build-tagged files silently dropped), Pitfall 5 (tests left at root testing empty facade), Pitfall 13 (stale `//go:build` comments).

### Phase 3: Root Facade Creation

**Rationale:** The facade can only be written after both internal packages are stable. Writing it last avoids mid-refactor compilation failures in the facade caused by referencing incomplete packages.

**Delivers:** `facade.go` (and optionally `doc.go`) at repo root with all type aliases, function vars, const/var forwards. `go test ./...` from root passes, including existing test suite unchanged.

**Addresses:** All table-stakes features — complete public API surface preserved with identical signatures.

**Avoids:** Pitfall 3 (type definition vs. alias), Pitfall 2 (zero state at root).

### Phase 4: Build System and Lint Update

**Rationale:** CI and tooling validation after structural changes. Catches the golangci-lint path exclusion and stale `gci` prefix issues that only surface when files are in their new locations.

**Delivers:** Updated `.golangci.yml` (remove stale `gci` prefix, verify path exclusions), confirmed `make test`/`make test-release`/`make lint`/`make test-all` all pass, cross-compile check on all three GOOS values, `go mod tidy` no-op confirmed.

**Avoids:** Pitfall 6 (golangci-lint scope breaks), Pitfall 7 (CI discovers zero tests), Pitfall 11 (`goimports` misclassifies internal paths), Pitfall 12 (`go.sum` drift).

### Phase 5: Compatibility Verification and Docs

**Rationale:** Final gate before PR. API surface comparison against v0.3.4 baseline, documentation updates, and example verification. This phase is low-risk but catches any symbol accidentally omitted from the facade.

**Delivers:** `compat_test.go` at root (compile-time surface gate), `go-apidiff v0.3.4 HEAD` output showing zero incompatible changes, updated `CLAUDE.md` file paths, updated `GO_API_SURFACE.md`, confirmed `examples/go/basic/main.go` compiles unchanged.

**Avoids:** Pitfall 9 (`pkg.go.dev` internal path exposure — evaluate and document).

### Phase Ordering Rationale

- Library before runtime: eliminates the risk of runtime referencing a package that doesn't yet exist.
- Internal packages before facade: facade compilation depends on both internal packages being stable; writing it last avoids cascading failures.
- Tooling update after migration: lint and CI configuration changes are meaningful only once files are in their final locations — updating them before the move generates false baselines.
- Compatibility verification last: `go-apidiff` compares against the v0.3.4 tag; the comparison is only meaningful when the facade is complete.

### Research Flags

Phases with standard, well-documented patterns (no additional research needed):
- **Phase 1:** Go module layout and `internal/` rules are fully specified in official docs; research is complete.
- **Phase 2:** File migration and package renaming are mechanical; no novel patterns.
- **Phase 3:** Type alias and function var facades are documented in official Go blog posts; patterns are explicit.
- **Phase 4:** golangci-lint config updates are routine; stale `gci` prefix fix is documented in research.
- **Phase 5:** `go-apidiff` usage is documented; `compat_test.go` pattern is established.

No phases require `/gsd:research-phase` — the research coverage is comprehensive and sourced from official specifications. The one area that may need a validation step (not research) is `pkg.go.dev` rendering of type aliases pointing to `internal/` paths, which requires a staging branch to observe.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All Go toolchain features verified against official release notes and specs; no external dependencies added |
| Features | HIGH | Feature list derived directly from codebase inspection and official Go compatibility rules; no ambiguity |
| Architecture | HIGH | Facade pattern and internal package rules are governed by Go spec; layout validated against official internal package documentation |
| Pitfalls | HIGH | Critical pitfalls are compile-time enforced (internal visibility, circular imports, type alias rules); moderate pitfalls verified via official issues and docs |

**Overall confidence:** HIGH

### Gaps to Address

- **`pkg.go.dev` type alias rendering**: aliases pointing to `internal/` packages may expose implementation paths in generated docs. The research identified this as a moderate pitfall but cannot prescribe the exact rendering without a live staging branch. Evaluate post-migration on a staging branch before merging; if unacceptable, switch exported types to wrapper structs (API-breaking, must be weighed deliberately).

- **Known finalizer/Close() race in CONCERNS.md**: the existing documented race must not regress. The migration moves `ffiMu` to `internal/runtime`; `go test -race ./...` gates at every phase protect against this, but the pre-existing condition should be explicitly re-verified in Phase 5.

- **`gci` `prefix()` stale value in `.golangci.yml`**: confirmed as a leftover from another project (`github.com/chaoslabs-bg/tclr-v2/`). Fix this in Phase 4 to `github.com/amikos-tech/chroma-go-local/`. Not a blocker for migration but produces incorrect import grouping in moved files.

## Sources

### Primary (HIGH confidence)
- [Organizing a Go module — go.dev/doc/modules/layout](https://go.dev/doc/modules/layout) — module layout, single-module pattern
- [Go Modules Reference — go.dev/ref/mod](https://go.dev/ref/mod) — internal package visibility rules
- [What's in an (Alias) Name? — go.dev/blog/alias-names](https://go.dev/blog/alias-names) — type alias semantics and method inheritance
- [Codebase Refactoring with Go — go.dev/talks/2016/refactor.article](https://go.dev/talks/2016/refactor.article) — forwarding pattern for large-scale moves
- [Go Blog: Keeping Your Modules Compatible](https://go.dev/blog/module-compatibility) — API compatibility rules
- [Go Blog: Backward Compatibility, Go 1.21, and Go 2](https://go.dev/blog/compat) — version compatibility guarantees
- [golang.org/x/exp/apidiff — pkg.go.dev](https://pkg.go.dev/golang.org/x/exp/apidiff) — API diff tooling
- [Go internal package spec — pkg.go.dev/cmd/go](https://pkg.go.dev/cmd/go#hdr-Internal_Directories) — internal package import rule
- [Go Issue #23042: methods with type alias receivers](https://github.com/golang/go/issues/23042) — type alias method limitation confirmed
- [Go Issue #12217: internal package importable outside subtree](https://github.com/golang/go/issues/12217) — internal visibility edge case
- [Go 1.24 Release Notes](https://go.dev/doc/go1.24) — generic type aliases (not needed for this project)
- Codebase direct inspection: `go.mod`, `chroma.go`, `config.go`, `embedded.go`, `errors.go`, `library.go`, `library_unix.go`, `library_windows.go`, `backup.go`, `rebuild.go`, `compaction.go`, `wal_prune.go`, `ci.yml`, `Makefile`, `.golangci.yml`

### Secondary (MEDIUM confidence)
- [joelanford/go-apidiff — github.com](https://github.com/joelanford/go-apidiff) — API diff CLI wrapper; actively maintained community tool
- [golangci-lint path handling issue #3717](https://github.com/golangci/golangci-lint/issues/3717) — known inconsistency in path exclusion matching
- [Go Modules Guide for Monorepos — Grab Engineering](https://engineering.grab.com/go-module-a-guide-for-monorepos-part-1) — single-module monorepo patterns

### Tertiary
- [Use internal packages to reduce your public API surface — Dave Cheney](https://dave.cheney.net/2019/10/06/use-internal-packages-to-reduce-your-public-api-surface) — internal package best practices

---
*Research completed: 2026-03-20*
*Ready for roadmap: yes*
