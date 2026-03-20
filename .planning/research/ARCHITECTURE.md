# Architecture Patterns: Go Subtree Reorganization

**Domain:** Multi-language repo (Go + Rust FFI + Java bindings) — Go subtree move with import compatibility
**Researched:** 2026-03-20
**Overall confidence:** HIGH (Go language specs verified; patterns match official documentation)

---

## Current Architecture (Before Reorganization)

All Go implementation files live at the repo root alongside `shim/` (Rust) and `java/` directories.
The single `go.mod` at root declares `module github.com/amikos-tech/chroma-go-local`.
The root package name is `chroma`.

```
local-go-chroma/
├── go.mod                          # module github.com/amikos-tech/chroma-go-local
├── chroma.go                       # package chroma — FFI bindings, Server type, Init()
├── config.go                       # package chroma — ServerConfig, ServerOption, NewServer()
├── embedded.go                     # package chroma — Embedded type, all DB/collection/record ops
├── errors.go                       # package chroma — error constants and sentinel vars
├── library.go                      # package chroma — dynamic library loading
├── library_unix.go / _windows.go   # package chroma — platform-specific loading
├── backup.go                       # package chroma — Backup/Restore, BackupManifest
├── rebuild.go                      # package chroma — RebuildCollection
├── compaction.go                   # package chroma — CompactCollection, CompactAll
├── wal_prune.go                    # package chroma — PruneCollectionWAL, PruneAllWAL
├── *_test.go                       # package chroma / chroma_test
├── shim/                           # Rust FFI shim (unchanged by this milestone)
└── java/                           # Java scaffold bindings (unchanged)
```

**Problem:** Go implementation is mixed with Rust and Java artifacts at repo root. This makes
language ownership boundaries ambiguous and complicates long-term maintenance.

---

## Target Architecture (After Reorganization)

```
local-go-chroma/
├── go.mod                          # module github.com/amikos-tech/chroma-go-local (UNCHANGED)
│
│   ── Root Facade (thin, import-compatibility layer) ──
├── doc.go                          # package chroma — package-level doc comment
├── facade.go                       # package chroma — type aliases + var/func reexports
│
│   ── Go Implementation Subtree ──
├── go/
│   ├── internal/
│   │   ├── runtime/
│   │   │   ├── chroma.go           # package runtime — FFI bindings, Server, Init
│   │   │   ├── config.go           # package runtime — ServerConfig, ServerOption, NewServer
│   │   │   ├── embedded.go         # package runtime — Embedded, all ops
│   │   │   ├── errors.go           # package runtime — error codes, sentinel vars
│   │   │   ├── backup.go           # package runtime — Backup/Restore
│   │   │   ├── rebuild.go          # package runtime — RebuildCollection
│   │   │   ├── compaction.go       # package runtime — CompactCollection, CompactAll
│   │   │   └── wal_prune.go        # package runtime — WAL pruning
│   │   └── library/
│   │       ├── library.go          # package library — dynamic library loading
│   │       ├── library_unix.go     # package library — Unix-specific
│   │       └── library_windows.go  # package library — Windows-specific
│   └── (tests co-located with implementation)
│       ├── internal/runtime/*_test.go
│       └── internal/library/*_test.go
│
│   ── Existing Language Subtrees (unchanged) ──
├── shim/                           # Rust FFI shim
└── java/                           # Java scaffold bindings
```

**Single `go.mod` at root** — no new modules, no `replace` directives. The Go module path stays
`github.com/amikos-tech/chroma-go-local`. Consumers import exactly as before.

---

## Component Boundaries

### After Reorganization

