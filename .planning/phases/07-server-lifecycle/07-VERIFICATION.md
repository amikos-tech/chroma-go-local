---
phase: 07-server-lifecycle
verified: 2026-03-26T09:00:00Z
status: passed
score: 9/9 must-haves verified
re_verification: false
human_verification:
  - test: "Start real server via JNA backend with CHROMA_LIB_PATH set and verify port/address accessors match config"
    expected: "JnaServerLifecycleTest.serverStartAccessorsStopClose passes with assertions on port(), address(), url()"
    why_human: "Requires built Rust shim (libchroma_shim) and real FFI call -- cannot run without CHROMA_LIB_PATH in this environment"
  - test: "Start real server via Panama backend and verify identical behavior"
    expected: "PanamaServerLifecycleTest.serverStartAccessorsStopClose passes"
    why_human: "Same as above -- requires Rust shim at runtime"
  - test: "Verify server actually listens on the configured port (not just returns the port value)"
    expected: "HTTP connection to http://127.0.0.1:{port} succeeds after startServer returns"
    why_human: "Requires Rust shim, real OS port binding, and live server"
---

# Phase 7: Server Lifecycle Verification Report

**Phase Goal:** Users can start a Chroma server from Java, retrieve its connection details, and cleanly shut it down using try-with-resources in both JNA and Panama backends
**Verified:** 2026-03-26T09:00:00Z
**Status:** passed
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|---------|
| 1 | JnaChromaRuntime extends AbstractChromaRuntime instead of implementing ChromaRuntime directly | VERIFIED | Line 13: `public final class JnaChromaRuntime extends AbstractChromaRuntime` -- no `implements ChromaRuntime` present |
| 2 | PanamaChromaRuntime extends AbstractChromaRuntime instead of implementing ChromaRuntime directly | VERIFIED | Line 18: `public final class PanamaChromaRuntime extends AbstractChromaRuntime` -- no `implements ChromaRuntime` present |
| 3 | All FFI calls in both backends route through callFfiHandle, callFfiBorrowedString, or callFfiVoid | VERIFIED | JNA: 7 template method calls (callFfiHandle x3, callFfiBorrowedString x3, callFfiVoid x1). Panama: 7 calls (same pattern). serverFree and embeddedFree intentionally bypass FFI lock per plan decision |
| 4 | Both backends implement readBorrowedString, readOwnedString, and readLastError abstract methods | VERIFIED | All 3 abstract methods implemented with @Override in both JnaChromaRuntime (lines 59-85) and PanamaChromaRuntime (lines 136-169) |
| 5 | startServer() in both backends returns a ServerSession wired with method-reference callbacks | VERIFIED | Both backends construct `new ServerSession(handle, this::serverStop, this::serverFree, this::serverPort, this::serverAddress, this::serverPersistPath)` |
| 6 | Integration tests verify startServer returns a working ServerSession in JNA backend | VERIFIED | JnaServerLifecycleTest.java exists at 125 lines with 6 test methods including serverStartAccessorsStopClose |
| 7 | Integration tests verify startServer returns a working ServerSession in Panama backend | VERIFIED | PanamaServerLifecycleTest.java exists at 125 lines with identical 6-test structure using PanamaChromaRuntime |
| 8 | Integration tests verify port(), address(), url() return correct values matching config | VERIFIED | Both test classes assert: assertEquals(port, session.port()), assertEquals("127.0.0.1", session.address()), assertEquals("http://127.0.0.1:" + port, session.url()) |
| 9 | Integration tests verify close() is idempotent and accessors throw IllegalStateException after close | VERIFIED | doubleCloseIsIdempotent and accessorsThrowAfterClose tests present in both test classes with correct assertions |

