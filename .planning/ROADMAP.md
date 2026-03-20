# Roadmap: chroma-go-local v0.4.0 — Go Subtree Reorganization

## Overview

A strictly sequential structural refactor: move all Go implementation files from the repo root into `internal/runtime/` and `internal/library/` subtrees, then restore the root package as a pure-forwarding facade. Each phase must be green before the next begins — the dependency chain is irreversible. The public import path `github.com/amikos-tech/chroma-go-local` and every exported symbol must be identical from a caller's perspective before and after.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Layout Design** - Lock directory structure, create skeleton packages, confirm `internal/` is anchored at module root
- [ ] **Phase 2: File Migration** - Move all Go implementation files into `internal/runtime/` and `internal/library/`, co-locate tests
- [ ] **Phase 3: Root Facade** - Write thin facade at repo root that re-exports every public symbol via type aliases and function vars
- [ ] **Phase 4: Build and Test** - Update Makefile, CI, lint config, and test layout; verify all `make` targets and cross-compile pass
- [ ] **Phase 5: Compatibility and Docs** - Run `go-apidiff` against v0.3.4, add `compat_test.go`, update CLAUDE.md and docs

## Phase Details

### Phase 1: Layout Design
**Goal**: The target directory structure exists as compilable skeleton packages; the `internal/` anchor is confirmed at module root; no files have moved yet
**Depends on**: Nothing (first phase)
**Requirements**: LAYOUT-01, LAYOUT-02
**Success Criteria** (what must be TRUE):
  1. `internal/runtime/` and `internal/library/` directories exist at the module root with valid `package` declarations
  2. `go build ./...` succeeds with the skeleton in place alongside existing root files
  3. The root facade import of `internal/runtime` compiles without "use of internal package not allowed" errors
  4. No existing Go files have been relocated; the skeleton is additive only
**Plans:** 1 plan
Plans:
- [x] 01-01-PLAN.md — Create skeleton packages and anchor validation test

### Phase 2: File Migration
**Goal**: All Go implementation files live in `internal/library/` and `internal/runtime/`; the repo root contains no implementation logic; `go test ./internal/...` passes with race detector on
**Depends on**: Phase 1
**Requirements**: LAYOUT-03, LAYOUT-04
**Success Criteria** (what must be TRUE):
  1. All FFI globals (`libHandle`, `libOnce`, `ffiMu`, and all function pointers) exist exclusively in `internal/runtime`; the module root has zero `var` state declarations
  2. Platform-specific files (`library_unix.go`, `library_windows.go`) are present in `internal/library/` with correct build tags verified via `go list`
  3. `go test -race ./internal/...` passes with no data-race reports
  4. Per-package test count in `internal/...` accounts for all tests that previously ran at root; no tests are orphaned
**Plans:** 2 plans
Plans:
- [x] 02-01-PLAN.md — Move library files to internal/library/, export LoadLibrary, bridge chroma.go Init()
- [x] 02-02-PLAN.md — Move runtime files and tests to internal/runtime/, fix imports, remove anchor test

### Phase 3: Root Facade
**Goal**: The repo root package re-exports every previously public symbol so that existing callers compile and behave identically without changing a single import statement
**Depends on**: Phase 2
**Requirements**: FACADE-01, FACADE-02, FACADE-03, FACADE-04, FACADE-05
**Success Criteria** (what must be TRUE):
  1. Every exported type is re-exported via `type X = internal/runtime.X` (alias, not definition); `chroma.Server` and `runtime.Server` are the same type
  2. Every exported function is delegated via `var F = runtime.F`; callers invoking `chroma.NewServer(...)` reach the implementation without wrapping overhead
  3. Every exported constant, sentinel error, and package-level variable is forwarded at root; the root file contains zero logic, zero `var` state, and zero `init()` functions
  4. `go test ./...` from the module root passes using the existing test suite with no changes to test files
  5. The import path `github.com/amikos-tech/chroma-go-local` resolves to the facade package and the package name remains `chroma`
**Plans:** 2 plans
Plans:
- [x] 03-01-PLAN.md — Create core facade files (doc.go, chroma.go, config.go, errors.go)
- [x] 03-02-PLAN.md — Create feature facade files (embedded.go, backup.go, rebuild.go, compaction.go, wal_prune.go)

### Phase 4: Build and Test
**Goal**: All `make` targets pass, the CI matrix stays green, lint is clean, and the test layout reflects the new package structure including a root-level compatibility gate
**Depends on**: Phase 3
**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04, BUILD-01, BUILD-02, BUILD-03, BUILD-04
**Success Criteria** (what must be TRUE):
  1. `make test`, `make test-release`, `make lint`, and `make test-all` all pass locally without modification by the caller
  2. `golangci-lint run ./...` reports zero issues; the stale `gci` prefix in `.golangci.yml` is corrected to `github.com/amikos-tech/chroma-go-local/`
  3. Cross-compile succeeds: `GOOS=linux go build ./...`, `GOOS=darwin go build ./...`, `GOOS=windows go build ./...` all complete without error
  4. `compat_test.go` exists at repo root in `package chroma_test` and references every exported symbol; it compiles successfully as a compile-time API surface gate
  5. CI workflows (`.github/workflows/ci.yml`) run without modification to the OS matrix or job structure
**Plans**: TBD

### Phase 5: Compatibility and Docs
**Goal**: Machine-verified API compatibility against the v0.3.4 baseline is confirmed, documentation reflects the new layout, and the release is ready to merge
**Depends on**: Phase 4
**Requirements**: DOCS-01, DOCS-02, DOCS-03, DOCS-04, COMPAT-01, COMPAT-02, COMPAT-03
**Success Criteria** (what must be TRUE):
  1. `go-apidiff` run against the v0.3.4 tag reports zero incompatible changes
  2. README.md and CLAUDE.md accurately describe the new `internal/runtime/` and `internal/library/` layout; no file paths point to the old root-level implementation files
  3. GO_API_SURFACE.md references are updated to reflect new file locations
  4. A compatibility checklist is completed and attached to the PR confirming: import path unchanged, no public API removals, no type definition breakages, race detector clean
  5. Release notes include a refactor summary and an explicit compatibility statement for existing users
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in strict sequential order: 1 → 2 → 3 → 4 → 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Layout Design | 1/1 | Complete | 2026-03-20 |
| 2. File Migration | 2/2 | Complete | 2026-03-20 |
| 3. Root Facade | 0/2 | Not started | - |
| 4. Build and Test | 0/TBD | Not started | - |
| 5. Compatibility and Docs | 0/TBD | Not started | - |
