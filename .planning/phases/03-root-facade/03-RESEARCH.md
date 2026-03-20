# Phase 3: Root Facade - Research

**Researched:** 2026-03-20
**Domain:** Go type aliases, package facade pattern, symbol re-export
**Confidence:** HIGH

## Summary

Phase 3 creates a thin facade at the repo root (`package chroma`) that re-exports every public symbol from `internal/runtime` so that the import path `github.com/amikos-tech/chroma-go-local` continues to work unchanged. The root package currently contains zero `.go` files (Phase 2 removed them all), so the facade must be created from scratch.

The Go language has native support for this pattern: type aliases (`type X = Y`) transparently forward all methods and interface satisfaction, constant re-export (`const X = pkg.X`) works for all constant types, and sentinel error variables can be shared via `var ErrX = pkg.ErrX`. The user-locked decision D-04 overrides the roadmap's `var F = runtime.F` strategy in favor of thin wrapper functions for better godoc rendering.

**Primary recommendation:** Create 9 files at root (8 facade files mirroring `internal/runtime` filenames + 1 `doc.go`), containing only type aliases, wrapper functions, const re-exports, and var re-exports. Zero logic, zero `init()`, zero local state. Validate by running `go build ./...` (including the existing `examples/go/basic/main.go` consumer).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** 8 facade files at root mirroring internal/runtime filenames: `chroma.go`, `config.go`, `embedded.go`, `errors.go`, `backup.go`, `rebuild.go`, `compaction.go`, `wal_prune.go`
- **D-02:** Separate `doc.go` for package-level documentation -- keeps the package comment out of alias files
- **D-03:** Each facade file contains only type aliases, wrapper funcs, const/var re-exports -- zero logic
- **D-04:** **ROADMAP OVERRIDE** -- Use thin wrapper functions (`func F(...) { return runtime.F(...) }`) instead of `var F = runtime.F` assignments
- **D-05:** Rationale: wrapper funcs preserve proper function signatures in godoc/pkg.go.dev, which is a significant usability feature for API consumers. The trade-off is one extra stack frame per call (negligible) and maintenance burden when the API changes.
- **D-06:** Type aliases (`type X = runtime.X`) for all exported types -- methods forward automatically, `chroma.Server` and `runtime.Server` are the same type
- **D-07:** Package doc comment in `doc.go` is user-facing -- describes what chroma-go-local does (FFI wrapper for Chroma), not that it's a facade. Internal structure is invisible to consumers.
- **D-08:** Error codes (int32 constants) and sentinel errors (var) grouped together in `errors.go` facade file
- **D-09:** Constants re-exported via `const X = runtime.X`; sentinel errors via `var ErrX = runtime.ErrX`
- **D-10:** Deferred to Phase 5's go-apidiff gate -- no automated drift detection in this phase

### Claude's Discretion
- Exact ordering of symbols within each facade file
- Wrapper function body style (single-line return vs. multi-line)
- Whether to add brief inline comments grouping symbol sections within files
- Exact wording of the package doc comment in doc.go

### Deferred Ideas (OUT OF SCOPE)
- None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| FACADE-01 | Root package exposes all current public types via type aliases (`type X = impl.X`) | Type alias pattern verified; complete type inventory documented (54 types) |
| FACADE-02 | Root package re-exports all public functions via variable assignments or wrapper calls | D-04 overrides to wrapper funcs; complete function inventory documented (37 functions) |
| FACADE-03 | Root package re-exports all constants, variables, and error types | Const/var re-export patterns verified; complete inventory documented (16 constants, 5 sentinel errors) |
| FACADE-04 | Root package contains zero implementation logic (pure forwarding only) | Pattern enforced by D-03; verification via grep for logic constructs |
| FACADE-05 | Import path `github.com/amikos-tech/chroma-go-local` remains valid and unchanged | `go.mod` module path confirmed; root package name `chroma` matches pre-migration |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.21+ (module requires) | Language runtime | Type aliases available since Go 1.9; no new features needed |

