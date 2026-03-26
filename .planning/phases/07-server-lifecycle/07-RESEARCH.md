# Phase 7: Server Lifecycle - Research

**Researched:** 2026-03-26
**Domain:** Java FFI backend retrofit (JNA + Panama) with server lifecycle integration testing
**Confidence:** HIGH

## Summary

Phase 7 retrofits both JNA and Panama backends to extend `AbstractChromaRuntime` (from Phase 6) and wires `ServerSession` callback slots with real FFI calls. The existing backends already have complete `startServer()`, `serverStop()`, `serverFree()`, `serverPort()`, `serverAddress()`, and `serverPersistPath()` private methods -- the retrofit routes these through `AbstractChromaRuntime`'s template methods (`callFfiHandle`, `callFfiBorrowedString`, `callFfiVoid`) and replaces inline `lastError()`/`ensureOpen()` with the abstract base class implementations.

The Rust shim's `ServerHandle` struct stores `listen_address` and `persist_path` as `CString` fields. The `chroma_server_address()` and `chroma_server_persist_path()` FFI functions return **borrowed** pointers (`*const c_char`) valid until `chroma_server_free()` is called. The `chroma_server_port()` returns `i32` directly. These are not owned strings and must NOT be freed with `chroma_string_free()`. This ownership distinction is the single most important technical detail for the retrofit.

Integration tests start a real Chroma server via FFI, verify accessor values, exercise stop/close lifecycle, and cover error cases (double close, close-then-access, invalid config). Tests use ephemeral ports and temp directories to avoid conflicts.

**Primary recommendation:** Retrofit both backends to extend `AbstractChromaRuntime`, replacing all inline FFI patterns with the base class's lock-protected template methods, then write comprehensive integration tests that exercise real server lifecycle through FFI.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Full retrofit of both JNA and Panama backends to extend AbstractChromaRuntime -- not just server methods, ALL FFI calls (version, startEmbedded, startServer, stop, free, accessors) go through callFfi* template methods
- **D-02:** Both backends lose their inline `lastError()`, `ensureOpen()`, and manual lock-free FFI patterns. Replaced by AbstractChromaRuntime's global ReentrantLock + readLastError/readBorrowedString/readOwnedString abstract methods
- **D-03:** JNA backend implements abstract methods using `Pointer.getString(0)` for borrowed, `ptr.getString(0) + chroma_string_free` for owned, `chroma_get_last_error + string_free` for lastError
- **D-04:** Panama backend implements abstract methods using `MemorySegment.reinterpret(MAX_LEN).getString(0)` for borrowed, same + `chroma_string_free` for owned
- **D-05:** Real server tests -- start actual Chroma server via FFI, verify accessors, stop and close. Requires Rust shim built (same as Go test pattern)
- **D-06:** Ephemeral ports (port 0 or high random port) to avoid CI port conflicts
- **D-07:** Identical test structure in both `:jna:test` and `:panama:test` modules -- same test cases, same assertions
- **D-08:** Comprehensive error matrix in integration tests (happy path, invalid config, double close, close-then-access, port-already-bound, concurrent start)

