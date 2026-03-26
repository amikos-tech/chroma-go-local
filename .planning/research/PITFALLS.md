# Pitfalls Research: Java API Surface Expansion (JNA + Panama)

**Domain:** Adding server lifecycle, builder configuration, and maintenance APIs to existing Java FFI wrappers (JNA + Panama dual-backend)
**Researched:** 2026-03-21
**Confidence:** HIGH (based on codebase analysis of existing Go patterns, Rust shim FFI contract, existing Java scaffold, and verified JNA/Panama documentation)

---

## Critical Pitfalls

### Pitfall 1: `chroma_get_last_error` Is a Global Mutex-Guarded Slot, Not Thread-Local

**What goes wrong:**
The Rust shim stores the last error in a `static LAST_ERROR: Mutex<Option<String>>`, not a thread-local. The Go side serializes all FFI calls through a single `ffiMu` mutex so `chroma_get_last_error()` is always read immediately after the failing call while the lock is held. If the Java side calls FFI functions concurrently from multiple threads without equivalent serialization, a race occurs: thread A's call fails, thread B's call overwrites `LAST_ERROR`, and thread A reads thread B's error message (or no message at all).

**Why it happens:**
The existing Java scaffold only calls `chroma_get_last_error` from `lastError()` in a sequential context (single `startEmbedded` call). Developers expanding the API might assume errors are thread-safe because the Java side wraps each call in try/catch. But the native error slot is shared state across all calls from all threads within the same process.

**How to avoid:**
Mirror the Go pattern: introduce a single Java-side lock (`ReentrantLock` or `synchronized`) that serializes ALL FFI calls. Every method that calls a native function must acquire this lock, make the FFI call, check the return value, read `chroma_get_last_error()` if needed, and only then release the lock. Both JNA and Panama implementations must use this lock.

Concrete implementation: add a `private final Object ffiLock = new Object()` to the runtime class. Every native call follows this pattern:
```java
synchronized (ffiLock) {
    Pointer result = bindings.chroma_server_start_from_string(yaml);
    if (result == null || Pointer.nativeValue(result) == 0L) {
        throw new ChromaException(lastError("server start failed"));
    }
    return result;
}
```

**Warning signs:**
- Intermittent wrong error messages in test output
- Error messages that don't match the operation that failed
- Tests that pass individually but fail when run in parallel

**Phase to address:**
Phase 1 (Server lifecycle). Must be established before adding any new FFI calls. Retrofit into existing `startEmbedded` and `version` methods simultaneously.

---

### Pitfall 2: Server Handle Memory Ownership — `chroma_server_stop` + `chroma_server_free` Are Separate Operations

**What goes wrong:**
The Rust shim has a two-step server teardown: `chroma_server_stop(handle)` sends the shutdown signal (returns error code), then `chroma_server_free(handle)` deallocates the `ServerHandle` struct (Box::from_raw). The Go side calls both in `Close()` with careful error handling: stop is attempted, its error is captured, free always runs, and the stop error is returned only if it wasn't `ErrAlreadyStopped`.

If the Java side only calls `chroma_server_free` without first calling `chroma_server_stop`, the Rust `ServerHandle._runtime` (tokio Runtime) is dropped while the server is still running. This may cause the tokio runtime to abort, potentially crashing the JVM. If Java calls `chroma_server_stop` but forgets `chroma_server_free`, the handle leaks.

**Why it happens:**
The existing `EmbeddedSession` only needs one call (`chroma_embedded_free`). Developers may copy the embedded pattern for server shutdown, missing the required two-step sequence.

**How to avoid:**
Create a `ServerSession` class (analogous to `EmbeddedSession`) that encodes the two-step sequence in its `close()` method:
```java
@Override
public void close() {
    if (closed.compareAndSet(false, true)) {
        try {
            stopAction.accept(handle);  // chroma_server_stop
        } finally {
            freeAction.accept(handle);  // chroma_server_free, always runs
        }
    }
}
```
The `ServerSession` constructor must receive both `stopAction` and `freeAction` separately to guarantee the free always runs even if stop fails.

