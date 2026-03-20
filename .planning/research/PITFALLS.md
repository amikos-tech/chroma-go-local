# Domain Pitfalls: Go Module Subtree Reorganization

**Domain:** Go module reorganization — moving implementation to subtree while preserving root import path
**Researched:** 2026-03-20
**Confidence:** HIGH (Go spec + official docs + verified patterns)

---

## Critical Pitfalls

Mistakes that cause build failures, broken imports, or silent behavioral regressions.

---

### Pitfall 1: Type Aliases Cannot Have Methods Attached at the Facade Layer

**What goes wrong:**
The root facade re-exports types from `go/internal/...` using Go type aliases (`type Server = impl.Server`). This looks correct and compiles, but you cannot add or attach any new methods to a type alias whose base type is defined in a different package. The Go spec prohibits method declarations when the receiver base type is not defined in the same package. If any adapter logic is needed at the root layer — even a convenience method — it silently cannot be expressed as a method on the aliased type.

**Why it happens:**
Go's method-receiver rule: "the receiver base type must be defined in the same package as the method." Type aliases are transparent identity, not new types. There is no mechanism to extend them with methods from the declaring package.

**Consequences:**
- If the facade needs to wrap or adapt behavior beyond pure delegation, the entire type-alias approach must be abandoned and replaced with embedding or wrapper structs — a rewrite.
- Embedding in a wrapper struct is API-breaking (users get a different concrete type, not the original).

**Prevention:**
Decide upfront: the root facade must be 100% identity-transparent (aliases only, zero logic). If any root-layer adaptation is ever needed, use wrapper structs from the start and document that they are wrappers. For this project, pure type aliases work as long as the root package is truly a thin pass-through.

**Detection:**
A compile error `"cannot define new methods on non-local type"` appears if you try. Catch this by auditing all exported types in the root facade before committing to the alias strategy.

**Phase:** Phase 1 (facade creation). Decide at design time, not mid-implementation.

---

### Pitfall 2: Package-Level State Splits Across Two Packages — The Global `libOnce` / `ffiMu` Problem

**What goes wrong:**
The current implementation holds critical global state at the package level in `package chroma` (root): `libHandle`, `libOnce`, `libErr`, `ffiMu`, and all `~40` FFI function variables. If implementation is moved to `go/internal/runtime` as a new package, and the root becomes `package chroma` that delegates, there are now **two separate package namespaces** for this global state. Any code that imports the root package (`package chroma`) and the internal package simultaneously could see different state. Worse: `sync.Once` in the implementation package fires independently from anything in the facade package.

**Why it happens:**
Go package-level variables are isolated per package. Moving the globals to the internal package and re-exporting `Init()` from the root means the root `Init()` is just a call forwarder. This is actually fine — **but only if the root package has zero duplicated globals**. The trap is accidentally leaving any state in the root package that mirrors state in the implementation package.

**Consequences:**
- Double-initialization: if the root wraps `Init()` and also initializes something at root scope, `libOnce.Do()` fires once per package, not once globally.
- Test confusion: tests in `package chroma` (root) versus `package chroma_test` (external) may observe different initialization states.

**Prevention:**
All global state — `libHandle`, `libOnce`, `libErr`, `ffiMu`, FFI function pointers — must live entirely in the implementation package. The root facade must contain zero state. Verify with `go vet` and a race detector run (`go test -race ./...`) after every state-migration step.

**Detection:**
- `go test -race` will report races if state is duplicated.
- Search for `var` declarations at root package scope after migration; there must be none except type aliases.

**Phase:** Phase 1 (core move). Highest risk step.

---

### Pitfall 3: `internal` Package Visibility Cuts Off the Root Facade

**What goes wrong:**
The proposed layout uses `go/internal/runtime/` and similar paths. Go's `internal` rule is: packages under `.../a/b/internal/...` can only be imported by code rooted at `.../a/b/`. If the module root is `github.com/amikos-tech/chroma-go-local` and internal packages are at `go/internal/runtime`, the import path is `github.com/amikos-tech/chroma-go-local/go/internal/runtime`. The root facade (at module root, package `chroma`) is **outside** the `go/internal/` subtree — meaning it cannot import those packages.

