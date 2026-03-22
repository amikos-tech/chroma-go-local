# Phase 4: Build and Test - Research

**Researched:** 2026-03-21
**Domain:** Go build tooling, linting, CI, test organization, cross-compilation
**Confidence:** HIGH

## Summary

Phase 4 is a build/test/lint stabilization phase following the Phase 3 facade creation. The codebase is in a state where the root facade files exist and cross-compilation already passes, but two concrete issues must be fixed: (1) the stale `gci` import prefix in `.golangci.yml` line 55, and (2) two `staticcheck SA1019` lint failures on deprecated type aliases (`ServerBackupOptions`, `EmbeddedBackupOptions`) in `backup.go`. The primary new deliverable is `compat_test.go` at the repo root -- a two-layer gate combining compile-time symbol references for all exported symbols and behavioral smoke tests for key entry points.

The Makefile and CI workflow require zero structural changes. `go test -v ./...` and `golangci-lint run ./...` already traverse the new internal packages. Cross-compilation (`GOOS=linux/darwin/windows go build ./...`) already succeeds. The work is contained and low-risk.

**Primary recommendation:** Fix the two lint issues (gci prefix + SA1019 nolint directives), create compat_test.go with exhaustive symbol refs and ~9-10 behavioral smoke tests, then verify all make targets pass.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Single file at repo root combining compile-only surface gate + behavioral smoke tests
- **D-02:** Compile-only section: `var _ = chroma.X` for every exported type, function, constant, and sentinel error
- **D-03:** Behavioral section: entry points + one test per feature area (~9-10 tests total): Init, Version, NewServer, StartEmbedded, DefaultConfigs, plus one each for backup, rebuild, compaction, WAL prune option builders
- **D-04:** Uses `package chroma_test` (external test package) -- proves the public import path works like a real consumer
- **D-05:** Individual `With*` builder functions are NOT smoke-tested -- the compile gate covers symbol existence, and internal/runtime tests cover behavior
- **D-06:** Root has exactly one test file: `compat_test.go` -- no integration tests, no mirrored internal tests
- **D-07:** All deep behavioral testing remains in `internal/runtime/` and `internal/library/` (12 existing test files)
- **D-08:** Clean separation: root = API surface gate, internal = implementation tests
- **D-09:** `make test` keeps `go test -v ./...` unchanged -- already traverses root + internal packages
- **D-10:** No new Makefile targets
- **D-11:** No structural Makefile changes
- **D-12:** Zero structural changes to `.github/workflows/ci.yml`
- **D-13:** `go test -v ./...` in CI already picks up the new compat_test.go at root
- **D-14:** Fix stale `gci` prefix in `.golangci.yml`: change `github.com/chaoslabs-bg/tclr-v2/` to `github.com/amikos-tech/chroma-go-local/`
- **D-15:** No other lint config changes -- remaining settings are valid for the reorganized layout

### Claude's Discretion
- Exact ordering of compile-only symbol references within compat_test.go
- Test helper setup/teardown patterns for behavioral smoke tests
- Whether behavioral tests use subtests (t.Run) or top-level TestXxx functions
- Cross-compile verification approach (local script vs. manual commands)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TEST-01 | Implementation-focused tests moved alongside new internal packages | Already complete from Phase 2 -- 12 test files exist in internal/runtime/ and internal/library/. Verification only. |
| TEST-02 | Public API compatibility tests remain at root level | compat_test.go at root in `package chroma_test` serves this role |
| TEST-03 | `compat_test.go` added at root as compile-time API surface gate | Full symbol inventory documented below; compile-only `var _` + behavioral smoke tests |
| TEST-04 | `make test` passes with reorganized test layout | `go test -v ./...` traverses root + internal. Requires compat_test.go to compile. |
| BUILD-01 | Makefile targets updated for new package paths | No changes needed -- `go test -v ./...` and `golangci-lint run ./...` already work |
| BUILD-02 | CI workflows updated for new structure | No changes needed -- CI uses same `go test -v ./...` pattern |
| BUILD-03 | Stale `gci` prefix corrected | Line 55 of `.golangci.yml`: change `prefix(github.com/chaoslabs-bg/tclr-v2/)` to `prefix(github.com/amikos-tech/chroma-go-local/)` |
| BUILD-04 | Cross-compile verification passes | Already verified: `GOOS=linux/darwin/windows go build ./...` all succeed today |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.21+ (module), 1.26.1 (local) | Language runtime | Module minimum specified in go.mod |
| golangci-lint | 2.11.3 | Go linting | Project linter; CI uses `golangci/golangci-lint-action@v9.2.0` with `version: latest` |
| stretchr/testify | v1.11.1 | Test assertions | Already in go.mod; used across all internal test files |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| purego | v0.9.1 | Pure Go FFI (no cgo) | Internal; facade does not import directly |