**Warning signs:**
- JVM crashes on server shutdown (SIGSEGV from tokio runtime abort)
- Native memory growing over time when server sessions are created and destroyed
- CI hangs during server teardown (tokio runtime waiting for shutdown that never comes)

**Phase to address:**
Phase 1 (Server lifecycle). This is the first thing to get right — the `ServerSession` wrapper design.

---

### Pitfall 3: Server Address and Persist Path Are Borrow Pointers, Not Owned Strings

**What goes wrong:**
The Rust shim functions `chroma_server_address(handle)` and `chroma_server_persist_path(handle)` return `*const c_char` that point into the `ServerHandle`'s `CString` fields. The comments say: "The returned string is valid until the handle is freed." This means these pointers must NOT be passed to `chroma_string_free` — they are borrowed from the handle, not heap-allocated copies.

If the Java side calls `chroma_string_free` on the pointer returned by `chroma_server_address`, it corrupts the `ServerHandle`'s internal `CString` (double-free or use-after-free). The next access to the address field crashes the JVM.

Conversely, strings returned by `chroma_get_last_error()`, `chroma_embedded_rebuild_collection()`, etc. ARE heap-allocated and MUST be freed with `chroma_string_free`.

**Why it happens:**
The existing Java scaffold correctly handles this distinction for `chroma_version()` (static data, no free) and `chroma_get_last_error()` (heap-allocated, free required). But the server APIs introduce a third ownership category: borrowed from a handle. Without clear documentation of which functions return borrowed vs. owned pointers, developers will guess wrong.

**How to avoid:**
Document every FFI function's return pointer ownership in a comment at the binding declaration site. Establish a naming convention:
- Functions that return owned strings (must free): all `chroma_embedded_*` that return `*mut c_char` (JSON responses)
- Functions that return borrowed strings (must NOT free): `chroma_server_address`, `chroma_server_persist_path`, `chroma_embedded_persist_path`
- Functions that return static strings (must NOT free): `chroma_version`

In the Java wrapper, read borrowed strings into a Java `String` immediately and never store the native pointer:
```java
// CORRECT: copy string and discard pointer reference
MemorySegment ptr = (MemorySegment) chromaServerAddress.invokeExact(handle);
String address = ptr.reinterpret(MAX_C_STRING_LEN).getString(0);
// Do NOT call chromaStringFree on ptr
```

**Warning signs:**
- JVM crash (SIGSEGV) when accessing server address/persist path after it was "freed"
- Memory corruption that manifests later in unrelated operations
- Valgrind/ASAN reports showing double-free or heap-use-after-free on CString data

**Phase to address:**
Phase 1 (Server lifecycle). Must classify all FFI return value ownership before writing any JNA/Panama bindings.

---

### Pitfall 4: Maintenance APIs Require Embedded Handle, Not Server Handle

**What goes wrong:**
The Go side's server maintenance operations (backup, rebuild, compaction, WAL prune) follow a complex pattern: stop server -> start temporary embedded runtime -> run operation on embedded -> close embedded -> restart server. The Java side might try to call `chroma_embedded_rebuild_collection` directly with a server handle, which causes undefined behavior (wrong struct layout, memory corruption, JVM crash).

This is especially dangerous because both server and embedded handles are opaque `*mut c_void` pointers at the FFI level — the Rust shim does an unchecked `&*(handle as *const EmbeddedHandle)` cast. Passing a `ServerHandle` pointer where an `EmbeddedHandle` is expected interprets memory at wrong offsets.

**Why it happens:**
The handle types are type-erased to `void*` / `long` / `MemorySegment` at the FFI boundary. Java has no compile-time enforcement that a handle belongs to the correct runtime type. A developer implementing `Server.rebuildCollection()` might directly pass the server handle to `chroma_embedded_rebuild_collection`.