| Component | Package Path | Responsibility | Communicates With |
|-----------|-------------|---------------|-------------------|
| Root facade | `github.com/amikos-tech/chroma-go-local` | Type aliases + delegating function vars for all exported names | Delegates to `go/internal/runtime`, `go/internal/library` |
| Runtime implementation | `github.com/amikos-tech/chroma-go-local/go/internal/runtime` | All Go logic: FFI bindings, Server/Embedded lifecycle, all operations | `library` package for shim loading; Rust shim via purego |
| Library loader | `github.com/amikos-tech/chroma-go-local/go/internal/library` | Dynamic `.so`/`.dylib`/`.dll` resolution and loading; platform-specific paths | Called by `runtime` during `Init()` |
| Rust shim | `shim/` (not a Go package) | C FFI symbols (`chroma_*`) linked via purego | Called by `runtime` package function pointers |
| Java scaffold | `java/` (not a Go package) | JNA + Panama bindings to Rust shim | Calls Rust shim directly, no dependency on Go packages |

**Key boundary rule:** Java and Rust layers have no dependency on any Go package — they link the
Rust shim library directly. The Go reorganization is purely internal to the Go module.

**`internal/` enforcement:** Go's compiler enforces that `go/internal/...` packages cannot be
imported by code outside the `github.com/amikos-tech/chroma-go-local` module. This means the
implementation subtree is protected from accidental external imports — only the root facade is
the public API.

---

## Facade Pattern: How Import Compatibility Works

The root package (`package chroma` at `go.mod` root) becomes a thin delegation layer. It contains
no logic — only aliases and delegating var assignments that point to the implementation package.

### What "thin facade" means in Go

Go does not have package aliases (the `proposal: allow packages to be aliased` issue #56611 is
unresolved as of Go 1.24). Instead, the facade must manually reexport each public symbol.

**Three reexport mechanisms:**

#### 1. Type aliases (preserve type identity — required for structs/interfaces)

```go
// facade.go (package chroma, at root)
import runtime "github.com/amikos-tech/chroma-go-local/go/internal/runtime"

type Server           = runtime.Server
type Embedded         = runtime.Embedded
type ServerConfig     = runtime.ServerConfig
type ServerOption     = runtime.ServerOption
type StartServerConfig = runtime.StartServerConfig
// ... all exported types
```

Type aliases (`type A = B`) preserve type identity. Code that uses `chroma.Server` after the
move continues to work with values returned from `runtime.Server` methods, because they are
the same type. Using type definitions (`type A B`) would break this — it creates a new, distinct
type.

#### 2. Const and var reexports (for error sentinels and numeric codes)

```go
// Numeric error code constants: must use const blocks
const (
    Success           = runtime.Success
    ErrNullInput      = runtime.ErrNullInput
    // ... all const int32 error codes
)

// Sentinel error vars: must use var declarations
var (
    ErrNullPointer        = runtime.ErrNullPointer
    ErrLibraryNotLoaded   = runtime.ErrLibraryNotLoaded
    ErrServerNotStarted   = runtime.ErrServerNotStarted
    ErrServerAlreadyStop  = runtime.ErrServerAlreadyStop
    ErrEmbeddedNotStarted = runtime.ErrEmbeddedNotStarted
)
```

Note: `errors.Is()` chains work correctly through `var` reexports because sentinel errors are
pointer-compared. Wrapping with `fmt.Errorf` in the facade would break `errors.Is()`.

#### 3. Function var delegation (for standalone functions)

```go
// Function vars — callable with same signature, zero overhead
var (
    Init             = runtime.Init
    Version          = runtime.Version
    VersionWithError = runtime.VersionWithError
    NewServer        = runtime.NewServer
    StartServer      = runtime.StartServer
    NewEmbedded      = runtime.NewEmbedded
    StartEmbedded    = runtime.StartEmbedded
    // builder option constructors
    WithPort                = runtime.WithPort
    WithListenAddress       = runtime.WithListenAddress
    WithMaxPayloadSize      = runtime.WithMaxPayloadSize
    WithPersistPath         = runtime.WithPersistPath
    WithSQLiteFilename      = runtime.WithSQLiteFilename
    WithAllowReset          = runtime.WithAllowReset
    WithCORSAllowOrigins    = runtime.WithCORSAllowOrigins
    WithOpenTelemetry       = runtime.WithOpenTelemetry
    WithRawYAML             = runtime.WithRawYAML
    DefaultServerConfig     = runtime.DefaultServerConfig
    // ... all exported functions
)
```