### Alternatives Considered
None -- this phase adds no new dependencies. `compat_test.go` uses only stdlib `testing` and `stretchr/testify` (already available).

## Architecture Patterns

### Current Project Structure (Post-Phase 3)
```
.
├── doc.go              # package chroma -- package doc
├── chroma.go           # Init, Server, StartServer, StartServerConfig, Version, VersionWithError
├── config.go           # ServerConfig, ServerOption, DefaultServerConfig, NewServer, With* builders
├── errors.go           # Error codes (consts) + sentinel errors (vars)
├── embedded.go         # Embedded types + functions + With* embedded options
├── backup.go           # Backup types + options + WithDestination, WithIncludeMetadata, etc.
├── rebuild.go          # Rebuild types + options
├── compaction.go       # Compaction types (no functions -- methods auto-forward via alias)
├── wal_prune.go        # WAL prune types + options (imports time for WithWALPruneMaxAge)
├── compat_test.go      # [TO CREATE] API surface gate + smoke tests
├── internal/
│   ├── runtime/        # 10 impl files + 12 test files (all behavioral tests)
│   └── library/        # 3 impl files + 1 test file (FFI loading)
├── Makefile            # Build/test targets (no changes needed)
├── .golangci.yml       # Lint config (gci prefix fix needed)
└── .github/workflows/ci.yml  # CI (no changes needed)
```

### Pattern 1: External Test Package for API Surface Gate
**What:** Use `package chroma_test` (not `package chroma`) for compat_test.go
**When to use:** Always for root-level compatibility tests -- proves the public import path works
**Example:**
```go
// compat_test.go
package chroma_test

import (
    chroma "github.com/amikos-tech/chroma-go-local"
)

// Compile-time API surface gate
var (
    _ = chroma.Init
    _ = chroma.Version
    // ... all exported symbols
)
```

### Pattern 2: Compile-Only Symbol References
**What:** `var _ = chroma.X` for types, functions, constants, and sentinel errors
**When to use:** To catch symbol removal/rename at compile time without running tests
**Detail:** For types use `var _ chroma.TypeName`; for functions use `var _ = chroma.FuncName`; for constants use `var _ = chroma.ConstName`; for error vars use `var _ = chroma.ErrX`
**Example:**
```go
// Types -- compile fails if any type is removed or renamed
var _ chroma.Server
var _ chroma.ServerConfig
var _ chroma.Embedded
// ...

// Functions -- compile fails if signature changes
var _ = chroma.Init
var _ = chroma.Version
var _ = chroma.NewServer
// ...

// Constants -- compile fails if removed
var _ = chroma.Success
var _ = chroma.DefaultTenantID
// ...

// Sentinel errors -- compile fails if removed
var _ = chroma.ErrNullPointer
// ...
```

### Pattern 3: Behavioral Smoke Tests via Root Facade
**What:** Lightweight tests that exercise key entry points through the public API
**When to use:** To verify the facade wrappers correctly delegate to internal implementations
**Detail:** Tests require `CHROMA_LIB_PATH` env var (set by Makefile). Skip gracefully when unavailable.
**Example:**
```go
func TestVersion(t *testing.T) {
    if err := chroma.Init(""); err != nil {
        t.Fatalf("Init failed: %v", err)
    }
    v := chroma.Version()
    if v == "" {
        t.Fatal("Version returned empty string")
    }
}
```

### Anti-Patterns to Avoid
- **Duplicating internal tests at root:** compat_test.go should NOT mirror the deep behavioral tests in internal/runtime/ -- that creates maintenance burden
- **Using `package chroma` for compat_test.go:** This would be a white-box test that accesses unexported symbols, defeating the purpose of testing the public API surface
- **Testing `With*` builder behavior at root:** The compile gate covers existence; internal tests cover behavior (per D-05)

## Complete Exported Symbol Inventory

This is the definitive list of all symbols that compat_test.go must reference. Verified via `go doc -all .` against the current codebase.

### Constants (14 total)

**Error codes (errors.go):**
- `Success`
- `ErrNullInput`
- `ErrInvalidUTF8`
- `ErrConfigParse`
- `ErrServerStart`
- `ErrInvalidHandle`
- `ErrRuntimeCreate`
- `ErrAlreadyStopped`
- `ErrOperation`

**Embedded defaults (embedded.go):**
- `DefaultTenantID`
- `DefaultDatabase`
- `DefaultEmbeddedDir`

