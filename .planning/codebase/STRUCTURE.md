# Codebase Structure

**Analysis Date:** 2026-03-19

## Directory Layout

```
local-go-chroma/
├── shim/                          # Rust FFI shim
│   ├── src/lib.rs                 # C FFI boundary & runtime management
│   ├── Cargo.toml                 # Rust dependencies & build config
│   ├── benches/                   # Rust performance benchmarks
│   ├── tests/                     # Rust integration tests
│   └── target/                    # Build artifacts (gitignored)
├── java/                          # Java bindings scaffold
│   ├── core/                      # Shared Java API (ChromaRuntime, EmbeddedSession)
│   ├── jna/                       # JNA implementation (Java 17 fallback)
│   ├── panama/                    # Project Panama implementation (Java 22+)
│   ├── build.gradle.kts           # Multi-module Gradle config
│   └── settings.gradle.kts        # Gradle settings
├── examples/                      # Usage examples
│   └── go/basic/main.go           # Basic Go server startup example
├── scripts/                       # Build/release helper scripts
│   ├── backfill-r2.sh             # Release artifact backfill
│   ├── build_releases_index.sh    # Index release artifacts
│   └── dev-windows.ps1            # Windows development tasks
├── .github/                       # GitHub Actions CI/CD
├── .planning/                     # GSD planning documents
│
│── Go Package Root Files (library loading & FFI):
├── chroma.go                      # Core FFI bindings & Server type
├── library.go                     # Dynamic library loading logic
├── library_unix.go                # Platform-specific (Unix/Linux/macOS)
├── library_windows.go             # Platform-specific (Windows)
│
│── Configuration & Initialization:
├── config.go                      # ServerConfig & options (builder pattern)
├── errors.go                      # Error codes & mappings
│
│── Core Runtime Modes:
├── embedded.go                    # Embedded mode: types & operations
│
│── Database Operations:
├── backup.go                      # Backup/restore functionality
├── rebuild.go                     # Collection rebuild operations
├── compaction.go                  # Collection compaction operations
├── wal_prune.go                   # WAL (write-ahead log) pruning
│
│── Testing:
├── chroma_test.go                 # FFI initialization & basic tests
├── library_test.go                # Library loading tests
├── embedded_test.go               # Embedded mode integration tests
├── embedded_*_test.go             # Focused integration tests (6 files)
├── backup_test.go                 # Backup operation tests
├── rebuild_test.go                # Rebuild operation tests
├── compaction_test.go             # Compaction operation tests
├── wal_prune_test.go              # WAL prune operation tests
├── embedded_benchmark_test.go      # Performance benchmarks
│
│── Build & Configuration:
├── go.mod, go.sum                 # Go module dependencies
├── Makefile                       # Build system (Go, Rust, Java)
├── .golangci.yml                  # Go linter configuration
│
│── Documentation:
├── README.md                      # Project overview
├── CLAUDE.md                      # Claude AI instructions
├── AGENTS.md                      # Agent configuration
├── GO_API_SURFACE.md              # Public API documentation
├── JAVA_API_SURFACE.md            # Java API documentation
├── EMBEDDED_PARITY_MATRIX.md      # Feature parity matrix
│
│── Ignore & Metadata:
├── .gitignore                     # Version control ignores
├── .gitattributes                 # Line ending config

└── Test Data (gitignored):
    ├── chroma_test_data*/         # Generated during tests
```

## Directory Purposes

