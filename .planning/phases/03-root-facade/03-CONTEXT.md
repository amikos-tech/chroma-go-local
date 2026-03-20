# Phase 3: Root Facade - Context

**Gathered:** 2026-03-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Write a thin facade at the repo root that re-exports every public symbol from `internal/runtime` (and transitively `internal/library`) so that the import path `github.com/amikos-tech/chroma-go-local` works identically to before the migration. Existing callers must compile and behave identically without changing a single import statement.

</domain>

<decisions>
## Implementation Decisions

### Facade file organization
- **D-01:** 8 facade files at root mirroring internal/runtime filenames: `chroma.go`, `config.go`, `embedded.go`, `errors.go`, `backup.go`, `rebuild.go`, `compaction.go`, `wal_prune.go`
- **D-02:** Separate `doc.go` for package-level documentation — keeps the package comment out of alias files
- **D-03:** Each facade file contains only type aliases, wrapper funcs, const/var re-exports — zero logic

### Function delegation strategy
- **D-04:** **ROADMAP OVERRIDE** — Use thin wrapper functions (`func F(...) { return runtime.F(...) }`) instead of `var F = runtime.F` assignments
- **D-05:** Rationale: wrapper funcs preserve proper function signatures in godoc/pkg.go.dev, which is a significant usability feature for API consumers. The trade-off is one extra stack frame per call (negligible) and maintenance burden when the API changes.
- **D-06:** Type aliases (`type X = runtime.X`) for all exported types — methods forward automatically, `chroma.Server` and `runtime.Server` are the same type

### Root package documentation
- **D-07:** Package doc comment in `doc.go` is user-facing — describes what chroma-go-local does (FFI wrapper for Chroma), not that it's a facade. Internal structure is invisible to consumers.

### Error and constant re-export
- **D-08:** Error codes (int32 constants) and sentinel errors (var) grouped together in `errors.go` facade file
- **D-09:** Constants re-exported via `const X = runtime.X`; sentinel errors via `var ErrX = runtime.ErrX`

### API drift detection
- **D-10:** Deferred to Phase 5's go-apidiff gate — no automated drift detection in this phase

### Claude's Discretion
- Exact ordering of symbols within each facade file
- Wrapper function body style (single-line return vs. multi-line)
- Whether to add brief inline comments grouping symbol sections within files
- Exact wording of the package doc comment in doc.go

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project structure
- `.planning/ROADMAP.md` — Phase 3 goal, success criteria (FACADE-01 through FACADE-05), note D-04 overrides criteria #2
- `.planning/REQUIREMENTS.md` — FACADE-01 (type aliases), FACADE-02 (function re-export), FACADE-03 (constants/vars/errors), FACADE-04 (zero logic), FACADE-05 (import path unchanged)
- `.planning/PROJECT.md` — Core value (backward-compatible import path), constraints, key decisions

### Prior phase context
- `.planning/phases/01-layout-design/01-CONTEXT.md` — File-to-package mapping, package naming decisions
- `.planning/phases/02-file-migration/02-CONTEXT.md` — Internal API boundary (library exports only LoadLibrary), runtime package structure

### Codebase (source of truth for symbols to re-export)
- `internal/runtime/chroma.go` — Init, Server, StartServer, StartServerConfig, Version, VersionWithError
- `internal/runtime/config.go` — ServerConfig, ServerOption, DefaultServerConfig, With* builder functions (8 total)
- `internal/runtime/embedded.go` — Embedded, EmbeddedConfig, EmbeddedOption, ~30 request/response types, NewEmbedded, StartEmbedded, DefaultEmbeddedConfig, WithEmbedded* builders
- `internal/runtime/errors.go` — 8 error code constants (Success through ErrOperation), 5 sentinel errors (ErrNullPointer through ErrEmbeddedNotStarted)
- `internal/runtime/backup.go` — BackupMode, BackupOption, BackupOptions, ServerBackupOptions, EmbeddedBackupOptions, BackupFileMetadata, BackupManifest, With* backup options, BackupMode constants
- `internal/runtime/rebuild.go` — RebuildCollectionResult, RebuildCollectionOption, WithRebuild* options
- `internal/runtime/compaction.go` — CompactCollectionRequest, CompactAllRequest, CompactionCollectionResult, CompactionResult
- `internal/runtime/wal_prune.go` — WALPruneCollectionResult, WALPruneResult, WALPruneOption, WithWALPrune* options
- `CLAUDE.md` — Architecture diagram, build commands, key patterns
- `examples/go/basic/main.go` — Consumer example that imports root package (must compile after facade)

</canonical_refs>

<code_context>
## Existing Code Insights

### Symbol inventory (approximate)
- 54 exported types (structs, interfaces, func types)
- 37 exported top-level functions (constructors, builders, Init, Version)
- 67 exported methods on types (automatic via type aliases — no facade code needed)
- 16 exported constants (error codes + BackupMode values)
- 5 exported sentinel error variables
- Total facade surface: ~112 explicit re-exports (types + funcs + consts + vars); methods are free

### Reusable Assets
- No existing facade code — this phase creates it from scratch
- `examples/go/basic/main.go` — existing consumer that validates facade works (imports `chroma "github.com/amikos-tech/chroma-go-local"`, calls `chroma.Init`, `chroma.Version`, `chroma.NewServer`, etc.)

### Established Patterns
- Type alias pattern: `type X = runtime.X` — Go 1.9+ feature, methods transfer automatically
- Wrapper func pattern: `func F(args) (returns) { return runtime.F(args) }` — preserves godoc signatures
- Const re-export: `const X = runtime.X` — works for all constant types
- Var re-export: `var ErrX = runtime.ErrX` — shares the same error value

### Integration Points
- Root package must be named `chroma` (matches pre-migration package name)
- Root package imports `internal/runtime` — this is valid because root is the module owner
- Root package does NOT import `internal/library` — library is accessed transitively through runtime
- `go build ./...` must succeed including `examples/go/basic/main.go` which imports root
- `go test ./...` runs internal/runtime tests (no tests at root until Phase 4 adds compat_test.go)

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

*Phase: 03-root-facade*
*Context gathered: 2026-03-20*
