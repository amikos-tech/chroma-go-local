# Phase 2: File Migration - Context

**Gathered:** 2026-03-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Move all Go implementation files from the repo root into `internal/runtime/` and `internal/library/`, co-locate tests alongside their implementation, and remove the Phase 1 anchor test. After this phase the repo root contains no Go implementation logic. `go test ./internal/...` must pass with race detector.

</domain>

<decisions>
## Implementation Decisions

### Internal API boundary
- `internal/library` exports exactly one symbol: `LoadLibrary(path string) (uintptr, error)`
- All path resolution helpers, types (`libraryCandidate`, `libraryLoadPlan`, `candidateSet`), and utility functions remain unexported
- `internal/runtime` calls `library.LoadLibrary()` from `Init()`, stores the returned handle, then registers FFI functions
- Dependency direction: runtime → library (library is a leaf package with zero upstream imports)

### File-to-package mapping (carried from Phase 1)
- `internal/library/`: library.go, library_unix.go, library_windows.go, library_test.go
- `internal/runtime/`: chroma.go, config.go, embedded.go, errors.go, backup.go, rebuild.go, compaction.go, wal_prune.go + all remaining test files
- Maintenance ops (backup, rebuild, compaction, wal_prune) stay in runtime — methods on Server/Embedded receiver types must be in the same package

### Test placement
- All tests move with their implementation code: library_test.go → internal/library/, all others → internal/runtime/
- Tests become `package library` and `package runtime` respectively (white-box, same-package access preserved)
- No tests remain at root after this phase
- `internal_test.go` (Phase 1 anchor test) removed — real code in internal/ is a stronger proof

### Move atomicity
- Two commits, each independently compilable:
  1. Move library package: library.go + library_unix.go + library_windows.go + library_test.go → internal/library/ (root still has all runtime files, go build passes)
  2. Move runtime package: all remaining .go files + tests → internal/runtime/, add `import library` in runtime, remove internal_test.go (root is empty of impl, go build passes)

### Package-level init
- Keep existing Init() + sync.Once + package-level globals pattern exactly as-is, just relocated to internal/runtime
- `Init()` calls `library.LoadLibrary(libPath)` instead of the former same-package `loadLibrary()`
- All 40+ FFI function pointers, libHandle, libOnce, libErr, ffiMu remain as package-level vars in internal/runtime
- Zero behavioral change from the caller's perspective

### Claude's Discretion
- Exact ordering of files within each commit (as long as the two-commit structure is preserved)
- Whether to rename `loadLibrary` → `LoadLibrary` in library_unix.go/library_windows.go or create a new exported wrapper
- Minor import path adjustments needed after the package rename

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project structure
- `.planning/ROADMAP.md` — Phase 2 goal, success criteria (LAYOUT-03, LAYOUT-04), dependency chain
- `.planning/REQUIREMENTS.md` — LAYOUT-03 (FFI globals moved atomically) and LAYOUT-04 (platform build tags retained)
- `.planning/PROJECT.md` — Core value (backward-compatible import path), constraints, key decisions

### Phase 1 context
- `.planning/phases/01-layout-design/01-CONTEXT.md` — File-to-package mapping decisions, package naming, skeleton depth

### Existing codebase
- `go.mod` — Module path `github.com/amikos-tech/chroma-go-local`; confirms internal/ anchor point
- `CLAUDE.md` — Architecture diagram, build commands, key patterns
- `chroma.go` — FFI globals block (lines 15-67), Init() function, Server type — primary file moving to runtime
- `library.go` — Path resolution logic — primary file moving to library
- `library_unix.go` / `library_windows.go` — Platform-specific loadLibrary() with build tags

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Phase 1 skeleton packages (`internal/runtime/runtime.go`, `internal/library/library.go`) — target directories already exist with valid package declarations

### Established Patterns
- All Go files use `package chroma` — will change to `package runtime` or `package library`
- Platform-specific files use build tags: `//go:build !windows` and `//go:build windows`
- All test files use `package chroma` (white-box) — will become `package runtime` or `package library`
- FFI initialization uses sync.Once with package-level globals — pattern preserved as-is

### Integration Points
- `loadLibrary()` (unexported, in library_unix.go/library_windows.go) must become `LoadLibrary()` (exported) for runtime to call
- `resolveLibraryLoadPlan()` called by `loadLibrary()` — stays in same package, no export needed
- `registerFunctions()` in chroma.go uses `libHandle` — both stay in runtime, no cross-package concern
- Phase 1 skeleton files will be replaced/merged with real code during the move

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-file-migration*
*Context gathered: 2026-03-20*
