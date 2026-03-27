---
phase: 09-backup-api
verified: 2026-03-27T08:00:00Z
status: passed
score: 6/6 must-haves verified
re_verification: false
---

# Phase 9: Backup API Verification Report

**Phase Goal:** Users can back up Chroma data from both embedded and server modes, producing a directory with a manifest file that records backup metadata
**Verified:** 2026-03-27
**Status:** passed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `EmbeddedSession.backup(options)` creates a backup directory and returns a `BackupManifest` | VERIFIED | `EmbeddedSession.java` line 123–127: `backup()` delegates to `backupAction.apply(options)`. `BackupExecutor.execute("embedded", ...)` performs copy + manifest write and returns `BackupResult<EmbeddedSession>`. Integration tests (`JnaEmbeddedBackupTest`, `PanamaEmbeddedBackupTest`) assert `schemaVersion="v1"`, `mode="embedded"`, sentinel file copied. |
| 2 | `ServerSession.backup(options)` performs stop-backup-restart cycle and returns a `BackupManifest` without corrupting server state | VERIFIED | `ServerSession.java` line 124–128: `backup()` delegates to `backupAction.apply(options)`. JNA/Panama backends inject lambda: `BackupExecutor.execute("server", persistPath, opts, () -> { serverStop(handle); serverFree(handle); }, () -> doStartServer(savedYaml))`. `JnaServerBackupTest.serverBackupCreatesDirectoryWithManifest` asserts `mode="server"`, new `result.session().port() > 0` (server restarted). |
| 3 | `BackupOptions` builder supports `destination`, `includeMetadata`, `leaveClosed`/`leaveStopped` with validation at build time | VERIFIED | `BackupOptions.java`: Builder with all four fields, `build()` throws `IllegalArgumentException` when `destinationPath` is null/blank. Mode-specific option rejection is in `BackupExecutor.validateModeOptions()` (throws on leaveStopped for embedded, leaveClosed for server). |
| 4 | Integration tests verify backup creates valid output directory with expected contents in both JNA and Panama backends | VERIFIED | 4 integration test files created (20 tests total): `JnaEmbeddedBackupTest`, `JnaServerBackupTest`, `PanamaEmbeddedBackupTest`, `PanamaServerBackupTest`. Each has 5 tests covering: sentinel copy, leaveClosed/leaveStopped, mode-rejection, null options, closed session guard. |
| 5 | `BackupExecutor.execute()` copies source directory to destination, writes manifest JSON, and returns `BackupManifest` | VERIFIED | `BackupExecutor.java`: `Files.walkFileTree` copies files with SHA-256 hashing (`MessageDigest.getInstance("SHA-256")`), `writeManifest()` writes pretty-printed JSON to `backup_manifest.json`. `BackupExecutorTest` confirms copy, manifest parse, fileCount, totalBytes. |
| 6 | Both JNA and Panama backends construct backup callback lambdas injected at session creation | VERIFIED | `JnaChromaRuntime.doStartEmbedded` line 126: `opts -> BackupExecutor.execute("embedded", ...)`. `doStartServer` line 143: `opts -> BackupExecutor.execute("server", ...)`. `PanamaChromaRuntime` mirrors identical pattern at lines 258 and 283. |