**How to avoid:**
The Java `Server` class must implement maintenance operations using the same stop-embedded-restart pattern as Go:
1. Save current config
2. Stop server (`chroma_server_stop` + `chroma_server_free`)
3. Start embedded (`chroma_embedded_start_from_string`)
4. Run maintenance on embedded handle
5. Close embedded (`chroma_embedded_free`)
6. Restart server (`chroma_server_start_from_string`)
7. Update server handle

This logic lives entirely in Java — no new Rust shim functions needed. Wrap this in a `runMaintenance(Consumer<EmbeddedSession>)` template method.

**Warning signs:**
- JVM crash (SIGSEGV) when calling maintenance APIs on server
- Wrong field values read from handle (port interpreted as a tokio Runtime pointer)
- Tests that pass with embedded but crash with server

**Phase to address:**
Phase 2 (Maintenance APIs). This is the single most important architectural decision for the Java server maintenance API.

---

### Pitfall 5: Panama Arena Lifetime vs. Multi-Handle Scenarios

**What goes wrong:**
The existing `PanamaChromaRuntime` uses a single `Arena.ofShared()` that owns the library lookup. The runtime's `close()` calls `arena.close()`, which invalidates the library symbols. If a server session or embedded session outlives the runtime (i.e., the user calls `runtime.close()` before closing all sessions), subsequent FFI calls through the now-invalidated `MethodHandle`s produce `IllegalStateException` at best, or undefined behavior at worst.

With server maintenance APIs, the problem compounds: a maintenance operation that stops and restarts the server must create a temporary embedded session. If this happens while the runtime's arena is being closed from another thread, the new embedded session's FFI calls operate on invalid memory.

**Why it happens:**
The existing `PanamaChromaRuntime.close()` already has a comment about this: "ensure all EmbeddedSession instances are closed first." But there's no enforcement mechanism — the runtime doesn't track live sessions.

**How to avoid:**
Two approaches, use option A for simplicity:

**Option A (simple):** Keep the existing pattern where `runtime.close()` fails if sessions are live. Document this clearly. The `ServerSession` and `EmbeddedSession` close actions call through MethodHandles that are valid as long as the runtime's arena is alive. This is sufficient because:
- Sessions hold `long` handle values, not `MemorySegment` references into the arena
- MethodHandle invocation only needs the arena to be alive for the downcall stub
- Users must close sessions before runtime (already documented)

**Option B (defensive):** Use reference counting. The runtime tracks live sessions via `AtomicInteger`. `close()` throws if count > 0. Session creation increments, session close decrements.

**Warning signs:**
- `IllegalStateException: already closed` from Panama arena during FFI calls
- JVM crash when calling server methods after runtime close
- Tests that pass with a single session but fail when multiple sessions overlap

**Phase to address:**
Phase 1 (Server lifecycle). Decide on session tracking strategy before adding server sessions.

---

### Pitfall 6: Builder Config Generates YAML That Must Match Rust's Figment Parser Exactly

**What goes wrong:**
The Go side generates YAML config strings (in `ServerConfig.toYAML()` and `EmbeddedConfig.toYAML()`) that are parsed by Rust's Figment YAML parser. The Java builder must generate byte-for-byte compatible YAML. Subtle differences cause failures:
- Go uses `fmt.Fprintf(&b, "port: %d\n", c.Port)` — Java must produce identical key names
- YAML quoting: Go uses `%q` for strings (adds double quotes), Java must match
- Boolean format: Go uses `%t` (produces `true`/`false`), not `True`/`False`
- Missing fields: if Java omits a field the Go side always includes, Figment may apply different defaults
- Extra fields: if Java includes a field Figment doesn't recognize, it may fail or silently ignore it

**Why it happens:**
There's no schema validation at the Java level. The YAML is a free-form string passed to the Rust FFI. The Figment parser is lenient about unknown fields but strict about types. A Java builder that produces `port: "8000"` (string) instead of `port: 8000` (integer) causes a parse error that surfaces as a `ChromaException` with an unhelpful error message.