**Backup modes (backup.go):**
- `BackupModeServer`
- `BackupModeEmbedded`

### Sentinel Error Variables (5 total, errors.go)
- `ErrNullPointer`
- `ErrLibraryNotLoaded`
- `ErrServerNotStarted`
- `ErrServerAlreadyStop`
- `ErrEmbeddedNotStarted`

### Types (40 total)

**chroma.go:** `Server`, `StartServerConfig`
**config.go:** `ServerConfig`, `ServerOption`
**embedded.go:** `Embedded`, `StartEmbeddedConfig`, `EmbeddedConfig`, `EmbeddedOption`, `EmbeddedCreateCollectionRequest`, `EmbeddedCollection`, `EmbeddedDatabase`, `EmbeddedTenant`, `EmbeddedCreateTenantRequest`, `EmbeddedGetTenantRequest`, `EmbeddedUpdateTenantRequest`, `EmbeddedCreateDatabaseRequest`, `EmbeddedListDatabasesRequest`, `EmbeddedGetDatabaseRequest`, `EmbeddedDeleteDatabaseRequest`, `EmbeddedListCollectionsRequest`, `EmbeddedGetCollectionRequest`, `EmbeddedCountCollectionsRequest`, `EmbeddedUpdateCollectionRequest`, `EmbeddedDeleteCollectionRequest`, `EmbeddedForkCollectionRequest`, `EmbeddedAddRequest`, `EmbeddedQueryRequest`, `EmbeddedQueryResponse`, `EmbeddedCountRecordsRequest`, `EmbeddedGetRecordsRequest`, `EmbeddedGetRecordsResponse`, `EmbeddedUpdateRecordsRequest`, `EmbeddedUpsertRecordsRequest`, `EmbeddedDeleteRecordsRequest`, `EmbeddedDeleteRecordsResponse`, `EmbeddedIndexingStatusRequest`, `EmbeddedIndexingStatusResponse`, `EmbeddedHealthCheckResponse`
**backup.go:** `BackupMode`, `BackupOptions`, `ServerBackupOptions`, `EmbeddedBackupOptions`, `BackupOption`, `BackupFileMetadata`, `BackupManifest`
**rebuild.go:** `RebuildCollectionResult`, `RebuildCollectionOption`
**compaction.go:** `CompactCollectionRequest`, `CompactAllRequest`, `CompactionCollectionResult`, `CompactionResult`
**wal_prune.go:** `WALPruneCollectionResult`, `WALPruneResult`, `WALPruneOption`

### Functions (28 total)

**chroma.go:** `Init`, `StartServer`, `Version`, `VersionWithError`
**config.go:** `DefaultServerConfig`, `NewServer`, `WithPort`, `WithListenAddress`, `WithMaxPayloadSize`, `WithCORSAllowOrigins`, `WithPersistPath`, `WithSQLiteFilename`, `WithAllowReset`, `WithOpenTelemetry`, `WithRawYAML`
**embedded.go:** `DefaultEmbeddedConfig`, `NewEmbedded`, `StartEmbedded`, `WithEmbeddedPersistPath`, `WithEmbeddedSQLiteFilename`, `WithEmbeddedAllowReset`, `WithEmbeddedRawYAML`
**backup.go:** `WithDestination`, `WithIncludeMetadata`, `WithLeaveStopped`, `WithLeaveClosed`
**rebuild.go:** `WithRebuildTenantID`, `WithRebuildDatabaseName`, `WithRebuildPrecheck`, `WithRebuildKeepBackup`
**wal_prune.go:** `WithWALPruneTenantID`, `WithWALPruneDatabaseName`, `WithWALPruneDryRun`, `WithWALPruneVacuum`, `WithWALPruneMaxAge`, `WithWALPruneMaxBytes`, `WithWALPruneWatermark`

**Grand total: 87 exported symbols** (14 constants + 5 error vars + 40 types + 28 functions)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| API surface tracking | Manual checklist of symbols | `var _ = chroma.X` compile gate in compat_test.go | Compiler catches removals automatically; zero maintenance |
| Full API diff tool | Custom script to compare exports | `go-apidiff` (Phase 5) | Well-maintained tool; this phase focuses on compile gate |
| Cross-platform test runner | Custom CI scripts per OS | Existing CI matrix in ci.yml | Already handles Linux/macOS/Windows with pwsh |

## Common Pitfalls

