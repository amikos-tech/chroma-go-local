# Stack Research: Java API Surface (v0.5.0)

**Domain:** Java FFI bindings for Chroma local runtime (JNA + Panama dual backend)
**Researched:** 2026-03-21
**Confidence:** HIGH

## Scope

This research covers ONLY the technology additions and patterns needed to implement the v0.5.0 Java API surface: server lifecycle, builder configuration, backup, rebuild, compaction, and WAL prune. It does NOT re-evaluate Go, Rust, or the existing Java scaffold -- those are validated.

---

## Recommended Stack

### Core Technologies (Already in Place -- No Version Changes)

| Technology | Version | Purpose | Status |
|------------|---------|---------|--------|
| JNA | 5.14.0 | FFI for Java 17+ backend | Keep current. 5.18.1 exists but 5.14.0 is stable and sufficient -- no features we need from newer versions. Upgrade is optional, not required. |
| Panama FFM API | JDK 22 built-in | FFI for Java 22+ backend | Finalized in JEP 454 (JDK 22). No external dependency. Already configured with `--enable-native-access=ALL-UNNAMED` in tests. |
| JUnit 5 | 5.11.4 (BOM) | Test framework | Keep current. 5.14.2 exists but 5.11.4 has everything we need (`@TempDir`, `Assumptions`, `@Nested`). Upgrade is optional. |
| Gradle | 9+ (Kotlin DSL) | Build system | No changes needed. Existing multi-module setup (`core`, `jna`, `panama`) is correct. |
| Java 17 | Toolchain (core, jna) | Minimum baseline | No change. |
| Java 22 | Toolchain (panama) | Panama FFM availability | No change. |

### New Libraries Required: Exactly One (Gson for JSON)

| Library | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| Gson | 2.13.2 | JSON serialization/deserialization for FFI request/response strings | See detailed rationale below. |

**Why JSON at all?** The maintenance APIs (rebuild, compaction, WAL prune) pass structured requests and return structured responses through the FFI boundary as JSON strings. The Go side uses `encoding/json`. The Rust shim's `chroma_embedded_rebuild_collection`, `chroma_embedded_compact_collection`, `chroma_embedded_compact_all`, `chroma_embedded_prune_wal_collection`, and `chroma_embedded_prune_wal_all` all accept a JSON C string as input and return a JSON C string as output. Java must serialize request objects to JSON strings and deserialize response strings to Java objects.

**Why Gson over Jackson?**

| Criterion | Gson 2.13.2 | Jackson 2.21.1 |
|-----------|-------------|----------------|
| Jar size | ~300 KB (single jar) | ~1.8 MB (databind + core + annotations) |
| Transitive deps | 0 | 2 (jackson-core, jackson-annotations) |
| Setup complexity | `new Gson()` | `new ObjectMapper()` with module registration |
| Performance at scale | Slower for very large payloads | Faster streaming parser |
| Our payload sizes | Tiny JSON (<1 KB per request/response) | Overkill |
| Annotation-free POJOs | Yes, works out of the box | Requires annotations for some features |

**Decision: Use Gson.** The FFI payloads are tiny JSON blobs (rebuild request ~100 bytes, compaction result ~500 bytes). Gson adds one 300 KB jar with zero transitive dependencies. Jackson would add ~1.8 MB across 3 jars for the same functionality. At these payload sizes, performance is irrelevant. Simplicity wins.

**Confidence: HIGH** -- Both libraries handle our use case trivially. Gson wins on footprint and simplicity for this specific scenario.

### Libraries NOT Needed