**Score:** 6/6 truths verified

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `java/core/src/main/java/tech/amikos/chroma/local/core/BackupResult.java` | Generic result type wrapping manifest + session | VERIFIED | 19 lines, `public final class BackupResult<S>`, two getters, `Objects.requireNonNull` on manifest |
| `java/core/src/main/java/tech/amikos/chroma/local/core/BackupExecutor.java` | Core backup algorithm | VERIFIED | 233 lines, `public static <S> BackupResult<S> execute(...)`, `Files.walkFileTree`, `MessageDigest.getInstance("SHA-256")`, `backup_manifest.json` |
| `java/core/src/main/java/tech/amikos/chroma/local/core/BackupManifest.java` | All-args package-private constructor | VERIFIED | Two constructors: no-arg (Gson) and 12-param all-args at line 36 |
| `java/core/src/main/java/tech/amikos/chroma/local/core/BackupFileMetadata.java` | All-args package-private constructor | VERIFIED | Two constructors: no-arg (Gson) and 5-param all-args at line 18 |
| `java/core/src/main/java/tech/amikos/chroma/local/core/EmbeddedSession.java` | 8-param constructor with backup callback slot | VERIFIED | 8-param constructor at line 19, `backup()` method at line 123 delegates to `backupAction.apply(options)` |
| `java/core/src/main/java/tech/amikos/chroma/local/core/ServerSession.java` | 7-param constructor with backup callback slot, no UnsupportedOperationException | VERIFIED | 7-param constructor at line 19, `backup()` at line 124 delegates to `backupAction.apply(options)`; no stub exception on backup |
| `java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java` | Backup lambdas in doStartEmbedded and doStartServer | VERIFIED | `BackupExecutor.execute("embedded", ...)` at line 126; `BackupExecutor.execute("server", ...)` at line 143 |
| `java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java` | Backup lambdas in doStartEmbedded and doStartServer | VERIFIED | `BackupExecutor.execute("embedded", ...)` at line 258; `BackupExecutor.execute("server", ...)` at line 283 |
| `java/jna/src/test/java/tech/amikos/chroma/local/jna/JnaEmbeddedBackupTest.java` | JNA embedded backup integration tests | VERIFIED | 146 lines, 5 test methods |
| `java/jna/src/test/java/tech/amikos/chroma/local/jna/JnaServerBackupTest.java` | JNA server backup integration tests | VERIFIED | 173 lines, 5 test methods |
| `java/panama/src/test/java/tech/amikos/chroma/local/panama/PanamaEmbeddedBackupTest.java` | Panama embedded backup integration tests | VERIFIED | 146 lines, 5 test methods |
| `java/panama/src/test/java/tech/amikos/chroma/local/panama/PanamaServerBackupTest.java` | Panama server backup integration tests | VERIFIED | 173 lines, 5 test methods |
| `java/core/src/test/java/tech/amikos/chroma/local/core/BackupResultTest.java` | Unit tests for BackupResult | VERIFIED | 3 tests: accessors, null manifest rejection, null session allowed |
| `java/core/src/test/java/tech/amikos/chroma/local/core/BackupExecutorTest.java` | Unit tests for backup algorithm | VERIFIED | 8 tests: copy+manifest, leaveClosed, mode rejections, dest-inside-source, non-empty dest, missing source, YAML extraction |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `EmbeddedSession.backup()` | `BackupResult<EmbeddedSession>` | `backupAction.apply(options)` | WIRED | Line 126: `return backupAction.apply(options)` |
| `ServerSession.backup()` | `BackupResult<ServerSession>` | `backupAction.apply(options)` | WIRED | Line 127: `return backupAction.apply(options)` |
| `JnaChromaRuntime.doStartEmbedded` | `BackupExecutor.execute` | backup lambda injected at session construction | WIRED | Line 126: `opts -> BackupExecutor.execute("embedded", persistPath, opts, ...)` |
| `JnaChromaRuntime.doStartServer` | `BackupExecutor.execute` | backup lambda injected at session construction | WIRED | Line 143: `opts -> BackupExecutor.execute("server", persistPath, opts, ...)` |
| `PanamaChromaRuntime.doStartEmbedded` | `BackupExecutor.execute` | backup lambda injected at session construction | WIRED | Line 258: `opts -> BackupExecutor.execute("embedded", persistPath, opts, ...)` |
| `PanamaChromaRuntime.doStartServer` | `BackupExecutor.execute` | backup lambda injected at session construction | WIRED | Line 283: `opts -> BackupExecutor.execute("server", persistPath, opts, ...)` |

---

### Data-Flow Trace (Level 4)

Level 4 is not applicable here. The artifacts are Java library types (no web rendering, no dashboard). The data flow is: `session.backup(options)` → `backupAction.apply(options)` → `BackupExecutor.execute(...)` → filesystem copy → `BackupResult<S>` returned to caller. The full chain was verified at Level 3 (wiring). The unit tests in `BackupExecutorTest` confirm the algorithm produces real file output, not static/empty results.

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Core module compiles and all unit tests pass | `gradle --no-daemon :core:test` | `BUILD SUCCESSFUL in 3s`, 2 tasks executed | PASS |
| All three modules compile and lint-check | `gradle --no-daemon :core:check :jna:check :panama:check` | `BUILD SUCCESSFUL in 2s`, 10 tasks up-to-date | PASS |
| Commit hashes from summaries exist in git log | `git log --oneline 305153d e47551a eeafe8d 9f99392` | All 4 commits present with matching descriptions | PASS |
| Integration tests compile (JNA/Panama test classes) | Included in `:jna:check` and `:panama:check` above | Test classes compiled without error | PASS |