### Pitfall 1: Deprecated Type Aliases Trigger SA1019
**What goes wrong:** `golangci-lint` flags `type ServerBackupOptions = runtime.ServerBackupOptions` and `type EmbeddedBackupOptions = runtime.EmbeddedBackupOptions` with `SA1019: deprecated` because the internal types have `// Deprecated:` doc comments.
**Why it happens:** The facade re-exports these types for backward compatibility; staticcheck sees the reference to a deprecated symbol.
**How to avoid:** Add `//nolint:staticcheck // re-export deprecated type for backward compatibility` inline on those two lines in `backup.go`.
**Warning signs:** `golangci-lint run ./...` reports 2 SA1019 issues (currently failing).

### Pitfall 2: compat_test.go Behavioral Tests Need CHROMA_LIB_PATH
**What goes wrong:** Behavioral smoke tests that call `chroma.Init("")` fail if `CHROMA_LIB_PATH` env var is not set or the Rust shim is not built.
**Why it happens:** `Init("")` reads `CHROMA_LIB_PATH` to find the shared library. The Makefile sets it automatically, but running `go test ./...` directly does not.
**How to avoid:** Behavioral tests should use the same Init pattern as internal tests (Init with empty string, relying on CHROMA_LIB_PATH). The Makefile already handles this. No special skip logic needed -- if the env is not set, Init returns an error and the test fails, which is correct behavior.
**Warning signs:** Tests pass with `make test` but fail with bare `go test ./...`.

### Pitfall 3: Type Alias vs Type Definition in var _ Declarations
**What goes wrong:** Using `var _ = chroma.ServerConfig` (with `=`) for a struct type produces a zero value, not a compile check. Using `var _ chroma.ServerConfig` (without `=`) declares a typed variable that fails to compile if the type is removed.
**Why it happens:** Go semantics: `var _ = X` evaluates X as an expression; `var _ T` declares a variable of type T.
**How to avoid:** Use `var _ chroma.TypeName` (no `=`) for types/structs. Use `var _ = chroma.FuncName` (with `=`) for functions and variables. Use `const _ = chroma.ConstName` or `var _ = chroma.ConstName` for constants.
**Warning signs:** A type could be removed without the compile gate catching it if `=` is used with a struct literal.

### Pitfall 4: Function Signature Changes Not Caught by var _ = F
**What goes wrong:** `var _ = chroma.Init` compiles even if Init's signature changes (e.g., from `func(string) error` to `func(string, int) error`), because the var just holds a function value.
**Why it happens:** Go function values are assignable regardless of signature.
**How to avoid:** For critical functions, also use typed assignments: `var _ func(string) error = chroma.Init`. For this phase, the compile-only gate plus behavioral smoke tests provide sufficient coverage.
**Warning signs:** API signature changes go undetected.

### Pitfall 5: gci Formatter Prefix Must Match Module Path Exactly
**What goes wrong:** If the `gci` prefix does not match the module path, all project imports get sorted into the `default` section instead of being grouped separately.
**Why it happens:** `gci` uses prefix matching to categorize imports. Wrong prefix = no match = wrong section.
**How to avoid:** Verify the prefix matches go.mod module path: `github.com/amikos-tech/chroma-go-local/`.
**Warning signs:** Import ordering issues reported by lint, or imports not grouped correctly in formatted files.

## Code Examples

### compat_test.go Structure (Compile-Only Section)

```go
package chroma_test

import (
    "testing"
    "time"

    chroma "github.com/amikos-tech/chroma-go-local"
    "github.com/stretchr/testify/require"
)

// -- Compile-time API surface gate --
// Every exported type, function, constant, and variable must appear below.
// If a symbol is removed or renamed, this file fails to compile.

// Types
var _ chroma.Server
var _ chroma.StartServerConfig
var _ chroma.ServerConfig
var _ chroma.ServerOption
var _ chroma.Embedded
// ... (all 40 types)

// Functions
var _ = chroma.Init
var _ = chroma.Version
var _ = chroma.VersionWithError
var _ = chroma.StartServer
var _ = chroma.NewServer
var _ = chroma.DefaultServerConfig
// ... (all 28 functions)

// Constants
var _ = chroma.Success
var _ = chroma.ErrNullInput
// ... (all 14 constants)

// Sentinel errors
var _ = chroma.ErrNullPointer
var _ = chroma.ErrLibraryNotLoaded
// ... (all 5 error vars)
```

### compat_test.go Structure (Behavioral Section)

```go
func TestInit(t *testing.T) {
    err := chroma.Init("")
    require.NoError(t, err)
}

func TestVersion(t *testing.T) {
    err := chroma.Init("")
    require.NoError(t, err)
    v := chroma.Version()
    require.NotEmpty(t, v)
}

func TestDefaultServerConfig(t *testing.T) {
    cfg := chroma.DefaultServerConfig()
    require.NotNil(t, cfg)
    require.Greater(t, cfg.Port, 0)
}

func TestDefaultEmbeddedConfig(t *testing.T) {
    cfg := chroma.DefaultEmbeddedConfig()
    require.NotNil(t, cfg)
}
```