| Library | Why Considered | Why Rejected |
|---------|---------------|-------------|
| SnakeYAML | YAML config builder serialization | The Java builder generates a YAML string directly via `StringBuilder`, exactly as the Go `config.toYAML()` does. No YAML library needed -- the configs are simple enough for hand-written string construction. |
| Lombok | Builder pattern generation | Adds annotation processing, IDE plugin requirements, and bytecode manipulation for what amounts to 5-6 simple builder classes. Hand-rolled builders are trivial and match the project's "radically simple" philosophy. |
| Immutables / AutoValue | Immutable value types for response objects | Response classes are simple data carriers with final fields. Records (Java 17+) or plain final-field classes are sufficient. No annotation processing overhead needed. |
| AssertJ | Fluent test assertions | Adds a dependency for marginally nicer assertion syntax. JUnit 5's built-in assertions are fine for the test patterns here (mostly null checks, string comparisons, and lifecycle verification). Keep the test dependency tree minimal. |
| Mockito | Mock testing | All Java tests are integration tests that call through to the real native library. Mocking the FFI layer would test nothing useful. `Assumptions.assumeTrue(libPath != null)` pattern for skipping when native lib unavailable is the correct approach. |

---

## No YAML Library Needed -- Hand-Rolled YAML Generation

The Go side generates YAML config strings with `fmt.Fprintf(&b, "port: %d\n", c.Port)` in `ServerConfig.toYAML()` and `EmbeddedConfig.toYAML()`. The Java builders should do the same with `StringBuilder`:

```java
public String toYaml() {
    StringBuilder b = new StringBuilder();
    b.append("port: ").append(port).append('\n');
    b.append("listen_address: \"").append(listenAddress).append("\"\n");
    b.append("persist_path: \"").append(persistPath).append("\"\n");
    // ...
    return b.toString();
}
```

This keeps Java output byte-identical to Go output, avoids a YAML library dependency, and is trivial to test. The YAML config format is controlled by the Chroma runtime and is a flat key-value structure -- no nested objects, no anchors, no multi-document streams. A YAML library adds complexity for zero benefit here.

**Confidence: HIGH** -- Direct examination of the existing Go `config.go` confirms YAML generation is simple string formatting.

---

## Builder Pattern: Hand-Rolled (No Library)

### Why Hand-Rolled

The project needs approximately 5 builders:
1. `ServerConfigBuilder` -- mirrors Go's `ServerConfig` with `WithPort`, `WithListenAddress`, etc.
2. `EmbeddedConfigBuilder` -- mirrors Go's `EmbeddedConfig`
3. `BackupOptionsBuilder` -- mirrors Go's `BackupOption` variadic pattern
4. `RebuildOptionsBuilder` -- mirrors Go's `RebuildCollectionOption` variadic pattern
5. `WALPruneOptionsBuilder` -- mirrors Go's `WALPruneOption` variadic pattern

Each builder has 3-8 setter methods and a `build()` method. This is 100-200 lines of code total. Adding Lombok or Immutables for this is negative ROI: the annotation processor dependency, IDE plugin setup, and build configuration cost more than writing the builders by hand.

### Pattern to Follow

Use static inner `Builder` class on the config/options type. This is idiomatic Java and matches what consumers expect:

```java
public final class ServerConfig {
    private final int port;
    private final String listenAddress;
    // ... fields

    private ServerConfig(Builder builder) { /* copy from builder */ }

    public static Builder builder() { return new Builder(); }

    public String toYaml() { /* generate YAML string */ }

    public static final class Builder {
        private int port = 8000;
        private String listenAddress = "127.0.0.1";
        // ... defaults matching Go's DefaultServerConfig()

        public Builder port(int port) { this.port = port; return this; }
        public Builder listenAddress(String addr) { this.listenAddress = addr; return this; }
        // ...
        public ServerConfig build() { /* validate and construct */ }
    }
}
```

**Go's functional options (e.g. `WithPort(8000)`) do NOT translate idiomatically to Java.** Java has no first-class function composition like Go's `...ServerOption`. The Java builder pattern is the direct equivalent and is what Java developers expect.

**Exception: Backup/Rebuild/WALPrune options.** For these, consider whether the Go variadic option pattern should map to a Java builder OR to a simple request object with setters. The Go pattern uses variadic options because Go lacks builder syntax; in Java, a mutable options object with setters is simpler than a builder when the output is a request JSON payload:

```java
// Simple mutable options for maintenance APIs
public final class WALPruneOptions {
    private String tenantId;
    private String databaseName;
    private boolean dryRun;
    // ... setters return this for chaining
}
```