**Why it happens:**
The `internal` visibility rule is relative to the *directory* containing `internal`, not the module root. A package at `./go/internal/runtime` is only importable by packages inside `./go/`. The root facade at `.` (module root) is outside `./go/`, so the import is illegal.

**Consequences:**
Compile error: `"use of internal package github.com/amikos-tech/chroma-go-local/go/internal/runtime not allowed"` when the root facade tries to import the implementation.

**Prevention:**
Two valid structural options:
1. Place implementation under `go/runtime/` (no `internal`) — keeps it importable from the root, but exposes it publicly (acceptable since the module boundary already restricts external use of what matters).
2. Place implementation under `internal/` at the **module root** (`./internal/runtime/`), not under `go/`. Then any package in the module can import it, including the root facade.

Option 2 is safer and more idiomatic. The `go/` directory still exists for organizing source files, but the `internal/` anchor point is at the module root.

**Detection:**
This compile error appears immediately on first `go build ./...` with the wrong layout. Validate the layout plan against the `internal` rule before writing any code.

**Phase:** Phase 1 (layout design). Must be resolved before any files are moved.

---

### Pitfall 4: Build-Tagged Files (`library_unix.go`, `library_windows.go`) Silently Stop Applying

**What goes wrong:**
The current implementation has `library_unix.go` and `library_windows.go` with OS-specific logic. These files rely on filename-based build constraints (`_unix`, `_windows` suffixes) or explicit `//go:build` tags. When moved to a subdirectory, the tags still apply — but the `go test ./...` pattern run from the repo root will pick them up correctly only if the working directory context and GOPATH/module resolution remain unchanged. The silent failure mode: if a moved file lands in the wrong package directory and a build tag stops matching (e.g., due to a typo in the new path), the file is excluded without error, and platform-specific code silently falls back to nothing.

**Why it happens:**
Go build tag evaluation is lexical. Filename-suffix tags (`_windows.go`) apply based on the filename alone, regardless of directory. If the file is renamed or placed in a package where the filename convention no longer signals the tag correctly, the constraint silently drops.

**Consequences:**
- Platform-specific library loading code is skipped on the target platform.
- The program compiles and links but fails at runtime when `Init()` cannot find the library on Linux or Windows.
- No compile error. CI on Linux may pass while Windows CI fails silently.

**Prevention:**
After moving `library_unix.go` and `library_windows.go`, run `go list -f '{{.GoFiles}}' ./go/runtime/` on all three platforms (or cross-compile with `GOOS=windows go build ./...`). Confirm the OS-specific files appear in the list. Add a CI smoke test that verifies platform detection.

**Detection:**
- `go list -f '{{.GoFiles}} {{.IgnoredGoFiles}}' <package>` shows which files are excluded.
- Cross-compile gate in CI: `GOOS=linux go build ./...` and `GOOS=windows go build ./...` from a Mac runner.

**Phase:** Phase 2 (file migration). Verify immediately after move, before merging.

---

### Pitfall 5: `go test -v ./...` and Makefile `go test` Commands Break After Subtree Move

**What goes wrong:**
The Makefile currently runs `go test -v ./...` from the module root, with `CHROMA_LIB_PATH` injected. The CI workflow also runs `go test -v ./...` from the workspace root. After moving all `.go` files out of the root directory, the root package (`package chroma`) still exists as a facade — but it now contains only type aliases and thin wrappers with no test files. The test runner pattern `./...` still works, but any test that was in `package chroma` (white-box, same-package tests) no longer has access to the unexported symbols they were testing, because those symbols now live in the implementation package.

**Why it happens:**
White-box tests (using `package chroma` not `package chroma_test`) test unexported functions and variables. After the move, those unexported symbols are in `go/internal/runtime` (or wherever implementation lives), not in the root `package chroma`. Moving the test files alongside the implementation is required — but if done wrong, tests end up in a package that has a different import path from what the facade exposes.