Functions assigned to `var` are first-class values in Go — callers use `chroma.Init(...)` and
it resolves to `runtime.Init(...)` with no semantic difference. The function signature seen by
callers is unchanged.

### What the facade does NOT need to reexport

- **Methods on types:** Methods travel with the type. Because `chroma.Server = runtime.Server`
  is a true alias, all methods on `Server` (e.g., `.Port()`, `.Stop()`, `.Backup()`) are
  automatically accessible through the alias — no wrapping needed.
- **Private symbols:** Everything lowercase stays in `runtime`/`library` packages. The facade
  only touches exported (uppercase) names.
- **`internal/library` details:** Library-loading types like `libraryCandidate` are all
  unexported — the facade has nothing to reexport from `library`.

### Full inventory of root facade reexports (this project)

**From `chroma.go` (runtime):**
- Types: `Server`, `StartServerConfig`
- Functions: `Init`, `StartServer`, `Version`, `VersionWithError`

**From `config.go` (runtime):**
- Types: `ServerConfig`, `ServerOption`
- Functions: `DefaultServerConfig`, `NewServer`, `WithPort`, `WithListenAddress`,
  `WithMaxPayloadSize`, `WithCORSAllowOrigins`, `WithPersistPath`, `WithSQLiteFilename`,
  `WithAllowReset`, `WithOpenTelemetry`, `WithRawYAML`

**From `embedded.go` (runtime):**
- Types: `Embedded`, `StartEmbeddedConfig`, `EmbeddedConfig`, `EmbeddedOption`,
  `EmbeddedCreateCollectionRequest`, `EmbeddedCollection`, `EmbeddedDatabase`,
  `EmbeddedTenant`, `EmbeddedCreateTenantRequest`, `EmbeddedGetTenantRequest`,
  `EmbeddedUpdateTenantRequest`, `EmbeddedCreateDatabaseRequest`,
  `EmbeddedListDatabasesRequest`, `EmbeddedGetDatabaseRequest`,
  `EmbeddedDeleteDatabaseRequest`, `EmbeddedListCollectionsRequest`,
  `EmbeddedGetCollectionRequest`, `EmbeddedCountCollectionsRequest`,
  `EmbeddedUpdateCollectionRequest`, `EmbeddedDeleteCollectionRequest`,
  `EmbeddedForkCollectionRequest`, `EmbeddedAddRequest`, `EmbeddedQueryRequest`,
  `EmbeddedQueryResponse`, `EmbeddedCountRecordsRequest`, `EmbeddedGetRecordsRequest`,
  `EmbeddedGetRecordsResponse`, `EmbeddedUpdateRecordsRequest`,
  `EmbeddedUpsertRecordsRequest`, `EmbeddedDeleteRecordsRequest`,
  `EmbeddedDeleteRecordsResponse`, `EmbeddedIndexingStatusRequest`,
  `EmbeddedIndexingStatusResponse`, `EmbeddedHealthCheckResponse`
- Functions: `NewEmbedded`, `StartEmbedded`, `DefaultEmbeddedConfig`,
  `WithEmbeddedPersistPath`, `WithEmbeddedSQLiteFilename`, `WithEmbeddedAllowReset`,
  `WithEmbeddedRawYAML`
- Constants: `DefaultTenantID`, `DefaultDatabaseID` (and any other exported consts)

**From `errors.go` (runtime):**
- Constants: `Success`, `ErrNullInput`, `ErrInvalidUTF8`, `ErrConfigParse`, `ErrServerStart`,
  `ErrInvalidHandle`, `ErrRuntimeCreate`, `ErrAlreadyStopped`, `ErrOperation`
- Vars: `ErrNullPointer`, `ErrLibraryNotLoaded`, `ErrServerNotStarted`, `ErrServerAlreadyStop`,
  `ErrEmbeddedNotStarted`

**From `backup.go` (runtime):**
- Types: `BackupMode`, `BackupOptions`, `ServerBackupOptions`, `EmbeddedBackupOptions`,
  `BackupOption` (interface), `BackupFileMetadata`, `BackupManifest`