**Decision: Builder for config types (ServerConfig, EmbeddedConfig) because they generate YAML. Simple chained-setter objects for maintenance API options because they serialize to JSON.** This avoids over-abstracting the maintenance API options with a builder when a POJO with setters is more direct.

**Confidence: HIGH** -- Standard Java pattern, direct examination of Go API surface confirms the mapping.

---

## FFI Symbol Mapping: What Java Must Bind

### Server Lifecycle Symbols (NEW for v0.5.0)

| FFI Symbol | C Signature | JNA Mapping | Panama Mapping |
|------------|-------------|-------------|----------------|
| `chroma_server_start` | `void* (const char*)` | `Pointer chroma_server_start(String)` | `MethodHandle(ADDRESS, ADDRESS) -> ADDRESS` |
| `chroma_server_start_from_string` | `void* (const char*)` | `Pointer chroma_server_start_from_string(String)` | `MethodHandle(ADDRESS, ADDRESS) -> ADDRESS` |
| `chroma_server_port` | `int32 (void*)` | `int chroma_server_port(Pointer)` | `MethodHandle(ADDRESS) -> JAVA_INT` |
| `chroma_server_address` | `const char* (void*)` | `Pointer chroma_server_address(Pointer)` | `MethodHandle(ADDRESS) -> ADDRESS` |
| `chroma_server_persist_path` | `const char* (void*)` | `Pointer chroma_server_persist_path(Pointer)` | `MethodHandle(ADDRESS) -> ADDRESS` |
| `chroma_server_stop` | `int32 (void*)` | `int chroma_server_stop(Pointer)` | `MethodHandle(ADDRESS) -> JAVA_INT` |
| `chroma_server_free` | `void (void*)` | `void chroma_server_free(Pointer)` | `MethodHandle(ADDRESS) -> void` |

### Maintenance Symbols (NEW for v0.5.0)

| FFI Symbol | C Signature | JNA Mapping | Panama Mapping |
|------------|-------------|-------------|----------------|
| `chroma_embedded_rebuild_collection` | `char* (void*, const char*)` | `Pointer ...(Pointer, String)` | `MethodHandle(ADDRESS, ADDRESS) -> ADDRESS` |
| `chroma_embedded_compact_collection` | `char* (void*, const char*)` | `Pointer ...(Pointer, String)` | `MethodHandle(ADDRESS, ADDRESS) -> ADDRESS` |
| `chroma_embedded_compact_all` | `char* (void*, const char*)` | `Pointer ...(Pointer, String)` | `MethodHandle(ADDRESS, ADDRESS) -> ADDRESS` |
| `chroma_embedded_prune_wal_collection` | `char* (void*, const char*)` | `Pointer ...(Pointer, String)` | `MethodHandle(ADDRESS, ADDRESS) -> ADDRESS` |
| `chroma_embedded_prune_wal_all` | `char* (void*, const char*)` | `Pointer ...(Pointer, String)` | `MethodHandle(ADDRESS, ADDRESS) -> ADDRESS` |

### Already Bound (No Changes)

| FFI Symbol | Status |
|------------|--------|
| `chroma_version` | Bound in both JNA and Panama |
| `chroma_get_last_error` | Bound in both JNA and Panama |
| `chroma_string_free` | Bound in both JNA and Panama |
| `chroma_embedded_start_from_string` | Bound in both JNA and Panama |
| `chroma_embedded_free` | Bound in both JNA and Panama |

### Key FFI Pattern: All Maintenance APIs Follow the Same Shape

Every maintenance symbol takes `(handle: void*, request_json: const char*) -> char*` and returns an allocated JSON string that must be freed with `chroma_string_free`. This is the exact same pattern already used for `chroma_embedded_start_from_string` but with JSON instead of YAML. The existing `lastError()` helper pattern in both JNA and Panama implementations handles the null-return-with-error-detail flow correctly.

**No struct mapping needed.** There are no C struct types crossing the FFI boundary. All structured data passes as JSON strings. This is intentional in the Rust shim design and eliminates the most complex part of JNA/Panama FFI work.

**Confidence: HIGH** -- Direct examination of Rust shim source (`shim/src/lib.rs`) confirms all maintenance symbols use JSON string I/O.

