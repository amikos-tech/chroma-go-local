# Phase 7: Server Lifecycle - Context

**Gathered:** 2026-03-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement server start/stop/close and connection accessors in both JNA and Panama backends, with comprehensive integration tests. This includes retrofitting both backends to extend AbstractChromaRuntime for FFI safety. No new FFI symbols — reuses existing chroma_server_* exports.

</domain>

<decisions>
## Implementation Decisions

### AbstractChromaRuntime retrofit
- **D-01:** Full retrofit of both JNA and Panama backends to extend AbstractChromaRuntime — not just server methods, ALL FFI calls (version, startEmbedded, startServer, stop, free, accessors) go through callFfi* template methods
- **D-02:** Both backends lose their inline `lastError()`, `ensureOpen()`, and manual lock-free FFI patterns. Replaced by AbstractChromaRuntime's global ReentrantLock + readLastError/readBorrowedString/readOwnedString abstract methods
- **D-03:** JNA backend implements abstract methods using `Pointer.getString(0)` for borrowed, `ptr.getString(0) + chroma_string_free` for owned, `chroma_get_last_error + string_free` for lastError
- **D-04:** Panama backend implements abstract methods using `MemorySegment.reinterpret(MAX_LEN).getString(0)` for borrowed, same + `chroma_string_free` for owned

### Integration test strategy
- **D-05:** Real server tests — start actual Chroma server via FFI, verify accessors, stop and close. Requires Rust shim built (same as Go test pattern)
- **D-06:** Ephemeral ports (port 0 or high random port) to avoid CI port conflicts
- **D-07:** Identical test structure in both `:jna:test` and `:panama:test` modules — same test cases, same assertions

### Error handling coverage
- **D-08:** Comprehensive error matrix in integration tests:
  - Happy path: start → port/address/url → stop → close
  - Invalid config: null, empty, malformed YAML
  - Double close: idempotent, no crash
  - Close-then-access: every accessor throws IllegalStateException
  - Port-already-bound: attempt to start two servers on same port
  - Concurrent start attempts (if feasible with FFI lock)

### Claude's Discretion
- Exact test class naming and organization within JNA/Panama test dirs
- Whether to extract a shared test base class or duplicate test methods across backends
- Port selection strategy details (port 0 vs random high port)
- Order of retrofit tasks (JNA first vs Panama first vs parallel)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Core module contracts (Phase 6 output)
- `java/core/src/main/java/tech/amikos/chroma/local/core/AbstractChromaRuntime.java` — FFI lock + template methods; backends must extend this
- `java/core/src/main/java/tech/amikos/chroma/local/core/ServerSession.java` — Callback-slot session; backends construct with wired lambdas
- `java/core/src/main/java/tech/amikos/chroma/local/core/ChromaRuntime.java` — Interface with startServer(String) returning ServerSession
- `java/core/src/main/java/tech/amikos/chroma/local/core/ServerConfigBuilder.java` — Config builder for server YAML
- `java/core/src/main/java/tech/amikos/chroma/local/core/ChromaException.java` — Unchecked FFI exception type

### Backend implementations (retrofit targets)
- `java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java` — Current JNA backend; has full startServer impl but no AbstractChromaRuntime
- `java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java` — Current Panama backend; same pattern as JNA

### Rust FFI shim (symbol reference)
- `shim/src/lib.rs` — chroma_server_start_from_string, chroma_server_stop, chroma_server_free, chroma_server_port, chroma_server_address, chroma_server_persist_path

### Go reference implementation
- `internal/runtime/chroma.go` — Go FFI lock pattern (ffiMu), server start/stop/free wrappers
- `internal/runtime/config.go` — DefaultServerConfig() for test config generation

### Requirements
- `.planning/REQUIREMENTS.md` — SRVR-01 through SRVR-04

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `ServerSession` (core): Fully defined with callback slots — backends just wire lambdas at construction time
- `AbstractChromaRuntime` (core): Ready for backends to extend — provides callFfiHandle, callFfiJson, callFfiVoid, callFfiBorrowedString
- `ServerConfigBuilder` (core): Produces YAML for test configs — use `.builder().port(X).persistPath(tmpDir).build().toYaml()`
- Both backends already have working `startServer()`, `serverStop()`, `serverFree()`, `serverPort()`, `serverAddress()`, `serverPersistPath()` methods — retrofit is re-routing these through AbstractChromaRuntime, not rewriting from scratch

### Established Patterns
- `EmbeddedSession` constructor pattern: `new EmbeddedSession(handle.address(), this::embeddedFree)` — ServerSession follows same approach
- JNA: `Pointer.nativeValue(handle)` to get long from JNA Pointer
- Panama: `handle.address()` to get long from MemorySegment
- Both backends use `AtomicBoolean` for close guard — AbstractChromaRuntime will absorb the ensureOpen pattern

### Integration Points
- `JnaChromaRuntime.init(libraryPath)` and `PanamaChromaRuntime.init(libraryPath)` — factory methods unchanged, but class now extends AbstractChromaRuntime
- `Makefile` `test-java` target — runs `:jna:test` and `:panama:test`; integration tests must work with `CHROMA_LIB_PATH` env var
- Existing smoke tests in JNA/Panama modules — new server lifecycle tests coexist alongside them

</code_context>

<specifics>
## Specific Ideas

- The retrofit should preserve the existing `init(String libraryPath)` factory method signature — only the superclass changes
- Panama's Windows workaround in `close()` (skip arena.close on Windows due to DLL unload crash) must be preserved
- ServerSession's two-step close (stop in try, free in finally) is already implemented — integration tests just verify it works end-to-end with real FFI
- Use temp directories for persist paths in tests to avoid test pollution

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 07-server-lifecycle*
*Context gathered: 2026-03-26*
