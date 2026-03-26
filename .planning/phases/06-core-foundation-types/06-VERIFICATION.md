---
phase: 06-core-foundation-types
verified: 2026-03-22T20:30:00Z
status: passed
score: 14/14 must-haves verified
re_verification: false
---

# Phase 06: Core Foundation Types Verification Report

**Phase Goal:** The core module contains all shared interfaces, builders, result types, and FFI safety infrastructure so that backend modules (JNA, Panama) can implement against stable contracts without any FFI dependency in core
**Verified:** 2026-03-22T20:30:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | All 7 result POJOs compile in core module without JNA or Panama imports | VERIFIED | grep confirms zero JNA/Panama imports in core/src/main; 7 POJO files exist and compile |
| 2 | Result POJOs deserialize from JSON matching Rust shim output format | VERIFIED | 4 test classes pass (RebuildCollectionResultTest, CompactionResultTest, WALPruneResultTest, BackupManifestTest) using inline JSON strings matching Go struct field names |
| 3 | Optional numeric fields use boxed Long and are null when absent from JSON | VERIFIED | CompactionCollectionResult.java has `private final Long pendingOpsBefore`; WALPruneCollectionResult.java has 5 boxed Long fields; tests verify null when absent |
| 4 | Gson shared instance uses LOWER_CASE_WITH_UNDERSCORES naming policy | VERIFIED | JsonUtil.java line 10: `FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES` |
| 5 | ServerConfigBuilder default build() produces YAML matching Go's DefaultServerConfig().toYAML() | VERIFIED | ServerConfigBuilder.java uses LinkedHashMap with all 6 required fields (port=8000, listenAddress="127.0.0.1", maxPayloadSizeBytes=41943040, persistPath="./chroma", sqliteFilename="chroma.sqlite3", allowReset=false); ServerConfigBuilderTest passes |
| 6 | EmbeddedConfigBuilder default build() produces YAML matching Go's DefaultEmbeddedConfig().toYAML() | VERIFIED | EmbeddedConfigBuilder.java produces persist_path, sqlite_filename, allow_reset with correct defaults; test passes |
| 7 | Config builders validate inputs at build() time and throw IllegalArgumentException | VERIFIED | Both builders call validate() before YAML generation; ServerConfigBuilder checks port 1-65535, non-blank persistPath/listenAddress; EmbeddedConfigBuilder checks non-blank persistPath |
| 8 | rawYaml() escape hatch overrides all other fields on both builders | VERIFIED | Both builders check `if (rawYaml != null) return rawYaml;` as first line of build() |
| 9 | Option types produce JSON via toJson() matching Go's FFI request format | VERIFIED | All 5 option/request types call JsonUtil.toJson(this); LOWER_CASE_WITH_UNDERSCORES policy produces snake_case keys matching Go struct json tags |
| 10 | WALPruneOptions validates watermark pairs and policy requirements at build() | VERIFIED | WALPruneOptions.Builder.build() validates: watermark high/low must be both-or-neither, lowBytes <= highBytes, maxAgeSeconds > 0, non-dry-run requires at least one policy |
| 11 | AbstractChromaRuntime provides callFfiHandle() and callFfiJson() template methods that acquire a global ReentrantLock | VERIFIED | AbstractChromaRuntime.java: `private static final ReentrantLock FFI_LOCK`; callFfiHandle, callFfiJson, callFfiVoid, callFfiBorrowedString all call FFI_LOCK.lock()/unlock() |
| 12 | Backends extend AbstractChromaRuntime and implement readBorrowedString(), readOwnedString(), readLastError() | VERIFIED | Three abstract methods declared in AbstractChromaRuntime; AbstractChromaRuntimeTest uses TestChromaRuntime concrete subclass to verify the contract |
| 13 | ServerSession wraps a long handle with callback slots for lifecycle, accessors, and maintenance | VERIFIED | ServerSession.java: handle, stopAction/freeAction (LongConsumer), portAccessor (LongToIntFunction), addressAccessor/persistPathAccessor (LongFunction), maintenance method stubs with ensureOpen() guards |
| 14 | ChromaRuntime interface includes startServer(String configYaml) returning ServerSession | VERIFIED | ChromaRuntime.java line 8: `ServerSession startServer(String configYaml)` |