---

## Response Type Mapping: JSON to Java

The maintenance FFI calls return JSON strings that need deserialization into Java types. These types live in the `core` module (shared between JNA and Panama).

### Types to Create in `core`

| Go Type | Java Type | Fields | Notes |
|---------|-----------|--------|-------|
| `RebuildCollectionResult` | `RebuildResult` | collectionId, name, tenantId, databaseName, precheck, wouldRebuild, rebuilt, recordsScanned, vectorsReindexed, durationMs, backupPath, warnings | List<String> for warnings |
| `CompactionCollectionResult` | `CompactionCollectionResult` | collectionId, name, tenantId, databaseName, pendingOpsBefore, pendingOpsAfter, pendingOpsBeforeError, pendingOpsAfterError, error | Long (nullable) for optional uint64 fields |
| `CompactionResult` | `CompactionResult` | collectionCount, durationMs, pendingOpsBeforeTotal, pendingOpsAfterTotal, collections | List of CompactionCollectionResult |
| `WALPruneCollectionResult` | `WALPruneCollectionResult` | collectionId, name, tenantId, databaseName, safeSeqCutoff, candidateSeqMin/Max, prunedSeqMin/Max, candidateCount/Bytes, prunedCount/Bytes, error | Long (nullable) for optional fields |
| `WALPruneResult` | `WALPruneResult` | collectionCount, durationMs, dryRun, vacuumRequested, vacuumExecuted, warning, candidateCountTotal, candidateBytesTotal, prunedCountTotal, prunedBytesTotal, collections | List of WALPruneCollectionResult |

### Gson Field Naming

The JSON field names from the Rust shim use `snake_case` (e.g., `collection_id`, `duration_ms`). Gson supports `@SerializedName` annotations or a global `FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES` setting.

**Decision: Use `FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES` on a shared `Gson` instance.** This avoids annotating every field in every response type. Create one `Gson` instance in the `core` module's utility class and reuse it.

```java
// In core module
public final class JsonCodec {
    private static final Gson GSON = new GsonBuilder()
        .setFieldNamingPolicy(FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES)
        .create();

    public static <T> T fromJson(String json, Class<T> type) {
        return GSON.fromJson(json, type);
    }

    public static String toJson(Object obj) {
        return GSON.toJson(obj);
    }
}
```

**Confidence: HIGH** -- Gson's `FieldNamingPolicy` is stable and well-documented.

---

## Gradle Dependency Changes

### core/build.gradle.kts (ADD Gson)

```kotlin
dependencies {
    api("com.google.code.gson:gson:2.13.2")

    testImplementation(platform("org.junit:junit-bom:5.11.4"))
    testImplementation("org.junit.jupiter:junit-jupiter")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
}
```

Gson is an `api` dependency (not `implementation`) because the response types use Gson's `@SerializedName` annotation in their public API, and JNA/Panama modules need to call `JsonCodec` from `core`.

### jna/build.gradle.kts and panama/build.gradle.kts

No new dependencies. They already depend on `api(project(":core"))` which transitively provides Gson.

**Confidence: HIGH** -- Standard Gradle multi-module dependency management.

---

## Testing Strategy

### No Mocking Libraries Needed

All Java tests are integration tests that call the real native library. The existing pattern is correct:

```java
String libPath = System.getenv("CHROMA_LIB_PATH");
Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");
```

Tests skip cleanly when the native library is not available. When it is available, they exercise real FFI calls. This is the ONLY meaningful test strategy for FFI code -- mocking the native layer would test Java boilerplate, not correctness.

### Test Patterns for New APIs

| API Category | Test Pattern | Key Assertions |
|-------------|-------------|----------------|
| Server lifecycle | Start server, verify port/address/URL, stop, close | Port > 0, address non-blank, URL format correct |
| Server config builder | Build config, verify YAML output | YAML string contains expected key-value pairs |
| Embedded maintenance | Start embedded, create collection, run maintenance, verify result | Non-null result, expected field values |
| Backup | Start embedded, backup to temp dir, verify manifest | Manifest file exists, field values match |
| Error handling | Invalid inputs, null handles, already-closed sessions | ChromaException or IllegalStateException thrown |

