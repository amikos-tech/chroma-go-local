# Architecture

**Analysis Date:** 2026-03-19

## Pattern Overview

**Overall:** FFI (Foreign Function Interface) wrapper architecture with multi-language bindings

**Key Characteristics:**
- Pure Go FFI (no cgo) using purego library to call Rust shim
- Rust FFI shim bridges Go/Java to Chroma core runtime
- Dual-mode runtime: Server (HTTP) and Embedded (in-process)
- Builder pattern for configuration with YAML backing
- Explicit resource lifecycle management with finalizer fallback

## Layers

**Go API Layer:**
- Purpose: User-facing Go package providing idiomatic Go API for Chroma
- Location: Root Go files (`chroma.go`, `config.go`, `embedded.go`, `errors.go`, `library.go`, `backup.go`, `rebuild.go`, `compaction.go`, `wal_prune.go`)
- Contains: Public types (Server, Embedded, config structs), public functions (NewServer, StartEmbedded, Init), request/response types
- Depends on: Rust shim FFI symbols (via purego), system FFI runtime
- Used by: User applications, examples, Java wrappers via Rust shim

**FFI Bridge Layer:**
- Purpose: Unsafe FFI binding to Rust shim; manages C string marshalling and error handling
- Location: `chroma.go` (FFI function pointers), `library.go` (dynamic library loading)
- Contains: Function pointer declarations, symbol registration, C string conversion, thread-safe FFI call wrappers
- Depends on: purego, Rust shim library (libchroma_shim.so/.dylib/.dll)
- Used by: Go API layer for all Rust communication

**Rust FFI Shim Layer:**
- Purpose: C FFI interface to Chroma backend; implements all exposed symbols with error handling
- Location: `shim/src/lib.rs`
- Contains: C-compatible exported functions (`chroma_*` symbols), Tokio runtime management, request/response serialization, panic catching
- Depends on: Chroma backend crates (chroma_config, chroma_frontend, chroma_log, chroma_system, etc.), tokio, serde_json
- Used by: Go layer via FFI, Java layer via FFI

**Java Scaffold Layer:**
- Purpose: Java bindings and API abstractions
- Location: `java/core/`, `java/jna/`, `java/panama/`
- Contains: ChromaRuntime interface, EmbeddedSession wrapper, JNA/Panama bridge implementations
- Depends on: Rust shim (via JNA or Panama), core interfaces
- Used by: Java applications

## Data Flow

**Server Startup Flow:**

1. User calls `chroma.NewServer(opts)` or `StartServer(config)` (Go API layer)
2. Go converts options to YAML config string via `ServerConfig.toYAML()`
3. Go creates C string from YAML and calls `chromaServerStartFromString()` via purego
4. Rust shim receives config, parses YAML, starts Tokio runtime, initializes Chroma frontend
5. Rust returns opaque `uintptr` handle back to Go
6. Go extracts port/address from Rust via `chromaServerPort()`, `chromaServerAddress()`
7. Go wraps handle in Server struct with RWMutex for state protection
8. Go attaches finalizer to Server for cleanup-on-GC fallback

**Embedded Mode Startup Flow:**

1. User calls `chroma.NewEmbedded(opts)` or `StartEmbedded(config)` (Go API layer)
2. Go converts options to YAML config string via `EmbeddedConfig.toYAML()`
3. Go creates C string and calls `chromaEmbeddedStartFromString()` via purego
4. Rust shim creates in-process Chroma frontend (no HTTP server)
5. Rust returns opaque `uintptr` handle to Go
6. Go wraps handle in Embedded struct with mutexes for state and backup coordination
7. Go attaches finalizer to Embedded for cleanup-on-GC fallback

**Operation Call Flow (Example: Query):**

1. User calls `embedded.Query(request)` on Go Embedded instance
2. Go serializes request struct to JSON via `json.Marshal()`
3. Go creates C string from JSON, calls `chromaEmbeddedQuery(handle, json_ptr)` with FFI lock
4. Rust receives handle and JSON, deserializes to Chroma types, executes query
5. Rust serializes response to JSON, allocates C string, returns pointer
6. Go reads C string (with mutex protection), deserializes to Go type via `json.Unmarshal()`
7. Go calls `chromaStringFree()` to free Rust-allocated C string memory
8. Go returns unmarshalled response to user