### Supporting
No additional libraries needed. The facade imports only `internal/runtime` and re-exports its symbols. No new dependencies.

## Architecture Patterns

### Recommended Project Structure
```
/ (module root, package chroma)
├── doc.go              # Package-level documentation
├── chroma.go           # Init, Server, StartServer, StartServerConfig, Version, VersionWithError
├── config.go           # ServerConfig, ServerOption, DefaultServerConfig, With* server builders
├── embedded.go         # Embedded, EmbeddedConfig, EmbeddedOption, ~30 request/response types, NewEmbedded, StartEmbedded, DefaultEmbeddedConfig, WithEmbedded* builders, constants (DefaultTenantID, DefaultDatabase, DefaultEmbeddedDir)
├── errors.go           # Error code constants, sentinel error variables
├── backup.go           # BackupMode, BackupOption, BackupOptions, ServerBackupOptions, EmbeddedBackupOptions, BackupFileMetadata, BackupManifest, With* backup options, BackupMode constants
├── rebuild.go          # RebuildCollectionResult, RebuildCollectionOption, WithRebuild* options
├── compaction.go       # CompactCollectionRequest, CompactAllRequest, CompactionCollectionResult, CompactionResult
├── wal_prune.go        # WALPruneCollectionResult, WALPruneResult, WALPruneOption, WithWALPrune* options
├── internal/
│   ├── runtime/        # All implementation (source of truth)
│   └── library/        # FFI loading
├── examples/
│   └── go/basic/main.go  # Consumer that validates facade (imports root package)
└── go.mod              # module github.com/amikos-tech/chroma-go-local
```

### Pattern 1: Type Alias Re-export
**What:** `type X = runtime.X` creates an identical type, not a new type. Methods and interface satisfaction transfer automatically.
**When to use:** All exported struct types, interface types, and function types from `internal/runtime`.
**Example:**
```go
// Source: Go spec https://go.dev/ref/spec#Type_declarations
import "github.com/amikos-tech/chroma-go-local/internal/runtime"

type Server = runtime.Server
type ServerConfig = runtime.ServerConfig
type ServerOption = runtime.ServerOption
```

### Pattern 2: Thin Wrapper Function (D-04 override)
**What:** A function that delegates to `runtime.F` with the same signature. Preserves godoc rendering.
**When to use:** All exported package-level functions (Init, Version, NewServer, StartServer, etc.).
**Example:**
```go
func Init(libPath string) error {
	return runtime.Init(libPath)
}

func NewServer(opts ...ServerOption) (*Server, error) {
	return runtime.NewServer(opts...)
}
```

### Pattern 3: Constant Re-export
**What:** `const X = runtime.X` copies the constant value. Works for all untyped and typed constants.
**When to use:** Error codes (int32 constants), string constants, BackupMode constants.
**Example:**
```go
const (
	Success           = runtime.Success
	ErrNullInput      = runtime.ErrNullInput
	BackupModeServer  = runtime.BackupModeServer
)
```

### Pattern 4: Sentinel Error Variable Re-export
**What:** `var ErrX = runtime.ErrX` shares the same error value. `errors.Is` checks work correctly because both variables point to the same underlying error.
**When to use:** All sentinel error variables (ErrNullPointer, ErrLibraryNotLoaded, etc.).
**Example:**
```go
var (
	ErrNullPointer        = runtime.ErrNullPointer
	ErrLibraryNotLoaded   = runtime.ErrLibraryNotLoaded
	ErrServerNotStarted   = runtime.ErrServerNotStarted
	ErrServerAlreadyStop  = runtime.ErrServerAlreadyStop
	ErrEmbeddedNotStarted = runtime.ErrEmbeddedNotStarted
)
```

