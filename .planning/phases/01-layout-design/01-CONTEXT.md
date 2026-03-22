# Phase 1: Layout Design - Context

**Gathered:** 2026-03-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Create the target directory structure (`internal/runtime/`, `internal/library/`) as compilable skeleton packages at the module root. Confirm that the `internal/` anchor allows root-package imports. No existing Go files are relocated — this phase is strictly additive.

</domain>

<decisions>
## Implementation Decisions

### File-to-package mapping
- `internal/runtime/` gets ALL implementation files: chroma.go, config.go, embedded.go, errors.go, backup.go, rebuild.go, compaction.go, wal_prune.go
- `internal/library/` gets FFI loading files: library.go, library_unix.go, library_windows.go
- Maintenance operations (backup, rebuild, compaction, wal_prune) stay in runtime because they define methods on Server/Embedded receiver types — Go requires methods in the same package as the receiver
- Dependency direction: runtime imports library (to load the shared lib); library is a leaf package with no upstream dependencies

### Package naming
- `internal/runtime` — keeps the roadmap name; stdlib collision handled via import aliasing if needed
- `internal/library` — matches existing library.go naming convention in the codebase

### Skeleton depth
- Bare package declarations only — no stub types or placeholder functions
- Each skeleton file includes a brief 1-2 line doc comment explaining the package's intended role
- Doc comment lives in the main .go file (runtime.go, library.go), not a separate doc.go
- A standalone `internal_test.go` at repo root validates the internal/ anchor:
  - Uses `package chroma_test` (external test perspective)
  - Blank-imports both `internal/runtime` and `internal/library`
  - Proves root package can import internal/ without "use of internal package not allowed" errors
  - Temporary file — removed after Phase 3 when real facade exists

### Claude's Discretion
- Exact doc comment wording for skeleton packages
- Whether to add a .gitkeep or similar in empty directories (likely unnecessary since .go files exist)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project structure
- `.planning/ROADMAP.md` — Phase 1 goal, success criteria, and dependency chain
- `.planning/REQUIREMENTS.md` — LAYOUT-01 and LAYOUT-02 requirements for this phase
- `.planning/PROJECT.md` — Core value (backward-compatible import path), constraints, key decisions

### Existing codebase
- `go.mod` — Module path `github.com/amikos-tech/chroma-go-local`; confirms internal/ anchor point
- `CLAUDE.md` — Architecture diagram, build commands, key patterns

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- No existing internal/ structure — this phase creates it from scratch

### Established Patterns
- All Go files use `package chroma` at root level
- Platform-specific files use build tags: `library_unix.go` (//go:build !windows), `library_windows.go` (//go:build windows)
- Test files mix `package chroma` (internal tests) — the new internal_test.go uses `package chroma_test` (external)

### Integration Points
- `go.mod` at root with module path `github.com/amikos-tech/chroma-go-local` — internal/ must be directly under this
- Existing `go build ./...` must continue to pass with skeletons added alongside current root files
- Makefile targets (`make test`, `make build`) are unaffected in this phase — skeletons are additive

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

*Phase: 01-layout-design*
*Context gathered: 2026-03-20*