**How to avoid:**
- Mirror the Go YAML generation exactly. Copy the field names and format strings from `config.go` lines 109-138.
- Write a test that generates YAML from the Java builder and passes it through `chroma_server_start_from_string` to verify it's accepted.
- Include a "golden YAML" test: hardcode the expected YAML output for a known configuration and assert string equality.
- Never use a YAML library to generate the config — manual string building (like Go does) is more predictable and avoids library-specific formatting differences.

**Warning signs:**
- `ChromaException: config parse error` when starting server with builder-generated YAML
- Server starts but on wrong port or with wrong settings
- Works with `WithRawYAML()` but fails with builder methods

**Phase to address:**
Phase 1 (Server lifecycle builder). The first test should verify that builder-generated YAML starts a server successfully.

---

### Pitfall 7: Dual Backend Drift — JNA and Panama Implementations Diverge

**What goes wrong:**
Every new API method must be implemented identically in both `JnaChromaRuntime` and `PanamaChromaRuntime`. As the API surface grows from 3 methods (version, startEmbedded, close) to ~15+ methods (server lifecycle, maintenance APIs, option builders), the risk of behavioral divergence increases:
- One backend forgets to free a returned string
- One backend misses a null check
- Error handling logic differs between backends
- One backend serializes FFI calls, the other doesn't
- Method signatures drift (different parameter names, types, or ordering)

**Why it happens:**
There is no shared abstract base class or template method pattern in the current design. `JnaChromaRuntime` and `PanamaChromaRuntime` independently implement `ChromaRuntime`. Each new feature is a manual copy-paste across backends with backend-specific pointer handling. As the count of methods grows, the copy-paste becomes error-prone.

**How to avoid:**
Extract shared logic into the `core` module:
1. **Option resolution and validation** in `core`: builders, request objects, YAML generation — pure Java, no FFI. Both backends use the same builder classes.
2. **Session wrappers** in `core`: `ServerSession`, `EmbeddedSession` — hold handle as `long`, delegate close to a `LongConsumer`. Already done for `EmbeddedSession`.
3. **Template methods** for maintenance operations in `core`: the stop-embedded-restart pattern is pure orchestration logic that doesn't need backend-specific code.
4. **Backend-specific code** limited to: library loading, MethodHandle/JNA binding declarations, and raw FFI call wrappers that convert between `Pointer`/`MemorySegment` and `long`.

Write a contract test suite in `core` that takes a `ChromaRuntime` instance and runs all operations. Both backend test suites instantiate their runtime and delegate to this shared test suite.

**Warning signs:**
- Tests pass for JNA but fail for Panama (or vice versa)
- Different error messages from the two backends for the same failure
- One backend leaks memory while the other doesn't
- Code reviews that must compare two large files side-by-side

**Phase to address:**
Phase 1 (before adding new APIs). Refactor `core` to hold shared logic before expanding the API surface.

---

## Moderate Pitfalls

### Pitfall 8: Server Restart After Maintenance Changes Handle Value

**What goes wrong:**
When a Java `Server` performs a maintenance operation (backup, rebuild, compaction, WAL prune), it must stop the server, run the operation on a temporary embedded session, and restart the server. The restarted server gets a NEW handle from the Rust shim (`chroma_server_start_from_string` returns a fresh `*mut c_void`). If the `ServerSession` stores the handle as a `final long`, the old handle value is stale after restart. Any code that cached the old handle will use a freed pointer.

**Why it happens:**
The Go side handles this with `atomic.SwapUintptr` and `sync.RWMutex` on the Server struct. Java's `EmbeddedSession` has a `final long handle` — this immutability pattern doesn't work for a server handle that changes on restart.

**How to avoid:**
Design the `ServerSession` differently from `EmbeddedSession`:
- Use `AtomicLong` for the handle instead of `final long`
- The maintenance template method atomically swaps in the new handle after restart
- All methods that access the handle read from `AtomicLong` and check for 0L (closed)
- Alternatively, use a `Server` wrapper in `core` that holds a mutable reference to the underlying session, replacing it on restart