**Consequences:**
- Test files left at the root with `package chroma` now test an empty facade — they compile but test nothing meaningful.
- Test files moved to the implementation package work, but they use `package chroma_internal_test` or similar, which is a different package identity.
- golangci-lint path exclusion rules (`path: third_party$`, `path: examples$`) continue to work, but any path-relative lint rules silently stop applying to moved files.

**Prevention:**
- Move all test files alongside the implementation files they test, maintaining the same `package` declaration.
- After the move, add a root-level `api_compat_test.go` in `package chroma_test` that imports the root facade and exercises the full public API — this is the compatibility gate.
- Verify `go test -v ./...` output still shows test counts across all subpackages after the move.

**Detection:**
- Compare `go test -v ./...` output before and after migration: test count per package should remain equal or increase.
- A sudden drop in test count is a red flag.

**Phase:** Phase 3 (test reorganization). Explicit phase dedicated to this.

---

### Pitfall 6: golangci-lint Scope and Path Rules Break Silently

**What goes wrong:**
The current `.golangci.yml` has `paths:` exclusions (`third_party$`, `builtin$`, `examples$`) and a `gci` formatter configured with `prefix(github.com/chaoslabs-bg/tclr-v2/)` (which looks like a leftover from another project). After moving Go files to `go/runtime/` or similar, the lint invocation `golangci-lint run ./...` picks up the new paths. However, path-relative exclusion rules in golangci-lint v2 are matched textually relative to the config file location. If the config remains at the root and files move to subdirectories, exclusion rules that matched file-relative paths must be reviewed.

The `gci` import ordering config has a stale `prefix(github.com/chaoslabs-bg/tclr-v2/)` section — this prefix matches nothing in this repo and is dead config, but after the move introduces new import paths in the subtree, import ordering may silently differ from expectations.

**Why it happens:**
golangci-lint path matching uses textual rules, not package-aware matching. Moving files changes their relative paths, potentially matching or not matching existing exclusion rules.

**Consequences:**
- New false positives or silently suppressed true positives in linting.
- Import ordering violations in moved files if `gci` config doesn't reflect actual import groupings.

**Prevention:**
- Audit `.golangci.yml` path exclusions after migration; re-verify all `paths:` rules still apply to the intended files.
- Remove the dead `prefix(github.com/chaoslabs-bg/tclr-v2/)` from `gci` settings.
- Run `golangci-lint run ./...` after migration and treat any new findings as real issues, not noise.

**Detection:**
- Compare `golangci-lint run ./...` output before and after move. New warnings = exclusion rule miss.
- Review any `path:` patterns in `.golangci.yml` against the new directory tree.

**Phase:** Phase 4 (build system update). Do not defer this; stale lint config accumulates.

---

### Pitfall 7: CI `go-version-file: go.mod` Still Resolves Correctly — But `working-directory` Overrides Are Missed

**What goes wrong:**
The CI workflow uses `go-version-file: go.mod` in `actions/setup-go`, resolved relative to the repository root. The `go.mod` stays at the root (correct — the module does not move). However, CI steps that run `go test -v ./...` use `$env:GITHUB_WORKSPACE` as the working directory. After migration, if any CI step hardcodes the old path structure (e.g., a path assertion about `chroma.go` being at repo root, or a `working-directory: .` that becomes meaningless), those steps silently pass or fail in unexpected ways.

The specific risk: the CI PowerShell step constructs `CHROMA_LIB_PATH` by joining `$env:GITHUB_WORKSPACE "shim/target/debug/$libName"`. This path is correct and does not change. But `go test -v ./...` runs from the workspace root — after migration, if the root package becomes a pure facade with no test files, the test step must still discover tests in subpackages via `./...`. Confirm this explicitly.

**Why it happens:**
CI steps that pass today are only tested by running them. Moving files without running CI on a branch is how regressions slip through.