### Claude's Discretion
- Exact test class naming and organization within JNA/Panama test dirs
- Whether to extract a shared test base class or duplicate test methods across backends
- Port selection strategy details (port 0 vs random high port)
- Order of retrofit tasks (JNA first vs Panama first vs parallel)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SRVR-01 | `ChromaRuntime.startServer(configYaml)` returns `ServerSession` in both JNA and Panama | Both backends already have working `startServer()` methods. Retrofit routes them through `callFfiHandle` from `AbstractChromaRuntime`. ServerSession constructor wiring pattern matches existing `EmbeddedSession` pattern. |
| SRVR-02 | `ServerSession` implements AutoCloseable with idempotent close and two-step teardown (stop + free) | `ServerSession` class already implements this in Phase 6 core module with `AtomicBoolean` close guard and stop-in-try/free-in-finally pattern. Backends wire `serverStop` and `serverFree` as lambda callbacks. |
| SRVR-03 | `ServerSession.port()`, `address()`, `url()` return server connection details | Accessor lambdas wired at construction time. Port uses `callFfiHandle` (returns int directly from `chroma_server_port`). Address and persist_path use `callFfiBorrowedString` (pointers valid until handle freed). `url()` is computed from `address()` + `port()` in ServerSession itself. |
| SRVR-04 | Integration tests verify server start, accessor values, stop, and close in both backends | Tests use `ServerConfigBuilder` to produce YAML, ephemeral ports via `ServerSocket(0)`, `@TempDir` for persist paths. Both JNA and Panama modules get identical test classes. |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **Conventional commits**: All commit messages must follow conventional commit format
- **No cgo**: Uses purego for Go FFI (not relevant to Java phase but important context)
- **Radically simple**: Keep architecture and implementation as simple as possible
- **Minimal comments**: Code and function/variable names should be self-explanatory
- **No shim changes**: Java reuses existing `chroma_*` FFI symbols -- no Rust shim modifications allowed
- **Testing**: `make test-java` runs `:jna:test` and `:panama:test` with `CHROMA_LIB_PATH` env var
- **Facade pattern**: Root package re-exports; zero implementation logic at root (Java doesn't have this pattern, but core module serves a similar shared-types role)

## Architecture Patterns

### Current Backend Structure (Pre-Retrofit)

Both `JnaChromaRuntime` and `PanamaChromaRuntime` currently:
1. Implement `ChromaRuntime` directly (not extending `AbstractChromaRuntime`)
2. Have their own inline `lastError(String fallback)` method
3. Have their own `ensureOpen()` / `AtomicBoolean closed` pattern
4. Have their own `AtomicBoolean` for close guard
5. Have complete server FFI method implementations that work but are not thread-safe

### Target Backend Structure (Post-Retrofit)

```
AbstractChromaRuntime (core)
├── FFI_LOCK (static ReentrantLock)
├── callFfiHandle(LongSupplier) → long
├── callFfiBorrowedString(LongSupplier) → String
├── callFfiVoid(Runnable)
├── callFfiJson(LongSupplier, Class<T>) → T
├── readBorrowedString(long) [abstract]
├── readOwnedString(long) [abstract]
└── readLastError() [abstract]

JnaChromaRuntime extends AbstractChromaRuntime
├── readBorrowedString → Pointer.getString(0)
├── readOwnedString → ptr.getString(0) + chroma_string_free(ptr)
├── readLastError → chroma_get_last_error + getString + string_free
├── version() → callFfiBorrowedString(bindings.chroma_version)
├── startEmbedded() → callFfiHandle(bindings.chroma_embedded_start_from_string)
├── startServer() → callFfiHandle(bindings.chroma_server_start_from_string)
└── close() → closed CAS + (no-op for JNA, no arena)

PanamaChromaRuntime extends AbstractChromaRuntime
├── readBorrowedString → segment.reinterpret(MAX_LEN).getString(0)
├── readOwnedString → segment.reinterpret(MAX_LEN).getString(0) + chroma_string_free
├── readLastError → chroma_get_last_error + reinterpret + getString + string_free
├── version() → callFfiBorrowedString(chromaVersion.invokeExact)
├── startEmbedded() → callFfiHandle(chromaEmbeddedStartFromString.invokeExact)
├── startServer() → callFfiHandle(chromaServerStartFromString.invokeExact)
└── close() → closed CAS + arena.close() (skip on Windows)
```

### ServerSession Wiring Pattern

```java
// In both backends, startServer() constructs ServerSession like this:
public ServerSession startServer(String configYaml) {
    // Validation already done before entering callFfiHandle
    long handle = callFfiHandle(() -> /* backend-specific FFI call */);
    return new ServerSession(
        handle,
        this::serverStop,    // LongConsumer: calls chroma_server_stop via callFfiVoid
        this::serverFree,    // LongConsumer: calls chroma_server_free (no error check)
        this::serverPort,    // LongToIntFunction: calls chroma_server_port
        this::serverAddress, // LongFunction<String>: calls chroma_server_address via callFfiBorrowedString
        this::serverPersistPath // LongFunction<String>: calls chroma_server_persist_path via callFfiBorrowedString
    );
}
```

### String Ownership in FFI Calls

**CRITICAL: This is the most important technical detail for the retrofit.**

| FFI Function | Return Type | Ownership | Java Action |
|---|---|---|---|
| `chroma_version()` | `*const c_char` | Borrowed (static) | `readBorrowedString` -- DO NOT free |
| `chroma_server_address(handle)` | `*const c_char` | Borrowed (handle-owned) | `readBorrowedString` -- DO NOT free |
| `chroma_server_persist_path(handle)` | `*const c_char` | Borrowed (handle-owned) | `readBorrowedString` -- DO NOT free |
| `chroma_get_last_error()` | `*const c_char` | **Owned** (caller must free) | `readOwnedString` -- MUST call `chroma_string_free` |
| `chroma_server_start_from_string(yaml)` | `*mut c_void` | Owned handle | Freed via `chroma_server_free` |

The `ServerHandle` struct in Rust stores `listen_address: CString` and `persist_path: CString`. The accessor functions return `server.listen_address.as_ptr()` and `server.persist_path.as_ptr()` -- these are pointers into the handle's memory, valid until `chroma_server_free()` drops the `Box<ServerHandle>`.

### Anti-Patterns to Avoid

- **Freeing borrowed strings**: Calling `chroma_string_free` on the pointer from `chroma_server_address()` or `chroma_server_persist_path()` will cause a double-free or use-after-free crash. These are borrowed pointers into the `ServerHandle`.
- **Accessing after free**: The `ServerSession.ensureOpen()` guard prevents calling accessor lambdas after close, which would dereference freed memory.
- **Lock contention on accessors**: Server accessors (port, address, persist_path) go through the FFI lock. This is correct because the Rust FFI is not thread-safe, but tests should not call accessors from multiple threads simultaneously expecting parallelism.
- **Skipping input validation before FFI**: Always validate `configYaml != null && !configYaml.isBlank()` before entering `callFfiHandle`. The Rust shim handles null gracefully but the Java layer should reject early.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| FFI lock serialization | Per-backend ReentrantLock | `AbstractChromaRuntime.FFI_LOCK` (static) | Single lock across all runtimes protects global error slot |
| String reading from native memory | Inline pointer-to-string conversions | `readBorrowedString` / `readOwnedString` abstract methods | Ownership semantics differ; template methods enforce correct handling |
| Error propagation from FFI | Manual null-check + lastError | `callFfiHandle` / `callFfiVoid` template methods | Consistent error handling, lock release guaranteed |
| Ephemeral port selection | Manual random port generation | `new ServerSocket(0)` + `getLocalPort()` | OS assigns guaranteed-free port |
| Temp directory management | Manual temp dir creation | JUnit `@TempDir` annotation | Auto-cleanup, unique per test |

## Common Pitfalls

### Pitfall 1: Wrong String Ownership for Server Accessors
**What goes wrong:** Calling `chroma_string_free()` on the pointer returned by `chroma_server_address()` or `chroma_server_persist_path()` causes memory corruption.
**Why it happens:** The existing backends (pre-retrofit) read these as borrowed strings without freeing. The retrofit must preserve this behavior using `readBorrowedString`, NOT `readOwnedString`.
**How to avoid:** Use `callFfiBorrowedString` for address and persist_path accessors. Only `chroma_get_last_error` returns owned strings that must be freed.
**Warning signs:** JVM crash (SIGSEGV) on the second call to an accessor, or crash during `chroma_server_free`.

### Pitfall 2: Port Race Condition in Tests
**What goes wrong:** `ServerSocket(0)` finds a free port, closes the socket, then the server tries to bind -- but another process grabbed the port in between.
**Why it happens:** TOCTOU race between port discovery and server startup.
**How to avoid:** Use high ephemeral ports and retry on bind failure. In practice, this race is rare with `ServerSocket(0)` on modern OS kernels. The existing test code uses this pattern and has been working.
**Warning signs:** Intermittent `ChromaException` with bind failure message in CI.

### Pitfall 3: ensureOpen Guard Must Come From AbstractChromaRuntime for Runtime, From Session for Session
**What goes wrong:** Confusing which `ensureOpen()` to use. The runtime's `ensureOpen()` checks if the runtime itself is closed. The `ServerSession.ensureOpen()` checks if the session is closed.
**Why it happens:** Both use `AtomicBoolean` + `IllegalStateException`, but they protect different resources.
**How to avoid:** `AbstractChromaRuntime` does NOT have `ensureOpen()` -- each backend keeps its own `closed` `AtomicBoolean` and `ensureOpen()` for runtime-level close checking. `ServerSession` has its own independent close guard.
**Warning signs:** Test passes even though session is closed, or runtime close prevents session operations.

### Pitfall 4: Panama MethodHandle.invokeExact Requires Exact Type Matching
**What goes wrong:** `MethodHandle.invokeExact()` throws `WrongMethodTypeException` if the argument types don't match exactly.
**Why it happens:** Panama's `invokeExact` is stricter than JNA's type coercion. Must pass `MemorySegment`, not `long`, to native functions.
**How to avoid:** Use `MemorySegment.ofAddress(handleAddress)` to convert `long` back to `MemorySegment` when calling FFI through MethodHandles. This is already the pattern in the existing code.
**Warning signs:** `WrongMethodTypeException` at runtime.

### Pitfall 5: Panama Windows Arena Close Workaround
**What goes wrong:** Calling `arena.close()` on Windows after the DLL is loaded causes JVM crash during process exit.
**Why it happens:** DLL unload order on Windows is not deterministic; the arena close triggers unloading of the native library while other finalizers may still reference it.
**How to avoid:** Preserve the existing `if (WINDOWS_OS) return;` guard in `PanamaChromaRuntime.close()`. The library stays loaded for process lifetime on Windows.
**Warning signs:** JVM crash on CI windows-latest runners during test cleanup.

### Pitfall 6: Server Port Returns int, Not Pointer
**What goes wrong:** Treating `chroma_server_port()` as a pointer-returning function when it directly returns `i32`.
**Why it happens:** All other accessors return pointers; port is the exception.
**How to avoid:** Use a dedicated `serverPort(long handle)` method that calls the FFI function and interprets the result as `int`. Check for negative values (error codes like `-1` for null handle). Do NOT use `callFfiBorrowedString` for port.
**Warning signs:** Garbage port values, or crash interpreting int as pointer.

## Code Examples

### JNA readBorrowedString / readOwnedString / readLastError

```java
// Source: Derived from existing JnaChromaRuntime patterns + AbstractChromaRuntime contract

@Override
protected String readBorrowedString(long address) {
    return new Pointer(address).getString(0);
}

@Override
protected String readOwnedString(long address) {
    Pointer ptr = new Pointer(address);
    try {
        return ptr.getString(0);
    } finally {
        bindings.chroma_string_free(ptr);
    }
}

@Override
protected String readLastError() {
    Pointer ptr = bindings.chroma_get_last_error();
    if (ptr == null || Pointer.nativeValue(ptr) == 0L) {
        return null;
    }
    try {
        return ptr.getString(0);
    } finally {
        bindings.chroma_string_free(ptr);
    }
}
```

### Panama readBorrowedString / readOwnedString / readLastError

```java
// Source: Derived from existing PanamaChromaRuntime patterns + AbstractChromaRuntime contract

@Override
protected String readBorrowedString(long address) {
    return MemorySegment.ofAddress(address).reinterpret(MAX_C_STRING_LEN).getString(0);
}

@Override
protected String readOwnedString(long address) {
    MemorySegment ptr = MemorySegment.ofAddress(address);
    try {
        return ptr.reinterpret(MAX_C_STRING_LEN).getString(0);
    } finally {
        try {
            chromaStringFree.invokeExact(ptr);
        } catch (Throwable t) {
            if (t instanceof Error error) throw error;
        }
    }
}

@Override
protected String readLastError() {
    try {
        MemorySegment ptr = (MemorySegment) chromaGetLastError.invokeExact();
        if (ptr.equals(MemorySegment.NULL)) return null;
        try {
            return ptr.reinterpret(MAX_C_STRING_LEN).getString(0);
        } finally {
            chromaStringFree.invokeExact(ptr);
        }
    } catch (Throwable t) {
        if (t instanceof Error error) throw error;
        return null;
    }
}
```

### Retrofitted startServer (JNA example)

```java
// Source: Combining existing JnaChromaRuntime.startServer() with AbstractChromaRuntime patterns

@Override
public ServerSession startServer(String configYaml) {
    ensureOpen();
    if (configYaml == null || configYaml.isBlank()) {
        throw new IllegalArgumentException("configYaml must be set");
    }
    long handle = callFfiHandle(
        () -> Pointer.nativeValue(bindings.chroma_server_start_from_string(configYaml)));
    return new ServerSession(
        handle,
        this::serverStop,
        this::serverFree,
        this::serverPort,
        this::serverAddress,
        this::serverPersistPath
    );
}
```

### Integration Test Pattern

```java
// Source: Existing JnaChromaRuntimeTest.serverLifecycleSmokeTest pattern, extended

@Test
void serverStartAccessorsStopClose(@TempDir Path persistDir) throws Exception {
    String libPath = System.getenv("CHROMA_LIB_PATH");
    Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

    int port = findFreePort();
    String yaml = new ServerConfigBuilder()
            .port(port)
            .listenAddress("127.0.0.1")
            .persistPath(persistDir.toAbsolutePath().toString())
            .allowReset(true)
            .build();

    try (var runtime = JnaChromaRuntime.init(libPath);
         ServerSession session = runtime.startServer(yaml)) {
        assertEquals(port, session.port());
        assertEquals("127.0.0.1", session.address());
        assertEquals("http://127.0.0.1:" + port, session.url());
        assertNotNull(session.persistPath());
    }
}

@Test
void doubleCloseIsIdempotent(@TempDir Path persistDir) throws Exception {
    // ... start server, close, close again -- no crash
}

@Test
void accessorsThrowAfterClose(@TempDir Path persistDir) throws Exception {
    // ... start server, close, then port()/address()/url() throw IllegalStateException
}
```

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | JUnit Jupiter 5.11.4 |
| Config file | `java/build.gradle.kts` (parent), per-module `build.gradle.kts` |
| Quick run command | `cd java && gradle --no-daemon :jna:test :panama:test` |
| Full suite command | `make test-java` (builds shim first, sets CHROMA_LIB_PATH) |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SRVR-01 | startServer returns ServerSession in both backends | integration | `cd java && CHROMA_LIB_PATH=... gradle --no-daemon :jna:test --tests "*ServerLifecycle*"` | Wave 0 |
| SRVR-02 | ServerSession.close() idempotent two-step teardown | integration | Same as above, double-close test case | Wave 0 |
| SRVR-03 | port(), address(), url() return correct values | integration | Same as above, accessor test cases | Wave 0 |
| SRVR-04 | Integration tests in both JNA and Panama | integration | `make test-java` | Wave 0 |

### Sampling Rate
- **Per task commit:** `cd java && gradle --no-daemon :core:test :jna:test :panama:test` (with CHROMA_LIB_PATH)
- **Per wave merge:** `make test-java` (includes shim build)
- **Phase gate:** `make test-java` green before verify

### Wave 0 Gaps
- [ ] `java/jna/src/test/java/.../jna/JnaServerLifecycleTest.java` -- covers SRVR-01 through SRVR-04 for JNA backend
- [ ] `java/panama/src/test/java/.../panama/PanamaServerLifecycleTest.java` -- covers SRVR-01 through SRVR-04 for Panama backend
- [ ] Existing smoke tests in `JnaChromaRuntimeTest` and `PanamaChromaRuntimeTest` must continue passing after retrofit

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Java | All Java code | Yes | 26 (JDK 26) | -- |
| Gradle | Build + test | No (standalone) | -- | `JAVA_GRADLE` env var or install Gradle 9+ |
| Rust toolchain | Building shim | No | -- | Pre-built shim binary or CI-only build |
| Rust shim (libchroma_shim.dylib) | Integration tests | No (not built) | -- | Must build with `make build` first (requires Rust) |

**Missing dependencies with no fallback:**
- Rust toolchain is needed to build the shim; without it integration tests cannot run locally. CI should have Rust installed. Code changes (the retrofit itself) can be done without Rust, but tests require the shim.
- Gradle is needed to compile and run Java tests. Must be installed or provided via `JAVA_GRADLE` env var.

**Missing dependencies with fallback:**
- None.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Each backend has inline FFI lock-free patterns | Shared `AbstractChromaRuntime` with `ReentrantLock` | Phase 6 (core types) | Backends become thread-safe via inheritance |
| Each backend implements `ChromaRuntime` directly | Backends extend `AbstractChromaRuntime` which implements `ChromaRuntime` | Phase 7 (this phase) | Eliminates duplicated lastError/ensureOpen/lock code |
| Manual YAML string construction in tests | `ServerConfigBuilder.build()` from core module | Phase 6 | Type-safe, validated config generation |

## Open Questions

1. **Port 0 behavior in Chroma server**
   - What we know: The Rust shim stores the configured port as `resolved_config.port`, which is the port from the YAML config, NOT necessarily the port the OS bound. If port 0 is passed, the actual bound port may differ from what `chroma_server_port()` reports.
   - What's unclear: Whether Chroma resolves port 0 to the actual bound port before storing it in `ServerHandle`.
   - Recommendation: Use `findFreePort()` (existing pattern) which finds a free port via `ServerSocket(0)`, closes the socket, then passes that specific port number to the config. This avoids the port-0 ambiguity entirely. HIGH confidence this works -- both existing backend tests already use this pattern.

2. **AbstractChromaRuntime ensureOpen() responsibility**
   - What we know: `AbstractChromaRuntime` does NOT provide `ensureOpen()`. Each backend maintains its own `closed` flag and `ensureOpen()`.
   - What's unclear: Whether `ensureOpen()` should be pulled into the abstract base class.
   - Recommendation: Keep `ensureOpen()` in each backend. The close semantics differ (JNA is a no-op; Panama closes the arena with Windows workaround). The abstract class provides FFI lock and string helpers only. HIGH confidence.

## Sources

### Primary (HIGH confidence)
- `shim/src/lib.rs` lines 166-172 (ServerHandle struct), 2848-2931 (server FFI functions) -- string ownership analysis
- `java/core/src/main/java/.../AbstractChromaRuntime.java` -- template method signatures
- `java/core/src/main/java/.../ServerSession.java` -- callback slot constructor, close semantics
- `java/jna/src/main/java/.../JnaChromaRuntime.java` -- current JNA backend (retrofit target)
- `java/panama/src/main/java/.../PanamaChromaRuntime.java` -- current Panama backend (retrofit target)
- `internal/runtime/chroma.go` -- Go reference implementation for server lifecycle + FFI lock pattern

### Secondary (MEDIUM confidence)
- Existing test files (`JnaChromaRuntimeTest.java`, `PanamaChromaRuntimeTest.java`) -- established test patterns
- `Makefile` -- `test-java` target wiring, `CHROMA_LIB_PATH` passing

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new libraries needed; all code uses existing Gradle dependencies (JNA 5.14.0, JUnit 5.11.4, Panama from JDK 22+)
- Architecture: HIGH -- both backends' current code is fully read and understood; AbstractChromaRuntime contract is well-defined from Phase 6
- Pitfalls: HIGH -- string ownership verified directly from Rust source; Panama Windows workaround confirmed from existing code
- Testing: HIGH -- existing smoke tests provide working patterns; `ServerConfigBuilder` available from Phase 6

**Research date:** 2026-03-26
**Valid until:** 2026-04-26 (stable domain -- FFI contract unlikely to change)