### Anti-Patterns to Avoid
- **Type definitions instead of aliases:** `type Server runtime.Server` strips all methods and breaks callers. MUST use `type Server = runtime.Server` (with `=`).
- **Adding logic to facade:** Any `if`, loop, or computation in a facade file violates FACADE-04 and creates divergence risk.
- **`init()` functions in facade:** Would add hidden initialization that doesn't exist in `internal/runtime`. Root must have zero `init()` functions per success criteria.
- **Importing `internal/library` from root:** The root must only import `internal/runtime`. Library access is transitional through runtime.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Type forwarding | Manual struct definitions with method wrappers | `type X = runtime.X` (type alias) | Aliases are zero-cost and automatically forward all methods, including future ones |
| Symbol inventory drift detection | Custom AST scanner in this phase | `go-apidiff` in Phase 5 | D-10 defers automated drift detection to Phase 5 |
| Package documentation | Doc comments on every alias | Single `doc.go` with package comment | D-02 decision; keeps alias files clean |

## Common Pitfalls

### Pitfall 1: Using Type Definition Instead of Type Alias
**What goes wrong:** `type Server runtime.Server` creates a new, distinct type with zero methods. All callers that call `server.Close()`, `server.Port()`, etc. get compile errors.
**Why it happens:** Forgetting the `=` sign. In Go, `type X Y` and `type X = Y` look nearly identical but have opposite semantics.
**How to avoid:** Every type re-export MUST include `=`. Verify with `go build ./...` and specifically check that `examples/go/basic/main.go` compiles (it calls methods like `.URL()` and `.Close()`).
**Warning signs:** Compile errors about undefined methods on facade types.