**Score:** 14/14 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `java/core/build.gradle.kts` | Gson and SnakeYAML dependencies | VERIFIED | Lines 11-12: `gson:2.13.2` and `snakeyaml:2.6` as implementation dependencies |
| `java/core/src/main/.../JsonUtil.java` | Shared Gson instance with LOWER_CASE_WITH_UNDERSCORES | VERIFIED | Package-private final class; `static final Gson GSON`; both fromJson/toJson static methods |
| `java/core/src/main/.../RebuildCollectionResult.java` | Rebuild result POJO | VERIFIED | 12 fields, package-private constructor, accessor-style methods |
| `java/core/src/main/.../CompactionCollectionResult.java` | Compaction collection POJO with boxed Long | VERIFIED | `private final Long pendingOpsBefore/pendingOpsAfter` |
| `java/core/src/main/.../CompactionResult.java` | Compaction result POJO | VERIFIED | `List<CompactionCollectionResult> collections` wired |
| `java/core/src/main/.../WALPruneCollectionResult.java` | WAL prune collection POJO with 5 boxed Long fields | VERIFIED | safeSeqCutoff, candidateSeqMin, candidateSeqMax, prunedSeqMin, prunedSeqMax all boxed Long |
| `java/core/src/main/.../WALPruneResult.java` | WAL prune result POJO | VERIFIED | `List<WALPruneCollectionResult> collections` wired |
| `java/core/src/main/.../BackupFileMetadata.java` | Backup file metadata POJO | VERIFIED | path, sizeBytes, mode, sha256, modifiedAt fields |
| `java/core/src/main/.../BackupManifest.java` | Backup manifest POJO | VERIFIED | `List<BackupFileMetadata> files` wired |
| `java/core/src/main/.../RebuildOptions.java` | Rebuild options with nested Builder | VERIFIED | `public static class Builder`; `toJson()` via JsonUtil; `defaults(String name)` factory |
| `java/core/src/main/.../WALPruneOptions.java` | WAL prune options with nested Builder and validation | VERIFIED | boxed Long policy fields; watermark pair validation; policy requirement validation |
| `java/core/src/main/.../BackupOptions.java` | Backup options with nested Builder | VERIFIED | destinationPath, includeMetadata, leaveStopped, leaveClosed; non-blank validation |
| `java/core/src/main/.../CompactCollectionRequest.java` | Compact collection request with Builder | VERIFIED | name required, databaseName >= 3 char validation |
| `java/core/src/main/.../CompactAllRequest.java` | Compact all request with Builder | VERIFIED | optional tenantId/databaseName with validation |
| `java/core/src/main/.../ServerConfigBuilder.java` | Server config YAML builder | VERIFIED | All 6 default fields match Go's DefaultServerConfig(); CORS and OTel conditional sections |
| `java/core/src/main/.../EmbeddedConfigBuilder.java` | Embedded config YAML builder | VERIFIED | 3 default fields match Go's DefaultEmbeddedConfig(); no `extends` (independent per D-08) |
| `java/core/src/main/.../AbstractChromaRuntime.java` | FFI safety base class with lock and template methods | VERIFIED | `private static final ReentrantLock FFI_LOCK`; 4 template methods; 3 abstract methods |
| `java/core/src/main/.../ChromaRuntime.java` | Runtime interface with startServer | VERIFIED | `ServerSession startServer(String configYaml)` added |
| `java/core/src/main/.../ServerSession.java` | Server session with callback slots | VERIFIED | All lifecycle callbacks, accessor callbacks, maintenance stubs with ensureOpen guards |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `JsonUtil.java` | `com.google.code.gson:gson` | `implementation` in build.gradle.kts | VERIFIED | build.gradle.kts line 11 |
| `CompactionResult.java` | `CompactionCollectionResult.java` | `List<CompactionCollectionResult> collections` | VERIFIED | CompactionResult.java line 10 |
| `ServerConfigBuilder.java` | `org.yaml:snakeyaml` | `new Yaml(options)` | VERIFIED | ServerConfigBuilder.java line 118 |
| `RebuildOptions.java` | `JsonUtil.java` | `JsonUtil.toJson(this)` | VERIFIED | RebuildOptions.java line 26 |
| `AbstractChromaRuntime.java` | `ChromaRuntime.java` | `implements ChromaRuntime` | VERIFIED | AbstractChromaRuntime.java line 6 |
| `AbstractChromaRuntime.java` | `ChromaException.java` | `throw new ChromaException(...)` | VERIFIED | AbstractChromaRuntime.java lines 22, 36, 60 |
| `ServerSession.java` | `EmbeddedSession.java` | `AtomicBoolean` pattern | VERIFIED | ServerSession.java uses same AtomicBoolean.compareAndSet idiom |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| FOUND-01 | 06-01, 06-02, 06-03 | Core module contains all shared interfaces, builders, and result types with no FFI dependencies | SATISFIED | 19 classes in core/src/main; zero JNA/Panama imports confirmed; `gradle :core:clean :core:test` exits 0 |
| FOUND-02 | 06-02 | ServerConfigBuilder produces valid YAML with fluent API | SATISFIED | ServerConfigBuilder.java with all 6 required fields and defaults matching Go; tests pass |
| FOUND-03 | 06-02 | EmbeddedConfigBuilder produces valid YAML with fluent API | SATISFIED | EmbeddedConfigBuilder.java with 3 fields matching Go's DefaultEmbeddedConfig(); tests pass |
| FOUND-04 | 06-01 | Result POJOs for all maintenance operations | SATISFIED | BackupManifest, RebuildCollectionResult, CompactionResult, WALPruneResult all created with nested collection types |
| FOUND-05 | 06-03 | FFI serialization lock pattern established to protect global error slot | SATISFIED | AbstractChromaRuntime `private static final ReentrantLock FFI_LOCK`; lock serialization verified by AbstractChromaRuntimeTest |
| FOUND-06 | 06-03 | String ownership helpers distinguish owned vs borrowed native pointers | SATISFIED | `readBorrowedString(long)` and `readOwnedString(long)` abstract methods in AbstractChromaRuntime define the contract |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `ServerSession.java` | 62, 71, 76, 81, 86, 91 | `throw new UnsupportedOperationException("... will be wired in Phase 8/9")` | Info | Expected and documented: maintenance method stubs are intentionally deferred to Phases 7-10. The public API surface is complete; backends wire callbacks in later phases. Not a blocker. |

### Human Verification Required

None. All phase goals can be verified programmatically through code inspection and test execution.

### Gaps Summary

No gaps. All 14 observable truths are verified, all required artifacts exist and are substantive, all key links are wired, all 6 requirement IDs (FOUND-01 through FOUND-06) are satisfied, and the test suite runs clean with 99 tests, 0 failures, 0 errors.

The maintenance method stubs in ServerSession (throwing `UnsupportedOperationException`) are intentional deferred implementations explicitly documented as "wired in Phase 8/9" — this is the designed state for Phase 6. The public API surface is complete and the stubs are not stubs in the deceptive sense; they exist to define method signatures for backends to depend on.

---

_Verified: 2026-03-22T20:30:00Z_
_Verifier: Claude (gsd-verifier)_