**Score:** 9/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java` | JNA backend extending AbstractChromaRuntime | VERIFIED | Exists, 169 lines, substantive, extends AbstractChromaRuntime |
| `java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java` | Panama backend extending AbstractChromaRuntime | VERIFIED | Exists, 336 lines, substantive, extends AbstractChromaRuntime |
| `java/jna/src/test/java/tech/amikos/chroma/local/jna/JnaServerLifecycleTest.java` | JNA server lifecycle integration tests | VERIFIED | Exists, 125 lines (min_lines=80 met), 6 test methods |
| `java/panama/src/test/java/tech/amikos/chroma/local/panama/PanamaServerLifecycleTest.java` | Panama server lifecycle integration tests | VERIFIED | Exists, 125 lines (min_lines=80 met), 6 test methods |
| `java/core/src/main/java/tech/amikos/chroma/local/core/AbstractChromaRuntime.java` | Base class with FFI lock template methods | VERIFIED | Exists, provides callFfiHandle, callFfiBorrowedString, callFfiVoid, callFfiJson |
| `java/core/src/main/java/tech/amikos/chroma/local/core/ServerSession.java` | ServerSession with idempotent close and guarded accessors | VERIFIED | Exists, close() uses CAS + stop-in-try/free-in-finally, all accessors guard via ensureOpen() |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| JnaChromaRuntime | AbstractChromaRuntime | extends | WIRED | `public final class JnaChromaRuntime extends AbstractChromaRuntime` |
| PanamaChromaRuntime | AbstractChromaRuntime | extends | WIRED | `public final class PanamaChromaRuntime extends AbstractChromaRuntime` |
| JnaChromaRuntime.startServer | ServerSession | callFfiHandle + constructor wiring | WIRED | `new ServerSession(handle, this::serverStop, ...)` at line 112 |
| PanamaChromaRuntime.startServer | ServerSession | callFfiHandle + constructor wiring | WIRED | `new ServerSession(handle, this::serverStop, ...)` at line 220 |
| JnaServerLifecycleTest | JnaChromaRuntime.startServer | runtime.startServer(yaml) | WIRED | Pattern `runtime.startServer` appears 6 times across 6 test methods |
| PanamaServerLifecycleTest | PanamaChromaRuntime.startServer | runtime.startServer(yaml) | WIRED | Pattern `runtime.startServer` appears 6 times across 6 test methods |
| Both test classes | ServerConfigBuilder | new ServerConfigBuilder().port(port).build() | WIRED | ServerConfigBuilder imported and used in both test classes |

### Data-Flow Trace (Level 4)

Not applicable -- these are FFI integration classes, not data-rendering components. The data flows through native FFI calls that require the Rust shim at runtime. Static analysis confirms the callback slots are wired; runtime behavior requires human verification.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All modules compile cleanly | `gradle :core:compileJava :jna:compileJava :panama:compileJava :jna:compileTestJava :panama:compileTestJava` | BUILD SUCCESSFUL in 45s | PASS |
| JnaChromaRuntime does not use implements ChromaRuntime | `grep "implements ChromaRuntime" JnaChromaRuntime.java` | No output | PASS |
| PanamaChromaRuntime does not use implements ChromaRuntime | `grep "implements ChromaRuntime" PanamaChromaRuntime.java` | No output | PASS |
| Old lastError(String fallback) method removed from both backends | `grep "lastError(String fallback)"` on both files | No output (exit 1) | PASS |
| Commit 2b7defc exists (JNA retrofit) | `git show --stat 2b7defc` | Confirmed: feat(07-01) JnaChromaRuntime retrofit | PASS |
| Commit 15035ec exists (Panama retrofit) | `git show --stat 15035ec` | Confirmed: feat(07-01) PanamaChromaRuntime retrofit | PASS |
| Commit b71fff3 exists (integration tests) | `git show --stat b71fff3` | Confirmed: test(07-02) JNA and Panama lifecycle tests | PASS |
| Integration test runtime (with shim) | Requires CHROMA_LIB_PATH and built shim | Not tested in this environment | SKIP |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|------------|-------------|-------------|--------|---------|
| SRVR-01 | 07-01-PLAN.md | ChromaRuntime.startServer(configYaml) returns ServerSession in both JNA and Panama | SATISFIED | Both backends implement startServer() calling callFfiHandle and returning new ServerSession with method-reference callbacks. ServerSession constructor takes all 6 required callbacks |
| SRVR-02 | 07-01-PLAN.md | ServerSession implements AutoCloseable with idempotent close and two-step teardown (stop + free) | SATISFIED | ServerSession.close() uses AtomicBoolean CAS for idempotency; calls stopAction in try block then freeAction in separate try block (two-step). Tests assert doubleCloseIsIdempotent and accessorsThrowAfterClose |
| SRVR-03 | 07-01-PLAN.md | ServerSession.port(), address(), url() return server connection details | SATISFIED | All three methods exist in ServerSession: port() calls portAccessor, address() calls addressAccessor, url() builds "http://"+address()+":"+port(). All guarded by ensureOpen() |
| SRVR-04 | 07-02-PLAN.md | Integration tests verify server start, accessor values, stop, and close in both backends | SATISFIED | 6 test methods in each of JnaServerLifecycleTest and PanamaServerLifecycleTest covering: happy path with accessor assertions, double close, post-close guard, null/empty/malformed config rejection |

**Note:** REQUIREMENTS.md traceability table still shows SRVR-01..04 as "Pending" (not updated after implementation). This is a documentation gap only -- the code fully satisfies all four requirements. The plan SUMMARYs record `requirements-completed: [SRVR-01, SRVR-02, SRVR-03]` and `requirements-completed: [SRVR-04]` respectively.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `java/core/src/main/java/tech/amikos/chroma/local/core/ServerSession.java` | 83-112 | `throw new UnsupportedOperationException(...)` for rebuildCollection, compactCollection, etc. | Info | These are intentional placeholders for Phase 8/9/10 scope -- explicitly documented as "will be wired in Phase 8/9". Not Phase 7 scope. No impact on phase goal. |
| `.planning/REQUIREMENTS.md` | 77-80 | SRVR-01..04 status still shows "Pending" in traceability table | Info | Documentation not updated after implementation. Code fully satisfies requirements. |

No blocker or warning anti-patterns found in Phase 7 implementation files.

### Human Verification Required

#### 1. JNA Server Start -- Real FFI Invocation

**Test:** Set `CHROMA_LIB_PATH` to built `libchroma_shim.dylib` path, run `gradle :jna:test --tests JnaServerLifecycleTest.serverStartAccessorsStopClose`
**Expected:** Test passes: server starts, port() returns configured port, address() returns "127.0.0.1", url() returns "http://127.0.0.1:{port}", persistPath() is non-blank
**Why human:** Requires built Rust shim artifact -- cannot build without `make build` or `make build-release` in this environment

#### 2. Panama Server Start -- Real FFI Invocation

**Test:** Same as above but `gradle :panama:test --tests PanamaServerLifecycleTest.serverStartAccessorsStopClose`
**Expected:** Identical behavior using Panama MethodHandle downcall path
**Why human:** Same runtime dependency on Rust shim

#### 3. Server Actually Binds the Port

**Test:** After serverStartAccessorsStopClose test runs, verify HTTP GET to `http://127.0.0.1:{port}/api/v2` returns a non-error response
**Expected:** Chroma server is actually listening, not just reported as started
**Why human:** Requires live server process -- cannot verify from static analysis