**Warning signs:**
- JVM crash after a maintenance operation (using freed server handle)
- Server methods returning stale port/address after restart
- `ConcurrentModificationException` or `IllegalStateException` from handle access during maintenance

**Phase to address:**
Phase 1 (Server lifecycle design). The handle mutability decision drives the entire `ServerSession` class design.

---

### Pitfall 9: Maintenance Operations Must Be Serialized Like Go's `backupMu`

**What goes wrong:**
The Go side serializes all maintenance operations (backup, rebuild, compaction, WAL prune) with `s.backupMu.Lock()` and documents: "Backups, compaction, rebuild, and WAL prune are mutually exclusive and serialized." If the Java side doesn't enforce mutual exclusion, two concurrent maintenance calls both stop the server, both try to start embedded runtimes (only one succeeds because the persist directory is locked by the first), and the second fails with an unclear error. Worse, both callers may try to restart the server, causing port conflicts.

**Why it happens:**
Java's `synchronized` block is per-instance, which is correct here (one lock per server instance). But developers might forget to synchronize maintenance methods or use different locks for different operations.

**How to avoid:**
Use a single `ReentrantLock` (or `synchronized` on a dedicated lock object) in the `Server` class that serializes ALL maintenance operations:
```java
private final ReentrantLock maintenanceLock = new ReentrantLock();

public RebuildResult rebuildCollection(String name, RebuildOption... options) {
    maintenanceLock.lock();
    try {
        return runMaintenance(embedded -> embedded.rebuildCollection(name, options));
    } finally {
        maintenanceLock.unlock();
    }
}
```

**Warning signs:**
- Port-in-use errors when two maintenance operations run concurrently
- "Server already stopped" errors from the second concurrent caller
- Embedded runtime start failures due to persist directory locking

**Phase to address:**
Phase 2 (Maintenance APIs). Establish the serialization lock when adding the first maintenance operation.

---

### Pitfall 10: JSON Request/Response Marshaling Must Match Go's Encoding Exactly

**What goes wrong:**
Maintenance APIs (rebuild, compaction, WAL prune) pass JSON-encoded request structs through the FFI boundary as C strings, and receive JSON-encoded response structs back. The Java side must produce JSON that matches the Go side's `json.Marshal` output field names exactly. Mismatches:
- Go uses `json:"tenant_id,omitempty"` — field name is `tenant_id`, omitted when empty
- Java's Jackson/Gson might produce `tenantId` (camelCase) by default
- `uint64` pointer fields in Go (e.g., `*uint64` for `MaxAgeSeconds`) become `null` in JSON when nil — Java must handle nullable `Long` correctly
- The Rust deserializer (serde) is strict about field types: sending `"dry_run": "true"` (string) instead of `"dry_run": true` (boolean) fails

**Why it happens:**
Go and Java have different JSON serialization defaults. Go uses struct tags with snake_case by convention. Java uses camelCase by default.