- Functions: `WithDestination`, `WithIncludeMetadata`, `WithLeaveStopped`, `WithLeaveClosed`
- Constants: `BackupMode*` constants

**From `rebuild.go` (runtime):**
- Types: `RebuildCollectionResult`, `RebuildCollectionOption` (interface)
- Functions: `WithRebuildTenantID`, `WithRebuildDatabaseName`, `WithRebuildPrecheck`,
  `WithRebuildKeepBackup`

**From `compaction.go` (runtime):**
- Types: `CompactCollectionRequest`, `CompactAllRequest`, `CompactionCollectionResult`,
  `CompactionResult`

**From `wal_prune.go` (runtime):**
- Types: `WALPruneCollectionResult`, `WALPruneResult`, `WALPruneOption` (interface)
- Functions: `WithWALPruneTenantID`, `WithWALPruneDatabaseName`, `WithWALPruneDryRun`,
  `WithWALPruneVacuum`, `WithWALPruneMaxAge`, `WithWALPruneMaxBytes`,
  `WithWALPruneWatermark`

---

## Data Flow After Reorganization

The logical data flow is unchanged. The facade adds one transparent indirection layer:

```
User code
  import "github.com/amikos-tech/chroma-go-local"
    → chroma.Init(path)             [var delegation in facade.go]
      → runtime.Init(path)          [actual implementation]
        → library.resolveLoadPlan() [library package]
          → purego.OpenLibrary()    [loads libchroma_shim.dylib/.so/.dll]

    → chroma.NewServer(opts...)     [var delegation in facade.go]
      → runtime.NewServer(opts...)  [actual implementation]
        → builds ServerConfig → toYAML() → chromaServerStartFromString() via purego
          → Rust shim: chroma_server_start_from_string()
            → returns uintptr handle
          → runtime.Server{handle}  [returned to user via facade alias]
```

The facade indirection is resolved at compile time (function vars are direct function pointers,
not interface dispatch). No measurable runtime overhead.

---

## Build Order (Dependency Chain for Implementation)

The reorganization has a strict sequential dependency chain. Each step must be stable (tests
green) before the next begins.

### Step 1: Create `go/internal/library/` package

Move `library.go`, `library_unix.go`, `library_windows.go` into `go/internal/library/`.
Change `package chroma` to `package library`. Update all internal references.
Move `library_test.go` alongside.

**Why first:** `runtime` will import `library`. Establishing the leaf package first avoids
a partial state where `runtime` code references a package that doesn't exist yet.

**No circular deps risk:** `library` has no dependency on `runtime`.

### Step 2: Create `go/internal/runtime/` package

Move all remaining root `.go` files (chroma.go, config.go, embedded.go, errors.go, backup.go,
rebuild.go, compaction.go, wal_prune.go) into `go/internal/runtime/`. Change package declaration
to `package runtime`. Update `Init()` and related code to call `library.resolveLibraryLoadPlan()`
from the `library` import. Move all `*_test.go` files alongside.

**Why second:** Depends on `library` existing. After this step, `go test ./go/internal/...`
must pass.

### Step 3: Add root facade

Create `facade.go` (and optionally `doc.go`) at root with `package chroma`. Add all type
aliases, const reexports, var reexports, and function var delegations. The root package now
re-exports everything from `runtime` and `library` (where public).

**Why third:** The facade can only be written after both `runtime` and `library` packages are
stable, because it imports them. Writing it last avoids mid-refactor compilation failures in
the facade itself.

### Step 4: Test reorganization completeness

Run `go test ./...` from root — this exercises both the internal packages (via their own tests)
and any root-package tests. Specifically, `go vet ./...` and `golangci-lint run ./...` must
pass to confirm no unintended exported symbols were dropped.

A compatibility test file at root (e.g., `compat_test.go`) that imports and exercises all
exported symbols from `package chroma` provides an explicit regression gate.

### Step 5: Update Makefile and CI

Makefile targets that set `CHROMA_LIB_PATH` and invoke `go test` should be updated to point
at `./go/internal/...` (for implementation tests) and `./...` (for root-level compat tests).