### Pitfall 2: Missing Symbols
**What goes wrong:** A caller references `chroma.SomeType` that exists in `internal/runtime` but wasn't added to the facade. Compile error for that caller.
**Why it happens:** Large API surface (~112 explicit re-exports). Easy to miss one.
**How to avoid:** Use `go doc ./internal/runtime/` output as the authoritative symbol list. Cross-check each facade file against the corresponding runtime file. The complete inventory is in this research document.
**Warning signs:** `go build ./examples/...` fails; `go vet ./...` reports unused imports in facade files (suggests symbols are missing, not that they're unused).

### Pitfall 3: Const Re-export of Typed Constants
**What goes wrong:** `const BackupModeServer = runtime.BackupModeServer` works because Go allows re-exporting typed string constants. But if someone tries `const X BackupMode = runtime.BackupModeServer` (with explicit type), it also works but is redundant.
**Why it happens:** Confusion about whether typed constants need explicit type annotations.
**How to avoid:** Use bare `const X = runtime.X` for all constants. The type information is preserved through the constant value.
**Warning signs:** None -- both forms compile, but bare form is cleaner.

### Pitfall 4: Package Name Mismatch
**What goes wrong:** If the facade files declare `package chromalocal` or any name other than `chroma`, the existing consumer `examples/go/basic/main.go` breaks because it uses `chroma "github.com/amikos-tech/chroma-go-local"`.
**Why it happens:** Uncertainty about what package name to use at root.
**How to avoid:** All facade files MUST declare `package chroma`. This matches the pre-migration package name and the example consumer's import alias.
**Warning signs:** Import resolution errors in example or downstream code.

### Pitfall 5: Var Re-export Creates Separate Error Identity
**What goes wrong:** This is actually NOT a problem with `var ErrX = runtime.ErrX` because it shares the same pointer. But if someone writes `var ErrX = errors.New("...")` duplicating the message, `errors.Is` checks fail.
**Why it happens:** Attempting to "clean up" error definitions rather than simply forwarding them.
**How to avoid:** Always forward: `var ErrX = runtime.ErrX`. Never recreate.
**Warning signs:** Tests using `errors.Is(err, chroma.ErrServerNotStarted)` fail.

### Pitfall 6: Golangci-lint GCI Import Ordering
**What goes wrong:** The `.golangci.yml` has a stale `gci` prefix (`github.com/chaoslabs-bg/tclr-v2/`). Lint may report import ordering issues for the new facade files.
**Why it happens:** The GCI config hasn't been updated for this project (noted as BUILD-03, deferred to Phase 4).
**How to avoid:** In Phase 3, facade files only import `internal/runtime` (one import). This is unlikely to trigger GCI complaints since there's only one import. If lint errors occur, they are Phase 4 scope.
**Warning signs:** `golangci-lint run ./...` reports import ordering issues.

## Code Examples

### Complete facade file pattern (chroma.go)
```go
package chroma

import "github.com/amikos-tech/chroma-go-local/internal/runtime"

// Server types
type Server = runtime.Server
type StartServerConfig = runtime.StartServerConfig

// Init initializes the Chroma library.
func Init(libPath string) error {
	return runtime.Init(libPath)
}

// StartServer starts a new Chroma server with the given configuration.
func StartServer(config StartServerConfig) (*Server, error) {
	return runtime.StartServer(config)
}

// Version returns the version of the Chroma shim library.
func Version() string {
	return runtime.Version()
}

// VersionWithError returns the version of the Chroma shim library.
func VersionWithError() (string, error) {
	return runtime.VersionWithError()
}
```

### Complete facade file pattern (errors.go)
```go
package chroma

import "github.com/amikos-tech/chroma-go-local/internal/runtime"

// Error codes
const (
	Success           = runtime.Success
	ErrNullInput      = runtime.ErrNullInput
	ErrInvalidUTF8    = runtime.ErrInvalidUTF8
	ErrConfigParse    = runtime.ErrConfigParse
	ErrServerStart    = runtime.ErrServerStart
	ErrInvalidHandle  = runtime.ErrInvalidHandle
	ErrRuntimeCreate  = runtime.ErrRuntimeCreate
	ErrAlreadyStopped = runtime.ErrAlreadyStopped
	ErrOperation      = runtime.ErrOperation
)

// Sentinel errors
var (
	ErrNullPointer        = runtime.ErrNullPointer
	ErrLibraryNotLoaded   = runtime.ErrLibraryNotLoaded
	ErrServerNotStarted   = runtime.ErrServerNotStarted
	ErrServerAlreadyStop  = runtime.ErrServerAlreadyStop
	ErrEmbeddedNotStarted = runtime.ErrEmbeddedNotStarted
)
```

## Complete Symbol Inventory

### chroma.go (from internal/runtime/chroma.go)
**Types (aliases):**
- `Server`
- `StartServerConfig`

**Functions (wrappers):**
- `Init(libPath string) error`
- `StartServer(config StartServerConfig) (*Server, error)`
- `Version() string`
- `VersionWithError() (string, error)`

### config.go (from internal/runtime/config.go)
**Types (aliases):**
- `ServerConfig`
- `ServerOption`

**Functions (wrappers):**
- `DefaultServerConfig() *ServerConfig`
- `NewServer(opts ...ServerOption) (*Server, error)`
- `WithPort(port int) ServerOption`
- `WithListenAddress(addr string) ServerOption`
- `WithMaxPayloadSize(bytes int) ServerOption`
- `WithCORSAllowOrigins(origins ...string) ServerOption`
- `WithPersistPath(path string) ServerOption`
- `WithSQLiteFilename(filename string) ServerOption`
- `WithAllowReset(allow bool) ServerOption`
- `WithOpenTelemetry(endpoint, serviceName string) ServerOption`
- `WithRawYAML(yaml string) ServerOption`

### embedded.go (from internal/runtime/embedded.go)
**Constants:**
- `DefaultTenantID`
- `DefaultDatabase`
- `DefaultEmbeddedDir`

**Types (aliases):**
- `Embedded`
- `StartEmbeddedConfig`
- `EmbeddedConfig`
- `EmbeddedOption`
- `EmbeddedCreateCollectionRequest`
- `EmbeddedCollection`
- `EmbeddedDatabase`
- `EmbeddedTenant`
- `EmbeddedCreateTenantRequest`
- `EmbeddedGetTenantRequest`
- `EmbeddedUpdateTenantRequest`
- `EmbeddedCreateDatabaseRequest`
- `EmbeddedListDatabasesRequest`
- `EmbeddedGetDatabaseRequest`
- `EmbeddedDeleteDatabaseRequest`
- `EmbeddedListCollectionsRequest`
- `EmbeddedGetCollectionRequest`
- `EmbeddedCountCollectionsRequest`
- `EmbeddedUpdateCollectionRequest`
- `EmbeddedDeleteCollectionRequest`
- `EmbeddedForkCollectionRequest`
- `EmbeddedAddRequest`
- `EmbeddedQueryRequest`
- `EmbeddedQueryResponse`
- `EmbeddedCountRecordsRequest`
- `EmbeddedGetRecordsRequest`
- `EmbeddedGetRecordsResponse`
- `EmbeddedUpdateRecordsRequest`
- `EmbeddedUpsertRecordsRequest`
- `EmbeddedDeleteRecordsRequest`
- `EmbeddedDeleteRecordsResponse`
- `EmbeddedIndexingStatusRequest`
- `EmbeddedIndexingStatusResponse`
- `EmbeddedHealthCheckResponse`

**Functions (wrappers):**
- `DefaultEmbeddedConfig() *EmbeddedConfig`
- `NewEmbedded(opts ...EmbeddedOption) (*Embedded, error)`
- `StartEmbedded(config StartEmbeddedConfig) (*Embedded, error)`
- `WithEmbeddedPersistPath(path string) EmbeddedOption`
- `WithEmbeddedSQLiteFilename(filename string) EmbeddedOption`
- `WithEmbeddedAllowReset(allow bool) EmbeddedOption`
- `WithEmbeddedRawYAML(yaml string) EmbeddedOption`

### errors.go (from internal/runtime/errors.go)
**Constants:**
- `Success` (int32)
- `ErrNullInput` (int32)
- `ErrInvalidUTF8` (int32)
- `ErrConfigParse` (int32)
- `ErrServerStart` (int32)
- `ErrInvalidHandle` (int32)
- `ErrRuntimeCreate` (int32)
- `ErrAlreadyStopped` (int32)
- `ErrOperation` (int32)

**Sentinel errors (var):**
- `ErrNullPointer`
- `ErrLibraryNotLoaded`
- `ErrServerNotStarted`
- `ErrServerAlreadyStop`
- `ErrEmbeddedNotStarted`

### backup.go (from internal/runtime/backup.go)
**Types (aliases):**
- `BackupMode`
- `BackupOptions`
- `ServerBackupOptions`
- `EmbeddedBackupOptions`
- `BackupOption` (interface)
- `BackupFileMetadata`
- `BackupManifest`

**Constants:**
- `BackupModeServer` (BackupMode)
- `BackupModeEmbedded` (BackupMode)

**Functions (wrappers):**
- `WithDestination(path string) BackupOption`
- `WithIncludeMetadata() BackupOption`
- `WithLeaveStopped() BackupOption`
- `WithLeaveClosed() BackupOption`

### rebuild.go (from internal/runtime/rebuild.go)
**Types (aliases):**
- `RebuildCollectionResult`
- `RebuildCollectionOption` (interface)

**Functions (wrappers):**
- `WithRebuildTenantID(tenantID string) RebuildCollectionOption`
- `WithRebuildDatabaseName(databaseName string) RebuildCollectionOption`
- `WithRebuildPrecheck() RebuildCollectionOption`
- `WithRebuildKeepBackup(keepBackup bool) RebuildCollectionOption`

### compaction.go (from internal/runtime/compaction.go)
**Types (aliases):**
- `CompactCollectionRequest`
- `CompactAllRequest`
- `CompactionCollectionResult`
- `CompactionResult`

(No package-level functions to wrap -- compaction methods are on Server/Embedded, forwarded automatically via type alias.)

### wal_prune.go (from internal/runtime/wal_prune.go)
**Types (aliases):**
- `WALPruneCollectionResult`
- `WALPruneResult`
- `WALPruneOption` (interface)

**Functions (wrappers):**
- `WithWALPruneTenantID(tenantID string) WALPruneOption`
- `WithWALPruneDatabaseName(databaseName string) WALPruneOption`
- `WithWALPruneDryRun() WALPruneOption`
- `WithWALPruneVacuum() WALPruneOption`
- `WithWALPruneMaxAge(maxAge time.Duration) WALPruneOption`
- `WithWALPruneMaxBytes(maxBytes uint64) WALPruneOption`
- `WithWALPruneWatermark(highBytes, lowBytes uint64) WALPruneOption`

### Summary Totals
| Category | Count |
|----------|-------|
| Type aliases | 54 |
| Wrapper functions | 37 |
| Constants (error codes) | 9 |
| Constants (embedded defaults) | 3 |
| Constants (backup modes) | 2 |
| Sentinel error variables | 5 |
| **Total explicit re-exports** | **110** |
| Methods (auto-forwarded via alias) | 67 |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `var F = runtime.F` (function variable) | `func F(...) { return runtime.F(...) }` (wrapper) | D-04 decision | Better godoc rendering; one extra stack frame (negligible) |
| Type definitions with method wrapping | Type aliases (`type X = Y`) | Go 1.9 (2017) | Zero-cost forwarding of all methods |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None (standard `go test`) |
| Quick run command | `go build ./...` (facade is compile-time only; no tests at root yet) |
| Full suite command | `go build ./... && go test ./internal/...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FACADE-01 | All types re-exported as aliases | compile-gate | `go build ./...` | N/A -- compilation is the test |
| FACADE-02 | All functions re-exported as wrappers | compile-gate | `go build ./examples/go/basic/` | Exists (examples/go/basic/main.go) |
| FACADE-03 | All constants and errors forwarded | compile-gate | `go build ./...` | N/A -- compilation is the test |
| FACADE-04 | Zero implementation logic | manual-only | `grep -rn 'if\|for\|switch\|select' *.go` (root files only) | N/A -- grep verification |
| FACADE-05 | Import path unchanged | compile-gate | `go build ./examples/go/basic/` | Exists (examples/go/basic/main.go) |

### Sampling Rate
- **Per task commit:** `go build ./...` (verifies compilation including examples)
- **Per wave merge:** `go build ./... && go test ./internal/...`
- **Phase gate:** `go build ./... && go build ./examples/go/basic/`

### Wave 0 Gaps
- None -- no test infrastructure needed. The existing `examples/go/basic/main.go` serves as the compilation gate. Test infrastructure (`compat_test.go`) is Phase 4 scope.

## Open Questions

1. **`time.Duration` import in wal_prune.go facade**
   - What we know: `WithWALPruneMaxAge` takes a `time.Duration` parameter. The facade wrapper needs to import `time` in addition to `internal/runtime`.
   - What's unclear: Nothing -- this is straightforward.
   - Recommendation: Add `import "time"` to `wal_prune.go` facade file. This is the only facade file needing a stdlib import beyond `internal/runtime`.

2. **Makefile `go test ./...` scope change**
   - What we know: The Makefile runs `go test -v ./...` which currently hits `internal/runtime/*_test.go`. After facade creation, `go test ./...` will also attempt to test the root package (which has no test files). This is fine -- `go test` reports `[no test files]` for packages without tests.
   - What's unclear: Nothing -- standard Go behavior.
   - Recommendation: No Makefile changes needed in Phase 3. Phase 4 addresses Makefile updates.

## Sources

### Primary (HIGH confidence)
- Go Language Specification - [Type declarations](https://go.dev/ref/spec#Type_declarations) -- type alias semantics
- Go Blog - [What's in an (Alias) Name?](https://go.dev/blog/alias-names) -- type alias design rationale
- Codebase `go doc ./internal/runtime/` output -- authoritative symbol list
- Codebase source files in `internal/runtime/` -- verified all exported symbols

### Secondary (MEDIUM confidence)
- Go Proposal [Type Aliases](https://go.googlesource.com/proposal/+/master/design/18130-type-alias.md) -- original design document

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- native Go features only, no libraries needed
- Architecture: HIGH -- pattern is well-established (type aliases since Go 1.9), all symbols inventoried from source
- Pitfalls: HIGH -- pitfalls are well-known Go gotchas (type definition vs alias), verified against spec

**Research date:** 2026-03-20
**Valid until:** Indefinite -- Go type alias semantics are stable language features