**Error Handling Flow:**

1. Rust detects error, calls `set_last_error(msg)` to store error in static LAST_ERROR Mutex
2. Rust returns error code (negative int32) or null pointer
3. Go detects error condition (return code < 0 or pointer == nil)
4. Go calls `chromaGetLastError()` to retrieve error message from LAST_ERROR
5. Go converts error code + message to Go error via `errorFromCode()`
6. Go calls `chromaStringFree()` to free error message string
7. Go returns error to user

**State Management:**

- **Server state:** Protected by `stateMu sync.RWMutex` in Server struct (gate access to port, addr, handle)
- **Backup operations:** Synchronized via `backupMu sync.Mutex` to prevent concurrent backup/operation conflicts
- **FFI calls:** Serialized via global `ffiMu sync.Mutex` to prevent race conditions in C string marshalling
- **Handle lifecycle:** Atomic swap/load on handle field to detect double-close and signal cleanup

## Key Abstractions

**Server Struct:**
- Purpose: Represents running HTTP server instance
- Examples: `chroma.go` lines 210-220
- Pattern: Opaque handle wrapper with explicit Close semantics; finalizer provides safety net; builder pattern configuration

**Embedded Struct:**
- Purpose: Represents in-process Chroma session
- Examples: `embedded.go` lines 24-32
- Pattern: Similar to Server but handles database/tenant/collection operations via method receivers

**Configuration Options:**
- Purpose: Fluent builder API for idiomatic Go config
- Examples: `config.go` (ServerOption functions), `embedded.go` (EmbeddedOption functions)
- Pattern: Functional options pattern; options accumulate into config, converted to YAML at startup

**Request/Response Types:**
- Purpose: Structured marshalling for Rust communication
- Examples: `EmbeddedCreateCollectionRequest`, `EmbeddedQueryRequest`, `EmbeddedQueryResponse` in `embedded.go`
- Pattern: JSON-serializable structs with `json` tags; support optional fields via pointers/omitempty; homogeneous array typing for metadata

## Entry Points

**Library Initialization:**
- Location: `chroma.Init(libPath)` in `chroma.go` line 73
- Triggers: Must be called once at application startup
- Responsibilities: Loads Rust shim library via purego, registers all FFI function pointers, handles platform-specific library names

**Server Mode:**
- Location: `chroma.NewServer(opts)` in `config.go` line 142 or `chroma.StartServer(config)` in `chroma.go` line 229
- Triggers: User wants HTTP server interface
- Responsibilities: Start HTTP server process, return Server handle for port/address queries and graceful shutdown

**Embedded Mode:**
- Location: `chroma.NewEmbedded(opts)` in `embedded.go` line 362 or `chroma.StartEmbedded(config)` in `embedded.go` line 372
- Triggers: User wants in-process mode
- Responsibilities: Initialize in-process Chroma frontend, provide database/collection/record operations

## Error Handling

**Strategy:** Error code convention (negative int32) with contextual error messages stored in Rust LAST_ERROR

**Patterns:**

- **Null return detection:** Go checks return pointers for nil; if nil, calls `chromaGetLastError()` to fetch context
- **Return code detection:** Go checks int32 return codes vs `Success` constant; negative codes trigger `errorFromCode()` lookup
- **Error wrapping:** Go uses `pkg/errors.Wrap()` and `fmt.Errorf()` with `%w` for error chain tracing
- **Panic safety:** Rust side catches panics via `AssertUnwindSafe` wrapper, converts to error code + message
- **Memory safety:** C strings allocated by Rust freed via `chromaStringFree()` to prevent leaks

## Cross-Cutting Concerns

**Logging:** No centralized logging in Go layer; Rust shim logs internally to stderr/files via Chroma backend

**Validation:** Input validation primarily in Rust (JSON deserialization, type constraints); Go validates YAML conversion and null pointers

**Authentication:** Not implemented at FFI layer; Chroma backend handles tenant/database/collection isolation in embedded mode; server mode delegates to HTTP auth (not in Go package scope)

---

*Architecture analysis: 2026-03-19*
