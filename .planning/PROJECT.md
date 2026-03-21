# chroma-go-local v0.4.0: Go Subtree Reorganization

## What This Is

A structural refactor of the chroma-go-local repository that moves Go implementation code out of the repo root into a dedicated `go/` subtree. This preserves the current public import path (`github.com/amikos-tech/chroma-go-local`) and all API behavior while making Go/Java/Rust ownership boundaries explicit for long-term maintenance.

## Core Value

The public Go import path and API surface must remain 100% backward-compatible — existing users must not need to change any code.

## Requirements

### Validated

- ✓ Pure Go FFI via purego (no cgo) — existing
- ✓ Server mode (HTTP) and Embedded mode (in-process) — existing
- ✓ Builder pattern configuration with YAML backing — existing
- ✓ Explicit resource lifecycle with finalizer fallback — existing
- ✓ Java scaffold bindings (JNA + Panama) — existing
- ✓ Rust FFI shim with C-compatible exports — existing
- ✓ Cross-platform support (Linux, macOS, Windows) — existing
- ✓ WAL prune maintenance APIs — v0.4.0 (#26)
- ✓ Collection rebuild maintenance API — v0.4.0 (#25)

### Active

- ✓ Add thin root facade to preserve import compatibility — Validated in Phase 3
- ✓ Reorganize tests for subtree layout + API compatibility coverage — Validated in Phase 4
- ✓ Update Make/CI for relocated Go package layout — Validated in Phase 4
- [ ] Refresh docs/examples after Go subtree move
- [ ] Add compatibility gate checklist for root cleanup refactor

### Validated in Phase 2: File Migration

- ✓ Move Go implementation into `internal/` subtree without breaking imports — all files in `internal/runtime/` and `internal/library/`

### Out of Scope

- New features or API additions — this is purely structural
- Java binding changes — layout remains unchanged
- Rust shim changes — shim stays in `shim/`
- Go module path change — must remain `github.com/amikos-tech/chroma-go-local`

## Context

The current repo has Go implementation files (chroma.go, config.go, embedded.go, errors.go, library.go, backup.go, rebuild.go, compaction.go, wal_prune.go) mixed at the repo root alongside Rust and Java code. This makes long-term maintenance and Java binding expansion harder. The v0.4.0 milestone moves Go implementation into a clean subtree while the root package becomes a thin facade.

GitHub milestone: v0.4.0 (amikos-tech/chroma-go-local milestone #4)
Umbrella issue: #45
Individual issues: #39, #40, #41, #42, #43, #44

## Constraints

- **Import compatibility**: Root module path `github.com/amikos-tech/chroma-go-local` must remain valid
- **API stability**: No public API signature changes allowed
- **Build system**: `make test`, `make lint`, `make test-all` must pass throughout
- **CI**: GitHub Actions workflows must remain green across OS matrix
- **No circular deps**: New package layout must not introduce circular dependencies

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Sequential phases (one issue per phase) | Refactoring has strict dependency chain; each step must be stable before next | — Pending |
| Subtree under `go/` directory | Keeps Go/Java/Rust separation explicit at repo root level | — Pending |
| Root becomes thin facade | Preserves import path without requiring Go module replace directives | — Pending |
| Proposed layout: `go/internal/runtime/`, `go/internal/library/`, etc. | Internal packages prevent unintended external imports | — Pending |

## Current State

- Phase 1 (Layout Design) complete — skeleton packages created
- Phase 2 (File Migration) complete — all Go implementation (8 files) and test files (11 files) moved to `internal/runtime/`, library FFI loading (4 files) moved to `internal/library/`. Zero .go files remain at repo root. Race-instrumented compilation and cross-compile pass.
- Phase 3 (Root Facade) complete — 9 facade files at repo root (doc.go, chroma.go, config.go, errors.go, embedded.go, backup.go, rebuild.go, compaction.go, wal_prune.go) re-export 54 type aliases and ~37 wrapper functions from internal/runtime. Full public API surface restored. `go build ./...` passes including examples.
- Phase 4 (Build and Test) complete — lint clean (gci prefix fixed, SA1019 suppressed), `compat_test.go` created with 110-symbol compile gate + 9 behavioral smoke tests, all make targets pass, cross-compile verified for 3 OS.

---
*Last updated: 2026-03-21 after Phase 4 completion*