### JUnit 5 Features Already in Use (No Additions Needed)

- `@TempDir` -- temporary directories for persist paths and backup destinations
- `Assumptions.assumeTrue` -- skip tests when native lib unavailable
- `assertThrows` -- verify exception types
- `assertDoesNotThrow` -- verify idempotent close

**Confidence: HIGH** -- Pattern proven by existing scaffold tests.

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Jackson | 3 jars, ~1.8 MB for tiny JSON payloads | Gson (1 jar, ~300 KB, zero transitive deps) |
| SnakeYAML | Adds complexity for simple flat YAML generation | `StringBuilder`-based YAML formatting |
| Lombok | Annotation processing overhead for 5 trivial builders | Hand-rolled static inner `Builder` classes |
| Immutables / AutoValue | Annotation processing for simple final-field data classes | Plain Java classes with final fields |
| Mockito | Mocking FFI calls tests nothing useful | Integration tests with `Assumptions.assumeTrue` |
| AssertJ | Marginal assertion improvement, extra dependency | JUnit 5 built-in assertions |
| JNA `Structure` mapping | Not needed -- no C structs cross FFI boundary | JSON string serialization via Gson |
| jextract (Panama code generator) | Generates bindings from C headers; adds build complexity and generates code we'd need to customize | Hand-written `MethodHandle` lookups (already established pattern) |

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| Gson 2.13.2 | Java 17+ | Supports Java 9 modules. No issues with Java 22. |
| JNA 5.14.0 | Java 8+ | Stable on Java 17 and 22 toolchains. |
| JUnit 5.11.4 | Java 8+ | Full support for both Java 17 and 22 test toolchains. |
| Gradle 9+ | Java 17-22 | Current build config already handles dual toolchain (17 for core/jna, 22 for panama). |

---

## Stack Patterns by Module

**core module:**
- Response types (final-field classes with Gson `FieldNamingPolicy`)
- Config builders (static inner `Builder` with `toYaml()`)
- `JsonCodec` utility (shared `Gson` instance)
- `ChromaRuntime` interface (extended with server and maintenance methods)
- `ServerSession` type (new, mirrors `EmbeddedSession` pattern)
- Maintenance option types (simple POJOs with chained setters)

**jna module:**
- Extended `JnaBindings` interface with new `chroma_server_*` and maintenance symbols
- `JnaChromaRuntime` methods implementing new `ChromaRuntime` interface methods
- JSON serialization for maintenance requests, deserialization for responses

**panama module:**
- Additional `MethodHandle` fields for new symbols (resolved in `init()`)
- `PanamaChromaRuntime` methods implementing new `ChromaRuntime` interface methods
- Same JSON ser/de pattern as JNA (shared `core` types)

---

## Sources

- [JNA GitHub releases](https://github.com/java-native-access/jna/releases) -- JNA 5.18.1 is latest, 5.14.0 in use is sufficient
- [JNA Structures and Unions documentation](https://github.com/java-native-access/jna/blob/master/www/StructuresAndUnions.md) -- confirmed struct mapping NOT needed for this project
- [JEP 454: Foreign Function & Memory API](https://openjdk.org/jeps/454) -- Panama FFM finalized in JDK 22, HIGH confidence
- [Gson Maven Central](https://mvnrepository.com/artifact/com.google.code.gson/gson) -- version 2.13.2 verified, MEDIUM confidence (WebSearch)
- [JUnit 5 Release Notes](https://junit.org/junit5/docs/current/release-notes/index.html) -- 5.11.4 features confirmed, HIGH confidence
- [Jackson vs Gson comparison (Baeldung)](https://www.baeldung.com/jackson-vs-gson) -- size/complexity tradeoff confirmed, MEDIUM confidence
- Direct codebase examination: `shim/src/lib.rs`, `internal/runtime/*.go`, `java/**/*.java` -- HIGH confidence, primary source for FFI symbol signatures and patterns

---
*Stack research for: Java API Surface v0.5.0*
*Researched: 2026-03-21*