### Gaps Summary

No gaps found. All must-haves from both plans are satisfied:

- Both JNA and Panama backends extend AbstractChromaRuntime (not `implements ChromaRuntime`)
- All 3 abstract methods (readBorrowedString, readOwnedString, readLastError) implemented in each backend
- All FFI calls route through template methods (callFfiHandle, callFfiBorrowedString, callFfiVoid)
- String ownership correct: borrowed for version/address/persistPath, owned+free for errors
- ServerSession wired with all 5 method-reference callbacks (serverStop, serverFree, serverPort, serverAddress, serverPersistPath)
- ServerSession.close() is idempotent via CAS and performs two-step teardown (stop then free)
- All 6 test methods present in both JnaServerLifecycleTest and PanamaServerLifecycleTest
- Tests use ServerConfigBuilder for YAML generation and findFreePort() for ephemeral ports
- Full compilation succeeds: `gradle :core:compileJava :jna:compileJava :panama:compileJava :jna:compileTestJava :panama:compileTestJava` exits 0
- All 3 commits (2b7defc, 15035ec, b71fff3) verified in git history
- Existing smoke test files (JnaChromaRuntimeTest.java, PanamaChromaRuntimeTest.java) are untouched

The phase goal is achieved at the code level. Runtime behavior with the Rust shim requires human verification with `CHROMA_LIB_PATH` set.

---

_Verified: 2026-03-26T09:00:00Z_
_Verifier: Claude (gsd-verifier)_