**Prevention:**
- Run the full CI matrix (all three OS runners) on a migration branch before merging.
- Add an explicit `go list ./...` step in CI after migration that prints all packages; visually verify coverage.

**Detection:**
A CI run with zero test output (all packages skipped) is the failure mode. Monitor per-package test counts across OS matrix runs.

**Phase:** Phase 4 (CI update). Verify on a live CI run.

---

### Pitfall 8: Package Name Constraint — Root Must Remain `package chroma`

**What goes wrong:**
The root directory's `package` declaration must remain `package chroma` (matching the directory base name convention users expect when they `import "github.com/amikos-tech/chroma-go-local"`). If any facade file at the root accidentally declares `package main` or `package runtime`, the module's root package breaks and `go build` errors on the import.

More subtle: if the implementation subtree declares `package chroma` as well (because the developer wants the same public package name), Go will reject this — two different directories cannot share a package if they're in different directory paths. The implementation package **must** use a different package name (e.g., `package runtime` or `package impl`).

**Why it happens:**
In Go, package name and import path are orthogonal. The root at `github.com/amikos-tech/chroma-go-local` is `package chroma`. The implementation at `github.com/amikos-tech/chroma-go-local/go/runtime` must be a different package — `package runtime`. The root facade imports it and re-exports.

**Consequences:**
- If implementation uses `package chroma`, the facade cannot import it (circular or duplicate package identity issues).
- Users who access internal types directly will see an unexpected package name in error messages and IDE auto-completion.

**Prevention:**
Name the implementation package something distinct from `chroma` (e.g., `package chromaImpl` or `package runtime` within a `go/internal/runtime` directory). The public API surface stays as `chroma.*` through aliases and wrappers at the root.

**Detection:**
`go build ./...` will immediately surface package name conflicts. Check with `go list -f '{{.Name}} {{.ImportPath}}' ./...`.

**Phase:** Phase 1 (layout design). Decided upfront.

---

## Moderate Pitfalls

### Pitfall 9: `go doc` and `pkg.go.dev` Display Degrades for Aliased Types

**What goes wrong:**
pkg.go.dev and `go doc` render type aliases differently from locally defined types. A type alias `type Server = impl.Server` at the root shows up in documentation as "type Server = impl.Server" with a link to the implementation package — exposing the internal package path to users. This is ugly but not a functional breakage. However, if the implementation package is under `internal/`, pkg.go.dev will not render the internal package's documentation at all, so users see a type alias pointing to an unexported path. The documentation experience degrades.

**Prevention:**
Evaluate whether `pkg.go.dev` rendering is acceptable after migration using a staging branch. If internal package paths surface in docs, use concrete wrapper types instead of aliases for exported types, accepting the higher implementation cost.

**Phase:** Phase 1 (facade design decision) and Phase 5 (docs review).

---

### Pitfall 10: `go test -race` Reveals Hidden Concurrency Bugs During Package Restructuring

**What goes wrong:**
The migration moves `ffiMu` and FFI globals into the implementation package. During the transition (partial move), tests may run against a mixed state: some functions calling into the implementation package's `ffiMu`, others still referencing the root-level copy. The race detector catches this only if concurrent tests exercise both paths.

**Prevention:**
Run `go test -race ./...` at every phase checkpoint — before, during, and after the move. The existing known race (finalizer vs. explicit `Close()`) documented in CONCERNS.md must not regress.

**Phase:** Every phase. Treat as a continuous gate, not a one-time check.

---

### Pitfall 11: `make fmt` Breaks if `goimports` and `gofmt` Paths Are Root-Relative

**What goes wrong:**
`make fmt-go` runs `gofmt -w .` and `goimports -w .` from the project root. After migration, these commands still apply recursively to all `.go` files in the module (including the new subtree). This is fine. The subtle issue: `goimports` reorders imports using the module path as the "local package" prefix. If `goimports` doesn't pick up the correct module path after files move, it may treat `github.com/amikos-tech/chroma-go-local/go/internal/runtime` as a third-party import rather than a local one, placing it in the wrong import group in files that import it.