### Lint Fix for Deprecated Type Aliases

```go
// In backup.go, lines 7-8:
type ServerBackupOptions = runtime.ServerBackupOptions   //nolint:staticcheck // re-export deprecated type for backward compatibility
type EmbeddedBackupOptions = runtime.EmbeddedBackupOptions //nolint:staticcheck // re-export deprecated type for backward compatibility
```

### golangci.yml gci Prefix Fix

```yaml
# Line 55: change from
- prefix(github.com/chaoslabs-bg/tclr-v2/)
# to
- prefix(github.com/amikos-tech/chroma-go-local/)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| golangci-lint v1 config format | golangci-lint v2 config format | v2.0.0 (2025) | This project already uses `version: "2"` in .golangci.yml |
| `issues.exclude-rules` | `exclusions.rules` | golangci-lint v2 | Already migrated in current config |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + stretchr/testify v1.11.1 |
| Config file | None (standard go test) |
| Quick run command | `make test` |
| Full suite command | `make test-all` |

### Phase Requirements Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TEST-01 | Internal tests alongside packages | verification | `go test ./internal/...` | Existing (12 files) |
| TEST-02 | Public API compat tests at root | compile + smoke | `go test -v -run TestCompat .` | Wave 0 (compat_test.go) |
| TEST-03 | compat_test.go compile gate | compile | `go build ./...` | Wave 0 (compat_test.go) |
| TEST-04 | make test passes | integration | `make test` | Existing (Makefile) |
| BUILD-01 | Makefile targets work | integration | `make test && make lint` | Existing (no changes) |
| BUILD-02 | CI workflows work | verification | Visual inspection of ci.yml | Existing (no changes) |
| BUILD-03 | gci prefix corrected | lint | `golangci-lint run ./...` | Existing (.golangci.yml) |
| BUILD-04 | Cross-compile passes | build | `GOOS=linux go build ./... && GOOS=darwin go build ./... && GOOS=windows go build ./...` | N/A |

### Sampling Rate
- **Per task commit:** `go build ./... && golangci-lint run ./...`
- **Per wave merge:** `make test && make lint`
- **Phase gate:** `make test-all` + cross-compile all 3 OS + `golangci-lint run ./...` reports zero issues

### Wave 0 Gaps
- [ ] `compat_test.go` -- root-level API surface gate + behavioral smoke tests (covers TEST-02, TEST-03)
- No framework install needed -- Go testing and testify already available

## Open Questions

1. **Deprecated type re-export lint suppression approach**
   - What we know: `backup.go` lines 7-8 re-export deprecated types, causing SA1019 lint failures. Inline `//nolint:staticcheck` is the standard approach.
   - What's unclear: Whether the project prefers inline nolint or a lint exclusion rule. CONTEXT D-15 says "no other lint config changes", suggesting inline nolint is the intended path.
   - Recommendation: Use inline `//nolint:staticcheck` with explanatory comment. This is the least invasive fix and aligns with D-15.

2. **Behavioral test ordering and TestMain**
   - What we know: Behavioral tests need `chroma.Init("")` called first. Internal tests call Init in each test function.
   - What's unclear: Whether to use a `TestMain` to init once, or call Init in each test.
   - Recommendation: Call Init per-test (matching internal test patterns). A TestMain adds complexity for minimal benefit with ~9-10 tests.

## Sources

### Primary (HIGH confidence)
- Direct codebase inspection: all 9 facade files, Makefile, .golangci.yml, ci.yml
- `go doc -all .` output: complete exported symbol inventory
- `golangci-lint run ./...`: verified 2 SA1019 issues are the only lint failures
- Cross-compile verification: `GOOS=linux/darwin/windows go build ./...` all succeeded
- `golangci-lint --version`: v2.11.3 confirmed

### Secondary (MEDIUM confidence)
- golangci-lint v2 configuration format: verified against current .golangci.yml which uses `version: "2"`

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all tools and libraries verified against local installation and go.mod
- Architecture: HIGH - complete symbol inventory verified via `go doc -all .`; project structure inspected directly
- Pitfalls: HIGH - lint issues reproduced locally; cross-compile verified; deprecated type behavior confirmed

**Research date:** 2026-03-21
**Valid until:** 2026-04-21 (stable -- no fast-moving dependencies in this phase)