**shim/**
- Purpose: Rust FFI shim providing C-compatible symbols to Go/Java
- Contains: Rust source, Cargo build config, benchmarks, tests
- Key files: `shim/src/lib.rs` (all FFI implementations)

**java/**
- Purpose: Java language bindings and platform-specific FFI implementations
- Contains: Gradle multi-module project with core API, JNA bridge, Panama bridge
- Key files: `java/core/` (shared types), `java/jna/` (JNA bridge), `java/panama/` (Panama bridge)

**examples/**
- Purpose: Reference implementations showing library usage
- Contains: Go and Java examples demonstrating both server and embedded modes
- Key files: `examples/go/basic/main.go` (server startup with builder pattern)

**scripts/**
- Purpose: Helper scripts for build, release, and platform-specific tasks
- Contains: Release indexing, backfill tooling, Windows dev environment
- Key files: `dev-windows.ps1` (Windows build tasks)

**Root Go Files (FFI Boundary):**
- `chroma.go` - FFI function pointers, Init(), Server type, lifecycle
- `library.go` - Dynamic library loading logic and candidate path resolution
- `library_unix.go`, `library_windows.go` - Platform-specific library loading

**Root Go Files (Configuration):**
- `config.go` - ServerConfig struct, WithPort/WithPersistPath options, NewServer() builder
- `errors.go` - Error code constants and errorFromCode() mapping

**Root Go Files (Runtime Modes):**
- `embedded.go` - Embedded type, all database/collection/record operations

**Root Go Files (Operations):**
- `backup.go` - Backup/restore with manifest, checksum validation
- `rebuild.go` - Collection rebuild operations
- `compaction.go` - Collection compaction operations
- `wal_prune.go` - WAL pruning operations

## Key File Locations

**Entry Points:**

- `chroma.Init(libPath)`: Must call once to load Rust shim - `chroma.go` line 73
- `chroma.NewServer(opts)`: Start HTTP server - `config.go` line 142
- `chroma.StartServer(config)`: Direct server startup - `chroma.go` line 229
- `chroma.NewEmbedded(opts)`: Start embedded mode - `embedded.go` line 362
- `chroma.StartEmbedded(config)`: Direct embedded startup - `embedded.go` line 372

**Configuration:**

- Server config: `config.go` (ServerConfig type, WithPort/WithPersistPath options)
- Embedded config: `embedded.go` (EmbeddedConfig type, WithEmbeddedPersistPath options)
- Error mapping: `errors.go` (error code constants, errorFromCode function)
- Library loading: `library.go` (resolveLibraryLoadPlan, buildLibraryPathCandidates)

**Core Logic:**

- FFI binding: `chroma.go` (FFI function pointers, registerFunctions, callFFI* wrappers)
- Library loading: `library.go` (cross-platform library candidate resolution)
- Server lifecycle: `chroma.go` (Server.Stop, Server.Close, runtime finalizer)
- Embedded operations: `embedded.go` (Create/List/Get/Delete collections, CRUD records)
- Backup: `backup.go` (Backup, Restore with manifest and checksums)
- Rebuild: `rebuild.go` (RebuildCollection operations)
- Compaction: `compaction.go` (CompactCollection, CompactAll)
- WAL: `wal_prune.go` (PruneWALCollection, PruneWALAll)

**Testing:**

- FFI tests: `chroma_test.go` (Init, Version)
- Library loading tests: `library_test.go` (candidate resolution, path normalization)
- Embedded integration: `embedded_test.go` (full lifecycle)
- Focused integration: `embedded_*_test.go` (specific features - metadata validation, create/persist, properties, edges)
- Backup tests: `backup_test.go` (backup/restore scenarios)
- Rebuild tests: `rebuild_test.go` (rebuild operations)
- Compaction tests: `compaction_test.go` (compaction operations)
- WAL tests: `wal_prune_test.go` (WAL pruning)
- Benchmarks: `embedded_benchmark_test.go` (performance measurements)

## Naming Conventions

**Files:**

- Core types: `type_name.go` (e.g., `chroma.go`, `embedded.go`, `backup.go`)
- Tests: `package_name_test.go` (e.g., `embedded_test.go`, `library_test.go`)
- Focused tests: `feature_name_test.go` (e.g., `embedded_metadata_validation_test.go`)
- Platform-specific: `file_platform.go` (e.g., `library_unix.go`, `library_windows.go`)

**Directories:**

- Bindings: Named by language (`java/`, `shim/`)
- Examples: `examples/{language}/{feature}/`
- Scripts: Root-level `scripts/` for helpers
- Build output: `.golang/`, `target/` (gitignored)

**Go Types/Functions:**

- Exported (public): `PascalCase` (Server, Embedded, NewServer, Init)
- Private: `camelCase` (ffiMu, chromaServerStart)
- Constants: `UPPER_CASE` or `PascalCase` (Success, DefaultTenantID)
- Methods: `(s *Server) Method()` pattern with receiver shorthand (s, e, c)
- Options: `WithOption` convention for builder functions (WithPort, WithPersistPath)
- Request/Response types: `Embedded{Operation}{Request|Response}` pattern

## Where to Add New Code

**New Feature (Command/Query):**

1. Define request type: `type EmbeddedNewOperationRequest struct { ... }` in `embedded.go`
2. Define response type: `type EmbeddedNewOperationResponse struct { ... }` if needed
3. Implement Go method: `func (e *Embedded) NewOperation(request) (*Response, error)` in `embedded.go`
4. Implement Rust symbol: `pub extern "C" fn chroma_embedded_new_operation(...)` in `shim/src/lib.rs`
5. Register FFI pointer: Add `chromaEmbeddedNewOperation func(...)` and register in `chroma.go` registerFunctions()
6. Add tests: `embedded_new_operation_test.go` with unit/integration tests
7. Document: Update GO_API_SURFACE.md and EMBEDDED_PARITY_MATRIX.md

**New Configuration Option:**

1. Add field to ServerConfig/EmbeddedConfig in `config.go` or `embedded.go`
2. Add WithOption function: `func WithNewOption(value) ServerOption { ... }`
3. Update toYAML() to include new field in YAML output
4. Update DefaultServerConfig/DefaultEmbeddedConfig with sensible defaults
5. Document in CLAUDE.md and GO_API_SURFACE.md

**New Utility/Helper:**

- Shared helpers: `{name}_helper.go` (e.g., for validation, conversion)
- Test helpers: `{feature}_test.go` file with helper functions
- Platform-specific: `{file}_{platform}.go` pattern

**Utilities:**

- Shared helpers: Keep in `embedded.go` or create focused file like `manifest.go` for backup
- Test utilities: Co-locate with tests in `*_test.go` files
- Validation: Implement in Rust shim for type safety; Go validates pointers/codes only

## Special Directories

**shim/target/**
- Purpose: Rust build artifacts
- Generated: Yes (by `cargo build`)
- Committed: No (.gitignored)

**chroma_test_data*, chroma_test_data_*/**
- Purpose: Generated SQLite databases and files created during tests
- Generated: Yes (by Go tests via Embedded)
- Committed: No (.gitignored)

**.github/**
- Purpose: GitHub Actions CI/CD workflows
- Generated: No
- Committed: Yes

**.planning/codebase/**
- Purpose: GSD codebase analysis and planning documents
- Generated: Yes (by GSD agents)
- Committed: Yes (markdown docs)

---

*Structure analysis: 2026-03-19*