### Step 6: Refresh docs and examples

`examples/go/basic/main.go` imports `github.com/amikos-tech/chroma-go-local` — this continues
to work unchanged after the facade is in place. Only internal cross-references in docs need
updating (e.g., file paths in CLAUDE.md, GO_API_SURFACE.md).

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Moving `go.mod` into `go/`

**What goes wrong:** Creates a separate module (`github.com/amikos-tech/chroma-go-local/go`)
requiring `require` + `replace` directives in all consumer `go.mod` files. Breaks the import
path entirely.

**Prevention:** Keep `go.mod` at the repo root. One module, one `go.mod`. The `go/` directory
is a regular package directory within the same module, not a separate module.

### Anti-Pattern 2: Using type definitions instead of type aliases in facade

**What goes wrong:** `type Server runtime.Server` creates a *new* type with no methods.
Code that calls `srv.Stop()` breaks. Assignment between `chroma.Server` and `runtime.Server`
requires explicit conversion.

**Prevention:** Always use `type A = B` (alias syntax) in the facade, not `type A B`
(definition syntax).

### Anti-Pattern 3: Wrapping error sentinel vars in `fmt.Errorf`

**What goes wrong:** `var ErrNullPointer = fmt.Errorf("wrapped: %w", runtime.ErrNullPointer)`
creates a new error value. `errors.Is(err, chroma.ErrNullPointer)` fails because it is not
the same pointer.

**Prevention:** Use direct assignment: `var ErrNullPointer = runtime.ErrNullPointer`. The
facade var and the runtime var point to the same underlying `errors.New(...)` value.

### Anti-Pattern 4: Introducing circular imports

**What goes wrong:** If `go/internal/runtime` imports from the root package (`package chroma`),
a circular dependency exists. The Go compiler rejects this.

**Prevention:** The dependency direction is strictly one-way:
`root facade → go/internal/runtime → go/internal/library → (external: purego, pkg/errors)`
The root package must never be imported by any internal subpackage.

### Anti-Pattern 5: Moving tests but not adjusting package declaration

**What goes wrong:** Tests using `package chroma_test` (external test package) import
`github.com/amikos-tech/chroma-go-local`. After moving to `go/internal/runtime/`, importing
the root module path from within `go/internal/runtime/` creates a circular dependency.

**Prevention:** Tests in `go/internal/runtime/` must use `package runtime_test` (or
`package runtime`) — not `package chroma_test`. The root-level `compat_test.go` is the place
for tests that validate the facade via `package chroma_test` imports.

---

## Scalability Considerations

This reorganization is a one-time structural change, not a scaling concern. However:

| Concern | Single `go/internal/runtime` package | Future: split `runtime` further |
|---------|-------------------------------------|----------------------------------|
| Compile speed | No change — same file count | Would improve incremental builds |
| Circular deps | Impossible (`internal` enforces) | Must re-check with each split |
| Facade maintenance | One facade file to update | One new alias per new export |

The project is not large enough (9 implementation files, ~13K LOC Go) to warrant further
splitting of `runtime` at this time. Keep it simple: two internal packages (`runtime`,
`library`) and one facade.

---

## Sources

- [Organizing a Go module — official Go documentation](https://go.dev/doc/modules/layout) — HIGH confidence
- [What's in an (Alias) Name? — Go Blog on type aliases](https://go.dev/blog/alias-names) — HIGH confidence
- [Alias declarations for Go — original design doc](https://go.googlesource.com/proposal/+/1487446b91599daa695905dc51a77d1bcc7086d8/design/16339-alias-decls.md) — HIGH confidence
- [proposal: allow packages to be aliased — golang/go #56611](https://github.com/golang/go/issues/56611) — HIGH confidence (confirms package-level aliasing not yet supported in Go)
- [Go Modules Guide for Monorepos — Grab Engineering](https://engineering.grab.com/go-module-a-guide-for-monorepos-part-1) — MEDIUM confidence
- Codebase analysis of `/Users/tazarov/experiments/amikos/local-go-chroma` — HIGH confidence (direct inspection)
