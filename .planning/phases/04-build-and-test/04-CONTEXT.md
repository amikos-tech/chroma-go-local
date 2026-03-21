# Phase 4: Build and Test - Context

**Gathered:** 2026-03-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Update Makefile, CI, lint config, and test layout to reflect the new `internal/runtime/` and `internal/library/` package structure. Add a root-level `compat_test.go` as a compile-time API surface gate with behavioral smoke tests. Verify all `make` targets, lint, and cross-compile pass. No new features or API changes.

</domain>

<decisions>
## Implementation Decisions

### compat_test.go design
- **D-01:** Single file at repo root combining compile-only surface gate + behavioral smoke tests
- **D-02:** Compile-only section: `var _ = chroma.X` for every exported type, function, constant, and sentinel error — catches symbol removal/rename at compile time
- **D-03:** Behavioral section: entry points + one test per feature area (~9-10 tests total): Init, Version, NewServer, StartEmbedded, DefaultConfigs, plus one each for backup, rebuild, compaction, WAL prune option builders
- **D-04:** Uses `package chroma_test` (external test package) — proves the public import path works like a real consumer
- **D-05:** Individual `With*` builder functions are NOT smoke-tested — the compile gate covers symbol existence, and internal/runtime tests cover behavior

### Root-level test strategy
- **D-06:** Root has exactly one test file: `compat_test.go` — no integration tests, no mirrored internal tests
- **D-07:** All deep behavioral testing remains in `internal/runtime/` and `internal/library/` (12 existing test files)
- **D-08:** Clean separation: root = API surface gate, internal = implementation tests

### Makefile adjustments
- **D-09:** `make test` keeps `go test -v ./...` unchanged — already traverses root + internal packages
- **D-10:** No new Makefile targets (no `test-compat`, `test-internal`, or `test-race`)
- **D-11:** No structural Makefile changes — all existing targets work with the new layout

### CI workflow
- **D-12:** Zero structural changes to `.github/workflows/ci.yml` — OS matrix, job structure, and commands stay as-is
- **D-13:** `go test -v ./...` in CI already picks up the new compat_test.go at root

### Lint configuration
- **D-14:** Fix stale `gci` prefix in `.golangci.yml`: change `github.com/chaoslabs-bg/tclr-v2/` to `github.com/amikos-tech/chroma-go-local/`
- **D-15:** No other lint config changes — remaining settings (linters, exclusions, paths) are valid for the reorganized layout

### Claude's Discretion
- Exact ordering of compile-only symbol references within compat_test.go
- Test helper setup/teardown patterns for behavioral smoke tests
- Whether behavioral tests use subtests (t.Run) or top-level TestXxx functions
- Cross-compile verification approach (local script vs. manual commands)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project structure
- `.planning/ROADMAP.md` — Phase 4 goal, success criteria (TEST-01 through TEST-04, BUILD-01 through BUILD-04)
- `.planning/REQUIREMENTS.md` — TEST-01 (tests alongside internal), TEST-02 (compat tests at root), TEST-03 (compat_test.go), TEST-04 (make test passes), BUILD-01 (Makefile), BUILD-02 (CI), BUILD-03 (gci prefix), BUILD-04 (cross-compile)
- `.planning/PROJECT.md` — Core value (backward-compatible import path), constraints

### Prior phase context
- `.planning/phases/01-layout-design/01-CONTEXT.md` — File-to-package mapping, package naming
- `.planning/phases/02-file-migration/02-CONTEXT.md` — Internal API boundary, test placement decisions
- `.planning/phases/03-root-facade/03-CONTEXT.md` — Facade file organization, wrapper function strategy (D-04: thin wrappers not var assignments), type alias approach

### Build and lint config (source of truth for current state)
- `Makefile` — Current test/lint/build targets; `RUN_GO_TEST_DEBUG` uses `go test -v ./...`
- `.golangci.yml` — Lint config with stale gci prefix on line 55
- `.github/workflows/ci.yml` — CI workflow: 3-OS matrix, `go test -v ./...`, golangci-lint action

### Codebase (facade symbols to reference in compat_test.go)
- `chroma.go` — Init, Server, StartServer, StartServerConfig, Version, VersionWithError
- `config.go` — ServerConfig, ServerOption, DefaultServerConfig, With* builder functions
- `embedded.go` — Embedded, EmbeddedConfig, EmbeddedOption, ~30 request/response types, NewEmbedded, StartEmbedded
- `errors.go` — 8 error code constants, 5 sentinel errors
- `backup.go` — BackupMode, BackupOption, BackupOptions, BackupFileMetadata, BackupManifest, With* backup options
- `rebuild.go` — RebuildCollectionResult, RebuildCollectionOption, WithRebuild* options
- `compaction.go` — CompactCollectionRequest, CompactAllRequest, CompactionCollectionResult, CompactionResult
- `wal_prune.go` — WALPruneCollectionResult, WALPruneResult, WALPruneOption, WithWALPrune* options

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- 9 facade files at root (doc.go, chroma.go, config.go, errors.go, embedded.go, backup.go, rebuild.go, compaction.go, wal_prune.go) — source of truth for symbols to reference in compat_test.go
- 12 existing test files in internal/runtime/ and internal/library/ — no changes needed
- `examples/go/basic/main.go` — existing consumer example that validates facade compiles

### Established Patterns
- External test packages use `package chroma_test` with named import `chroma "github.com/amikos-tech/chroma-go-local"`
- Internal tests use `package runtime` / `package library` (white-box)
- All Go tests require CHROMA_LIB_PATH pointing to the Rust shim artifact

### Integration Points
- `go test -v ./...` from Makefile and CI traverses root + internal packages
- `golangci-lint run ./...` traverses all Go code including new compat_test.go
- Cross-compile (`GOOS=linux/darwin/windows go build ./...`) validates the facade compiles on all platforms

</code_context>

<specifics>
## Specific Ideas

- compat_test.go serves as a two-layer gate: compile-only for 100% symbol coverage, behavioral for key entry points + one per feature domain
- Defense-in-depth: compile gate (Phase 4) + behavioral smoke (Phase 4) + go-apidiff (Phase 5) together provide comprehensive regression protection

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-build-and-test*
*Context gathered: 2026-03-21*