**How to avoid:**
- Define request/response POJOs in the `core` module with explicit JSON field name annotations: `@JsonProperty("tenant_id")`
- Use Jackson with `FAIL_ON_UNKNOWN_PROPERTIES = false` for response deserialization (Rust may add fields in future)
- Write golden tests: serialize a Java request object and compare against the known JSON that Go produces
- For optional fields, use `@JsonInclude(Include.NON_NULL)` to match Go's `omitempty` behavior
- Alternatively, avoid a JSON library dependency entirely: use simple string concatenation for requests (like Go's `marshalRequestJSON`) and manual parsing for responses

**Warning signs:**
- `chroma_embedded_rebuild_collection` returns NULL with `LAST_ERROR = "unknown field tenantId"`
- Operations succeed but with wrong parameters (e.g., wrong tenant because field name was ignored)
- Response parsing exceptions from Jackson/Gson on valid Rust responses

**Phase to address:**
Phase 2 (Maintenance APIs). Decide on JSON strategy before implementing the first maintenance operation.

---

### Pitfall 11: `chroma_string_free` Called on Wrong Pointer Type Corrupts Heap

**What goes wrong:**
In the Rust shim, `chroma_string_free` does `CString::from_raw(s as *mut c_char)` which takes ownership and drops the CString. If called on:
- A static string (from `chroma_version`): corrupts read-only data
- A borrowed pointer (from `chroma_server_address`): double-free when the handle is later freed
- A NULL pointer: the Rust code guards against this, but the Java side should still check
- An already-freed pointer: use-after-free

The new APIs return many string pointers with different ownership. As the API surface grows, the probability of calling `chroma_string_free` on the wrong pointer increases.

**Why it happens:**
All native pointers look the same in Java (`Pointer` in JNA, `MemorySegment` in Panama). There's no type system enforcement of ownership.

**How to avoid:**
Create two distinct wrapper methods in each backend:
```java
// For owned strings (must free after reading)
private String readOwnedString(Pointer ptr, String context) {
    if (ptr == null) throw new ChromaException(lastError(context));
    try {
        return ptr.getString(0);
    } finally {
        bindings.chroma_string_free(ptr);
    }
}

// For borrowed strings (must NOT free)
private String readBorrowedString(Pointer ptr, String context) {
    if (ptr == null) throw new ChromaException(context + " returned NULL");
    return ptr.getString(0);
}
```

Every FFI call site uses the appropriate wrapper. Code review must verify which wrapper is used.

**Warning signs:**
- JVM crash (SIGSEGV) in `chroma_string_free` or later in unrelated code
- ASAN/Valgrind reports on the native side
- Intermittent crashes that only reproduce under memory pressure

**Phase to address:**
Phase 1 (Server lifecycle). Establish the ownership wrappers before adding server APIs.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Copy-paste between JNA and Panama | Faster initial implementation | Behavioral drift, doubled bug surface, maintenance burden | Never acceptable for more than 3-5 methods |
| Skip FFI call serialization | Simpler code, no lock contention | Race conditions on error slot, potential JVM crashes | Never — Rust shim's LAST_ERROR is global |
| Use raw YAML strings instead of builders | Works immediately, no validation needed | Users must know YAML schema, no IDE completion, easy mistakes | Acceptable for power users as an escape hatch, but builders should be primary |
| Skip Panama backend temporarily | Ship JNA-only faster | Panama users blocked, hard to retrofit, may never catch up | Only if explicitly scoped as "JNA first, Panama follows in next milestone" |
| Inline JSON construction with string concat | No library dependency | Quoting bugs, injection risk, unmaintainable for complex requests | Acceptable for simple requests (1-3 fields), not for WAL prune options |
| Use `synchronized` on the runtime instance | Simple, familiar | Blocks version() and other safe reads during maintenance | Acceptable if maintenance operations are infrequent; use ReadWriteLock if contention matters |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Server handle + port/address | Reading port/address before server is fully started | Read port and address immediately after `chroma_server_start_from_string` returns, inside the FFI lock, before returning the session |
| Backup API | Forgetting that backup stops the server, making it temporarily unavailable | Document that `Server.backup()` is blocking and the server is stopped during backup; hold maintenance lock |
| Embedded maintenance on server | Passing server handle to embedded FFI functions | Always create temporary embedded session for server maintenance; never mix handle types |
| Config YAML generation | Using a YAML library that adds document markers (`---`) or trailing newlines differently | Use manual string building matching Go's `fmt.Fprintf` patterns exactly |
| Panama confined arena for call-scoped strings | Using the runtime's shared arena for call-scoped allocations (parameter strings) | Use `Arena.ofConfined()` in try-with-resources for each FFI call's parameter allocation (existing pattern in `startEmbedded` is correct) |
| JNA string parameter encoding | Not null-terminating strings or using wrong encoding | JNA auto-converts `String` parameters to null-terminated C strings; verify UTF-8 encoding |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Single FFI lock blocks all operations | High latency on read-only calls (version, port) during maintenance | Use ReadWriteLock: read lock for queries, write lock for state-changing ops | Under concurrent access from multiple threads |
| Creating new Arena per FFI call in Panama | GC pressure from arena allocation/deallocation | Reuse the runtime's shared arena for MethodHandle lookups; only use confined arenas for parameter marshaling | High-throughput scenarios (unlikely for this use case) |
| JSON serialization/deserialization overhead | Latency on maintenance operations | Use a pre-allocated ObjectMapper (Jackson) or avoid library entirely | Irrelevant — maintenance ops are infrequent and dominated by native execution time |
| Holding FFI lock during server restart | Entire runtime blocked for seconds during maintenance | Accept this — Go has the same limitation; document it | When maintenance operations are triggered frequently in production |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Logging raw YAML config containing file paths | Exposes filesystem structure in logs | Only log config presence, not content; never include YAML in exception messages |
| Accepting arbitrary YAML from untrusted input | YAML injection if config is user-provided | Builder pattern sanitizes inputs; `WithRawYAML` should be documented as "trusted input only" |
| Not validating persist path for path traversal | Writing data outside intended directory | Validate persist path is absolute and within expected parent (or delegate to Rust shim which does this) |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Server `close()` doesn't report if it was already closed | User can't tell if close succeeded or was redundant | Return silently on double-close (match Go's idempotent Close behavior), but log a debug warning |
| Builder has no validation until `build()` is called | User sets invalid port (-1) and only discovers at runtime | Validate in `build()` before generating YAML; throw `IllegalArgumentException` with specific field name |
| Maintenance operations silently restart the server | User doesn't know the server was temporarily unavailable | Return a result object that includes `serverRestarted: true` and `downtimeMs` |
| No way to check if server is running | User calls maintenance on a stopped server, gets confusing error | Add `isRunning()` method that checks handle liveness |

## "Looks Done But Isn't" Checklist

- [ ] **Server lifecycle:** Often missing `chroma_server_stop` before `chroma_server_free` -- verify close() calls both in correct order
- [ ] **Server address/port:** Often missing ownership classification -- verify borrowed pointer is NOT passed to `chroma_string_free`
- [ ] **Maintenance APIs:** Often missing the embedded-passthrough pattern for server mode -- verify server maintenance creates temporary embedded session
- [ ] **Maintenance serialization:** Often missing mutual exclusion between concurrent maintenance ops -- verify single lock guards all maintenance methods
- [ ] **Builder YAML:** Often missing field name match against Go -- verify generated YAML field names match `config.go` exactly
- [ ] **JSON request fields:** Often missing snake_case annotation -- verify `@JsonProperty("tenant_id")` not `tenantId`
- [ ] **Dual backend parity:** Often missing one backend's implementation -- verify every `ChromaRuntime` method has both JNA and Panama implementations
- [ ] **String ownership:** Often missing free call on owned strings or adding free call on borrowed strings -- verify each FFI return's ownership is documented and handled correctly
- [ ] **FFI lock coverage:** Often missing lock on new methods -- verify every method that calls native code acquires the FFI lock
- [ ] **Panama parameter arena:** Often using runtime arena for call parameters -- verify `Arena.ofConfined()` is used in try-with-resources for each call's parameter strings
- [ ] **Error propagation:** Often losing native error context -- verify `lastError()` is called inside FFI lock before releasing
- [ ] **AutoCloseable:** Often missing `implements AutoCloseable` on new session types -- verify `ServerSession` implements AutoCloseable with try-with-resources support

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Wrong string ownership (free/no-free) | LOW | Fix the wrapper method used; no data loss, just crashes prevented |
| Server handle not freed after stop | MEDIUM | Add `chroma_server_free` call in `close()`; requires identifying all code paths |
| Missing FFI lock serialization | MEDIUM | Add synchronized blocks to all FFI call sites; requires auditing every method |
| Dual backend drift | HIGH | Diff JNA and Panama implementations line-by-line; extract shared logic to core; write shared contract tests |
| Wrong YAML field names in builder | LOW | Fix string literals; test will catch this immediately |
| Server handle type confused with embedded | HIGH | Requires architectural fix: type-safe handle wrappers that prevent cross-use at compile time |
| Missing maintenance serialization | MEDIUM | Add lock; audit for existing concurrent usage; may need to handle stuck operations |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Global error slot race (Pitfall 1) | Phase 1: Server lifecycle | All FFI calls acquire lock; race detector test with concurrent callers |
| Two-step server teardown (Pitfall 2) | Phase 1: Server lifecycle | `ServerSession.close()` calls stop then free; test verifies both are called |
| Borrowed vs owned pointers (Pitfall 3) | Phase 1: Server lifecycle | Every FFI return has documented ownership; `readOwnedString`/`readBorrowedString` wrappers used consistently |
| Wrong handle type for maintenance (Pitfall 4) | Phase 2: Maintenance APIs | Server maintenance creates temporary embedded; never passes server handle to embedded functions |
| Panama arena lifetime (Pitfall 5) | Phase 1: Server lifecycle | Close runtime after all sessions; test verifies proper error on early runtime close |
| YAML field name mismatch (Pitfall 6) | Phase 1: Server builder | Golden YAML test comparing Java output to Go output; server starts successfully with builder config |
| Dual backend drift (Pitfall 7) | Phase 1: Before new APIs | Shared contract test suite in core; both backends pass same tests |
| Handle mutation on restart (Pitfall 8) | Phase 1: Server lifecycle | Use AtomicLong for handle; maintenance swaps handle atomically |
| Maintenance serialization (Pitfall 9) | Phase 2: Maintenance APIs | Single lock for all maintenance ops; test concurrent maintenance calls |
| JSON field name mismatch (Pitfall 10) | Phase 2: Maintenance APIs | Golden JSON test; maintenance operations succeed end-to-end |
| String free on wrong pointer (Pitfall 11) | Phase 1: Server lifecycle | readOwnedString/readBorrowedString helpers; code review checklist |

## Sources

- Codebase analysis: `shim/src/lib.rs` lines 79-110 (LAST_ERROR global mutex), 166-180 (ServerHandle/EmbeddedHandle structs), 2860-2890 (borrowed pointer docs), 2897-2930 (stop/free separation) -- HIGH confidence
- Codebase analysis: `internal/runtime/chroma.go` lines 19-23 (ffiMu global lock), 166-175 (callFFIHandle pattern), 327-378 (Server.Stop/Close two-step) -- HIGH confidence
- Codebase analysis: `internal/runtime/backup.go`, `rebuild.go`, `compaction.go`, `wal_prune.go` (maintenance stop-embedded-restart pattern) -- HIGH confidence
- Codebase analysis: `java/panama/src/main/java/.../PanamaChromaRuntime.java` (existing Arena pattern, confined arena for parameters) -- HIGH confidence
- Codebase analysis: `java/core/src/main/java/.../EmbeddedSession.java` (existing AtomicBoolean close pattern, LongConsumer design) -- HIGH confidence
- [JEP 454: Foreign Function & Memory API](https://openjdk.org/jeps/454) -- HIGH confidence on Arena lifetime semantics
- [Common JNA Pitfalls and How to Avoid Them](https://javanexus.com/blog/common-jna-pitfalls-avoid) -- MEDIUM confidence
- [JNA thread safety: Native.synchronizedLibrary](https://www.tabnine.com/code/java/methods/com.sun.jna.Native/synchronizedLibrary) -- MEDIUM confidence
- [JNA memory management patterns](https://groups.google.com/g/jna-users/c/0a10BsvE3RE) -- MEDIUM confidence
- [Project Panama's FFM API in Production](https://www.javacodegeeks.com/2026/03/project-panamas-ffm-api-in-production-replacing-jni-without-writing-c-wrappers.html) -- MEDIUM confidence
- [Panama memory segment lifetime](https://dev.java/learn/ffm/) -- HIGH confidence

---
*Pitfalls research for: Java API surface expansion (JNA + Panama dual-backend) on chroma-go-local*
*Researched: 2026-03-21*