**Prevention:**
After migration, run `make fmt-go` and check `git diff` for unexpected import reorderings. If `goimports` misclassifies internal paths, pass `-local github.com/amikos-tech/chroma-go-local` explicitly to `goimports`.

**Phase:** Phase 4 (build system update).

---

## Minor Pitfalls

### Pitfall 12: `go.sum` Stays Correct — But `go mod tidy` After Adding No New Dependencies Should Be a No-Op

**What goes wrong:**
This refactor adds no new external dependencies. `go mod tidy` after the migration should produce zero changes to `go.mod` and `go.sum`. If it does change, something unexpected was added or a dependency path changed. This is a sanity check, not a likely failure.

**Prevention:**
Run `go mod tidy && git diff go.mod go.sum` as a migration phase exit criterion.

**Phase:** Phase 1 and Phase 4. Quick check.

---

### Pitfall 13: Stale `//go:build` Comments on Moved Files

**What goes wrong:**
If any file has a manual `//go:build` comment that references a tag that no longer applies (e.g., a file that was excluded from a certain build now becomes reachable or unreachable due to directory changes), the build silently includes or excludes it incorrectly.

**Prevention:**
After migration, run `go list -f '{{.GoFiles}} {{.IgnoredGoFiles}}' ./...` and verify every file appears in the expected list.

**Phase:** Phase 2 (file migration). Part of the post-move verification step.

---

## Phase-Specific Warning Table

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Layout design (Phase 1) | `internal/` anchor at wrong level blocks root facade import | Place `internal/` at module root, not under `go/` |
| Layout design (Phase 1) | Type alias method limitation | Confirm root is zero-logic before committing to alias strategy |
| Layout design (Phase 1) | Implementation package name conflicts with `package chroma` | Choose a distinct package name for implementation subtree |
| Core file move (Phase 2) | FFI globals duplicated across packages, causing double-init | Move ALL state; root has zero `var` declarations |
| Core file move (Phase 2) | OS-specific build-tagged files silently dropped | Cross-compile check: `GOOS=linux/windows/darwin go build ./...` |
| Test reorganization (Phase 3) | White-box tests left at root testing an empty facade | Move test files alongside implementation; add compatibility gate test |
| Build system update (Phase 4) | `golangci-lint` path exclusions and stale `gci` prefix config | Audit `.golangci.yml` after move; remove dead `gci` prefix |
| Build system update (Phase 4) | `goimports` misclassifies internal paths as third-party | Pass `-local github.com/amikos-tech/chroma-go-local` to `goimports` |
| CI update (Phase 4) | CI passes with zero tests discovered | Monitor per-package test count across OS matrix |
| Docs review (Phase 5) | `pkg.go.dev` exposes internal paths via type alias rendering | Validate rendered docs on staging branch |

---

## Sources

- [Go Specification: Internal Packages](https://go.dev/ref/mod#internal-packages) — HIGH confidence
- [Organizing a Go module — official layout guidance](https://go.dev/doc/modules/layout) — HIGH confidence
- [Go Blog: Keeping Your Modules Compatible](https://go.dev/blog/module-compatibility) — HIGH confidence
- [Go Blog: What's in an (Alias) Name?](https://go.dev/blog/alias-names) — HIGH confidence, type alias semantics
- [Go Issue #23042: methods with type alias receivers](https://github.com/golang/go/issues/23042) — HIGH confidence, confirmed limitation
- [Go Issue #12217: internal package importable outside subtree](https://github.com/golang/go/issues/12217) — HIGH confidence, confirmed rule
- [golangci-lint: relative path handling issue #3717](https://github.com/golangci/golangci-lint/issues/3717) — MEDIUM confidence, known inconsistency
- [Go Wiki: Target-Specific Code (build tags)](https://go.dev/wiki/TargetSpecific) — HIGH confidence
- [Go Package Initialization Order](https://yourbasic.org/golang/package-init-function-main-execution-order/) — HIGH confidence