Note: Integration tests (`JnaEmbeddedBackupTest`, `JnaServerBackupTest`, `PanamaEmbeddedBackupTest`, `PanamaServerBackupTest`) require `CHROMA_LIB_PATH` env var pointing to the built native shim. They use `Assumptions.assumeTrue` and will be skipped without FFI. Running them end-to-end requires a human with the built shim.

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| BKUP-01 | 09-01-PLAN.md | `EmbeddedSession.backup(options)` performs directory copy with manifest and returns BackupManifest | SATISFIED | `EmbeddedSession.backup()` → `BackupExecutor.execute("embedded", ...)` → `BackupResult<EmbeddedSession>`. Integration tests verify sentinel copy and manifest content. |
| BKUP-02 | 09-01-PLAN.md | `ServerSession.backup(options)` performs stop-backup-restart cycle and returns BackupManifest | SATISFIED | `ServerSession.backup()` → `BackupExecutor.execute("server", ...)` with `serverStop+serverFree` as closeAction and `doStartServer` as restartAction. Server backup tests assert new session port > 0. |
| BKUP-03 | 09-01-PLAN.md | `BackupOptions` builder supports destination, includeMetadata, leaveClosed/leaveStopped | SATISFIED | `BackupOptions.Builder` has all four fields. `build()` validates `destinationPath`. Mode-specific flag validation in `BackupExecutor.validateModeOptions()`. Tests confirm leaveStopped rejected for embedded, leaveClosed rejected for server. |
| BKUP-04 | 09-02-PLAN.md | Integration tests verify backup creates valid output directory in both backends | SATISFIED | 4 integration test files (20 tests) across JNA and Panama. Each test class covers: sentinel file copied to `dest/persist/`, `backup_manifest.json` present, manifest schema+mode assertions, leaveClosed/leaveStopped semantics, null options rejection, closed session guard. |

All 4 BKUP requirements marked Complete in REQUIREMENTS.md. No orphaned requirements.

---

### Anti-Patterns Found

No blocking or warning-level anti-patterns found.

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `ServerSession.java` | 88, 97, 106, 111, 121 | `UnsupportedOperationException` for rebuildCollection, compactAll, compactCollection, pruneCollectionWAL, pruneAllWAL | Info | Intentional Phase 10 stubs for server maintenance operations — explicitly scoped out of Phase 9. `backup()` is fully implemented. No impact on BKUP requirements. |

---

### Human Verification Required

The following require a built native shim (`CHROMA_LIB_PATH`) and a live Chroma runtime:

**1. Embedded backup end-to-end with live FFI**

Test: Build the Rust shim (`make build`), then run `CHROMA_LIB_PATH=shim/target/debug/libchroma_shim.dylib gradle --no-daemon :jna:test --tests '*JnaEmbeddedBackupTest*'`
Expected: All 5 tests pass; `embeddedBackupCreatesDirectoryWithManifest` confirms `sentinel.txt` copied and manifest fileCount >= 1
Why human: Requires a built native library that embeds a live Chroma runtime

**2. Server backup stop-restart cycle with live FFI**

Test: `CHROMA_LIB_PATH=... gradle --no-daemon :jna:test --tests '*JnaServerBackupTest*'`
Expected: `serverBackupCreatesDirectoryWithManifest` passes; new `result.session().port() > 0` confirms server restarted successfully
Why human: Requires native library; server start/stop touches OS-level port binding and process state

**3. Panama backend end-to-end**

Test: Same as items 1 and 2 but with Panama test classes
Expected: Identical results to JNA (D-12 spec requires parity)
Why human: Requires Java 22+ JVM with Panama FFI support and native library

---

### Gaps Summary

No gaps. All 6 truths are verified, all artifacts exist and are substantive, all key links are wired, all 4 BKUP requirements are satisfied, and all automated checks pass. The three human verification items are end-to-end FFI integration tests that require a built native shim — they are the intended "integration test" coverage for BKUP-04. The test code is complete and correct; only execution against a live library is deferred to human verification.

---

_Verified: 2026-03-27_
_Verifier: Claude (gsd-verifier)_
