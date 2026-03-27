# Phase 9: Backup API - Research

**Researched:** 2026-03-27
**Domain:** Java backup API — filesystem copy, manifest generation, session lifecycle
**Confidence:** HIGH

## Summary

Phase 9 implements backup functionality for both embedded and server sessions in Java. Unlike maintenance operations (Phase 8), backup involves zero FFI calls — the entire algorithm runs in wrapper-side Java code. The core backup algorithm (close runtime, copy persistence directory, write JSON manifest, restart runtime) is already proven in the Go implementation (`internal/runtime/backup.go`) and the Java implementation must replicate this pattern.

The key architectural challenge is that backup invalidates the session handle (it closes and restarts the runtime), so the session object itself cannot be reused. The CONTEXT.md decision D-03 resolves this by returning a `BackupResult<S>` containing both the manifest and a new session instance. The old session is invalidated. This differs from Go where the same `Server`/`Embedded` struct mutates its internal handle.

All three Phase 6 data types (`BackupOptions`, `BackupManifest`, `BackupFileMetadata`) already exist in `core` and are ready to use. The primary implementation work is: (1) a `BackupExecutor` utility in core that owns the filesystem algorithm, (2) a `BackupResult<S>` result type in core, (3) wiring backup callback slots into `EmbeddedSession` and `ServerSession`, and (4) backend lambdas in JNA and Panama that construct the backup callback.

**Primary recommendation:** Implement a `BackupExecutor` class in core that receives close/restart callbacks and owns the copy-manifest-restart algorithm. Backends inject their lifecycle hooks via the existing callback slot pattern. Tests use sentinel files pre-seeded in `@TempDir` to verify backup correctness.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Hybrid approach — core module owns 100% of the filesystem algorithm (directory copy, SHA256 hashing, manifest JSON writing). Backends inject close + restart callbacks via the existing slot pattern. No logic duplication across JNA and Panama.
- **D-02:** A utility class in core (e.g., `BackupExecutor`) implements the close -> copy -> write-manifest -> restart algorithm. Backends construct it with their lifecycle hooks.
- **D-03:** `backup()` returns a `BackupResult<S>` containing both the `BackupManifest` and a new session instance. The old session is invalidated (closed). Caller uses the new session going forward.
- **D-04:** Sessions remain immutable — `final long handle` is preserved. No mutable handle refactoring.
- **D-05:** Generic `BackupResult<S>` — one class with `manifest()` and `session()` getters. Returns `BackupResult<EmbeddedSession>` from embedded backup, `BackupResult<ServerSession>` from server backup.
- **D-06:** Session's `backup()` uses a `Function<BackupOptions, BackupResult<S>>` callback slot injected at construction time. The backend's lambda internally calls core's backup utility with the right restarter (a `Supplier<S>` that creates a fresh session from saved config). The Supplier is internal to the lambda — not exposed in the public API.
- **D-07:** User calls `session.backup(options)` cleanly — no internal plumbing exposed. The callback slot bundles the full close-copy-restart algorithm.
- **D-08:** Validation happens at `backup()` call time inside the core backup algorithm, not at `BackupOptions.build()` time. Embedded rejects `leaveStopped`, server rejects `leaveClosed`. Matches Go's `resolveBackupOptions` pattern.
- **D-09:** `BackupOptions` class stays unchanged from Phase 6 — no new builder subclasses or mode parameters.
- **D-10:** Filesystem verify + edge cases (B+D tier). Pre-seed a sentinel file in `@TempDir` before starting the session, call `backup()`, verify the sentinel appears in the backup output directory and manifest JSON is parseable with correct `fileCount`.
- **D-11:** Edge case tests: invalid destination (null/empty), non-empty destination directory, cross-mode option rejection (leaveStopped on embedded, leaveClosed on server).
- **D-12:** Identical test structure in both `:jna:test` and `:panama:test` modules — consistent with Phase 7 D-07 and Phase 8 D-10.

### Claude's Discretion
- Exact utility class naming and internal structure for the backup algorithm
- Whether `BackupResult` implements `AutoCloseable` (delegating to the contained session)
- Sentinel file content and naming in tests
- Order of implementation (core utility first vs session wiring first)

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BKUP-01 | `EmbeddedSession.backup(options)` performs directory copy with manifest and returns BackupManifest | Core BackupExecutor with Files.walkFileTree + SHA256 hashing; BackupResult wraps manifest + new session; EmbeddedSession gets backup callback slot |
| BKUP-02 | `ServerSession.backup(options)` performs stop-backup-restart cycle and returns BackupManifest | Same BackupExecutor; ServerSession gets backup callback slot; backend lambda handles stop+free -> copy -> restart cycle |
| BKUP-03 | `BackupOptions` builder supports destination, includeMetadata, leaveClosed/leaveStopped | Already implemented in Phase 6. D-08: mode validation at backup() call time, not build time. D-09: no changes to BackupOptions. |
| BKUP-04 | Integration tests verify backup creates valid output directory in both backends | Sentinel file pattern from Go tests; edge case tests for invalid inputs; identical test structure in JNA and Panama modules |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Java SE (java.nio.file) | 17+ | Files.walkFileTree, Path operations, file copying | JDK built-in, cross-platform directory traversal |
| Java SE (java.security) | 17+ | MessageDigest for SHA-256 hashing | JDK built-in, no external dependency needed |
| Gson | 2.13.2 | JSON manifest serialization (via existing JsonUtil) | Already in project as core dependency |
| JUnit 5 | 5.11.4 | Test framework with @TempDir support | Already in project |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| SnakeYAML | 2.6 | Config builder YAML output (existing) | Test setup only — building config YAML for embedded/server startup |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Files.walkFileTree | Apache Commons IO FileUtils.copyDirectory | Extra dependency for no benefit — JDK walkFileTree is sufficient and matches Go's filepath.WalkDir |
| MessageDigest (SHA-256) | Guava Hashing | Extra dependency — JDK MessageDigest is identical in functionality |

**Installation:** No new dependencies. All required libraries are already in `java/core/build.gradle.kts`.

## Architecture Patterns

### Recommended Project Structure (new files)
```
java/core/src/main/java/tech/amikos/chroma/local/core/
  BackupExecutor.java       # Core backup algorithm (close -> copy -> manifest -> restart)
  BackupResult.java         # Generic result type wrapping manifest + new session

java/core/src/test/java/tech/amikos/chroma/local/core/
  BackupExecutorTest.java   # Unit tests for backup algorithm (pure filesystem)
  BackupResultTest.java     # Unit tests for result type

java/jna/src/test/java/tech/amikos/chroma/local/jna/
  JnaEmbeddedBackupTest.java  # Integration: embedded backup via JNA
  JnaServerBackupTest.java    # Integration: server backup via JNA

java/panama/src/test/java/tech/amikos/chroma/local/panama/
  PanamaEmbeddedBackupTest.java  # Integration: embedded backup via Panama
  PanamaServerBackupTest.java    # Integration: server backup via Panama
```

### Pattern 1: BackupExecutor Utility (D-01, D-02)
**What:** A package-private utility class in core that owns the entire backup algorithm. It receives lifecycle callbacks (close + restart) from the calling backend. This centralizes all filesystem logic in one place.
**When to use:** Always — the only entry point for backup operations.
**Example:**
```java
// BackupExecutor — core module, package-private
final class BackupExecutor {

    static <S> BackupResult<S> execute(
            String mode,           // "embedded" or "server"
            String persistPath,    // source persist directory
            BackupOptions options,
            Runnable closeAction,
            Supplier<S> restartAction) {

        // 1. Validate options (mode-specific rejection)
        // 2. Resolve and validate destination path
        // 3. Ensure destination is empty
        // 4. closeAction.run()  -- close the runtime
        // 5. Copy persistence directory with optional SHA256
        // 6. Write manifest JSON
        // 7. If !leaveClosed/!leaveStopped: S newSession = restartAction.get()
        // 8. Return BackupResult<S>(manifest, newSession)
    }
}
```

### Pattern 2: BackupResult Generic Type (D-03, D-05)
**What:** A simple final generic class that wraps both the `BackupManifest` and the new session after restart.
**When to use:** Return type from `session.backup(options)`.
**Example:**
```java
public final class BackupResult<S> {
    private final BackupManifest manifest;
    private final S session;  // null if leaveClosed/leaveStopped

    public BackupResult(BackupManifest manifest, S session) {
        this.manifest = Objects.requireNonNull(manifest);
        this.session = session;  // may be null
    }

    public BackupManifest manifest() { return manifest; }
    public S session() { return session; }  // null means left closed/stopped
}
```

### Pattern 3: Callback Slot for Backup (D-06, D-07)
**What:** Sessions receive a `Function<BackupOptions, BackupResult<S>>` callback at construction time, matching the existing callback injection pattern used for maintenance operations.
**When to use:** Session wiring in both EmbeddedSession and ServerSession.
**Example for EmbeddedSession:**
```java
// EmbeddedSession constructor adds one more parameter:
private final Function<BackupOptions, BackupResult<EmbeddedSession>> backupAction;

public BackupResult<EmbeddedSession> backup(BackupOptions options) {
    ensureOpen();
    if (options == null) throw new IllegalArgumentException("options is required");
    return backupAction.apply(options);
}
```

### Pattern 4: Backend Lambda Construction (D-06)
**What:** Each backend (`doStartEmbedded`, `doStartServer`) constructs the backup callback lambda that internally:
  1. Captures the config YAML used to start the session
  2. Calls `BackupExecutor.execute(mode, persistPath, options, closeRunnable, restartSupplier)`
  3. The `closeRunnable` closes the current FFI handle
  4. The `restartSupplier` starts a new session from saved config

**Example for JNA embedded:**
```java
// In JnaChromaRuntime.doStartEmbedded():
String savedYaml = configYaml;  // capture for restart
Function<BackupOptions, BackupResult<EmbeddedSession>> backupAction = opts -> {
    String persistPath = /* read from handle before close */;
    return BackupExecutor.execute(
        "embedded",
        persistPath,
        opts,
        () -> embeddedFree(handle),          // close action
        () -> doStartEmbedded(savedYaml));   // restart action
};
```

### Anti-Patterns to Avoid
- **Duplicating filesystem logic in backends:** The entire copy/hash/manifest algorithm belongs in core's `BackupExecutor`. Backends only provide close + restart lambdas.
- **Mutating session handles:** D-04 locks `final long handle`. Backup creates a new session instead of mutating the existing one.
- **Validating mode options in BackupOptions.build():** D-08 explicitly places mode validation in the backup algorithm, not at build time. This matches Go's `resolveBackupOptions` pattern.
- **Using BackupOptions.toJson() for backup execution:** The `toJson()` method exists for serialization symmetry, not for driving the backup algorithm. BackupExecutor reads fields directly.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Directory tree copy | Manual byte-copy loops | `Files.walkFileTree` with `FileVisitor` | Handles nested dirs, permission preservation, symlink detection |
| SHA-256 file hashing | Custom hash function | `MessageDigest.getInstance("SHA-256")` with streaming | JDK standard, handles large files via streaming |
| JSON manifest writing | String concatenation | `JsonUtil.toJson()` with Gson pretty-printing | Already configured with `LOWER_CASE_WITH_UNDERSCORES` naming policy |
| Path containment check | String prefix matching | `Path.relativize()` + check for `..` prefix | Handles edge cases (symlinks, trailing slashes, normalized paths) |
| Empty directory validation | Manual stat + readdir | `Files.list(path).findFirst()` | Concise, auto-closeable stream |

**Key insight:** Go's backup implementation (`internal/runtime/backup.go`) solves every edge case that matters: path containment validation, symlink rejection, empty-dir enforcement, source-path-not-exists handling. The Java implementation should replicate these checks faithfully using JDK equivalents, not invent new validation logic.

## Common Pitfalls

### Pitfall 1: Gson Field Naming for Manifest Serialization
**What goes wrong:** Gson with `LOWER_CASE_WITH_UNDERSCORES` policy converts camelCase Java fields to snake_case JSON keys. The `BackupManifest` class uses Java field names (e.g., `schemaVersion`) which Gson maps to `schema_version`. If the `BackupExecutor` creates a manifest with misnamed fields, deserialization in tests will fail silently (fields stay null/zero).
**Why it happens:** BackupManifest was designed for Gson deserialization from FFI JSON. For backup, we're constructing it in Java and serializing to disk.
**How to avoid:** Use the existing `BackupManifest` class with its package-private constructor. Either add a builder/factory method, or construct a JSON object manually and write it. Verify in tests that `JsonUtil.fromJson(readFile, BackupManifest.class)` round-trips correctly.
**Warning signs:** `fileCount` is 0 when it should be non-zero; fields are null after deserialization.

### Pitfall 2: BackupManifest Construction
**What goes wrong:** `BackupManifest` currently has only a package-private no-arg constructor (for Gson deserialization). There is no public constructor or builder for creating a manifest from scratch.
**Why it happens:** Phase 6 designed BackupManifest for deserialization, not construction.
**How to avoid:** Add a package-private constructor with all fields, or add a static factory method in `BackupManifest`. Since `BackupExecutor` lives in the same package (`core`), it can access package-private constructors. Alternatively, construct the manifest as a `Map<String, Object>` and serialize with Gson.
**Warning signs:** Compile errors when trying to construct BackupManifest in BackupExecutor.

### Pitfall 3: Session Handle Becomes Invalid After Backup Close
**What goes wrong:** After `BackupExecutor` calls the close action, the old session's `handle` field is still non-zero but the native handle is freed. If anything tries to use the old session, it will segfault or corrupt memory.
**Why it happens:** D-04 says handles are `final` — they cannot be zeroed out.
**How to avoid:** The backup callback must mark the old session as closed via its `AtomicBoolean` closed flag before calling `BackupExecutor.execute()`. The callback lambda has access to the session's internal state because it's constructed in the backend. Alternatively, the close action passed to BackupExecutor should flip the session's closed flag.
**Warning signs:** `IllegalStateException("session is closed")` after backup, or segfaults if the guard is missing.

### Pitfall 4: Persist Path Retrieval Timing
**What goes wrong:** The persist path must be read from the session before close. After close, accessors throw `IllegalStateException`.
**Why it happens:** `EmbeddedSession` does not currently expose `persistPath()` (only `ServerSession` does). The persist path is needed by BackupExecutor to know what to copy.
**How to avoid:** The backend lambda captures the persist path before passing it to BackupExecutor. For embedded, the persist path is known from the config YAML. For server, `session.persistPath()` is called before close. The backend lambda constructs everything upfront.
**Warning signs:** `IllegalStateException` when trying to read persist path after close.

### Pitfall 5: Windows @TempDir Cleanup Failures
**What goes wrong:** JUnit's `@TempDir` cleanup fails on Windows when the Chroma native library holds file locks on the persist directory.
**Why it happens:** The Rust runtime may keep SQLite database files locked.
**How to avoid:** Use `@TempDir(cleanup = CleanupMode.NEVER)` for integration tests that interact with real FFI. This is the established pattern from Phase 7 (see `JnaServerLifecycleTest`).
**Warning signs:** Flaky test failures on Windows CI with "unable to delete" errors.

### Pitfall 6: Destination Inside Source Path
**What goes wrong:** If the backup destination is a subdirectory of the source persist path, `copyDirectory` would recurse infinitely.
**Why it happens:** User passes a path like `persistDir/backups/2024-01-01` as destination.
**How to avoid:** Replicate Go's `isWithinPath` containment check. Use `Path.relativize()` and verify the relative path starts with `..` (meaning it's outside). Check this before creating the destination directory.
**Warning signs:** Infinite loop or disk full error during backup.

## Code Examples

### Directory Copy with SHA-256 Hashing (from Go reference)
```java
// Java equivalent of Go's copyDirectory + copyFile pattern
// Source: internal/runtime/backup.go lines 515-580
static CopyResult copyDirectory(Path source, Path destination, boolean includeMetadata)
        throws IOException {
    Files.createDirectories(destination);
    List<BackupFileMetadata> files = new ArrayList<>();
    AtomicInteger fileCount = new AtomicInteger();
    AtomicLong totalBytes = new AtomicLong();

    Files.walkFileTree(source, new SimpleFileVisitor<>() {
        @Override
        public FileVisitResult preVisitDirectory(Path dir, BasicFileAttributes attrs)
                throws IOException {
            Path target = destination.resolve(source.relativize(dir));
            Files.createDirectories(target);
            return FileVisitResult.CONTINUE;
        }

        @Override
        public FileVisitResult visitFile(Path file, BasicFileAttributes attrs)
                throws IOException {
            if (Files.isSymbolicLink(file)) {
                throw new IOException("backup does not support symbolic links: " + file);
            }
            Path target = destination.resolve(source.relativize(file));
            String sha256 = copyFileWithHash(file, target);
            fileCount.incrementAndGet();
            totalBytes.addAndGet(attrs.size());
            if (includeMetadata) {
                // Build BackupFileMetadata for each file
            }
            return FileVisitResult.CONTINUE;
        }
    });
    return new CopyResult(fileCount.get(), totalBytes.get(), files);
}
```

### SHA-256 Streaming Hash (from Go reference)
```java
// Java equivalent of Go's io.Copy(io.MultiWriter(destinationFile, hash), sourceFile)
// Source: internal/runtime/backup.go lines 582-622
static String copyFileWithHash(Path source, Path destination) throws IOException {
    MessageDigest digest = MessageDigest.getInstance("SHA-256");
    try (InputStream in = Files.newInputStream(source);
         OutputStream out = Files.newOutputStream(destination,
                 StandardOpenOption.CREATE_NEW, StandardOpenOption.WRITE)) {
        byte[] buffer = new byte[8192];
        int read;
        while ((read = in.read(buffer)) != -1) {
            out.write(buffer, 0, read);
            digest.update(buffer, 0, read);
        }
    }
    // Preserve file permissions and modification time
    Files.setLastModifiedTime(destination,
            Files.getLastModifiedTime(source));
    return hexEncode(digest.digest());
}
```

### Manifest JSON Writing
```java
// Java equivalent of Go's writeManifest (json.MarshalIndent)
// Source: internal/runtime/backup.go lines 482-492
// Use Gson with pretty printing for manifest output
static void writeManifest(Path manifestPath, Map<String, Object> manifest)
        throws IOException {
    Gson gson = new GsonBuilder()
            .setFieldNamingPolicy(FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES)
            .setPrettyPrinting()
            .create();
    String json = gson.toJson(manifest) + "\n";
    Files.writeString(manifestPath, json, StandardOpenOption.CREATE_NEW);
}
```

### Path Containment Check
```java
// Java equivalent of Go's isWithinPath
// Source: internal/runtime/backup.go lines 687-697
static boolean isWithinPath(Path path, Path parent) {
    Path normalized = path.toAbsolutePath().normalize();
    Path normalizedParent = parent.toAbsolutePath().normalize();
    return normalized.startsWith(normalizedParent);
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| BackupOptions validated at build() time | Validation at backup() call time (D-08) | Phase 9 decision | Mode-specific flags checked in algorithm, not builder |
| Session.backup() returns BackupManifest | Session.backup() returns BackupResult<S> (D-03) | Phase 9 decision | Caller gets new session handle after backup |
| ServerSession.backup() stub throws UnsupportedOperationException | Real backup implementation | Phase 9 | Full backup support |

**Deprecated/outdated:**
- `ServerSession.backup(BackupOptions)` returning `BackupManifest` (the current stub) will change to return `BackupResult<ServerSession>`

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | JUnit 5.11.4 |
| Config file | `java/build.gradle.kts` (root test task config) |
| Quick run command | `cd java && gradle --no-daemon :jna:test :panama:test` |
| Full suite command | `make test-java` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| BKUP-01 | EmbeddedSession.backup() creates backup dir with manifest | integration | `cd java && gradle --no-daemon :jna:test --tests '*JnaEmbeddedBackupTest*' -x :panama:test` | Wave 0 |
| BKUP-02 | ServerSession.backup() stop-backup-restart cycle | integration | `cd java && gradle --no-daemon :jna:test --tests '*JnaServerBackupTest*' -x :panama:test` | Wave 0 |
| BKUP-03 | BackupOptions builder fields + mode validation | unit | `cd java && gradle --no-daemon :core:test --tests '*BackupOptionsTest*'` | Exists (Phase 6 tests) + Wave 0 for mode validation |
| BKUP-04 | Both JNA and Panama backends produce valid backup output | integration | `make test-java` | Wave 0 |

### Sampling Rate
- **Per task commit:** `cd java && gradle --no-daemon :core:test :jna:test :panama:test`
- **Per wave merge:** `make test-java`
- **Phase gate:** `make test-all` green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `java/core/src/test/java/.../BackupExecutorTest.java` -- covers BKUP-01 algorithm unit tests
- [ ] `java/core/src/test/java/.../BackupResultTest.java` -- covers BackupResult type
- [ ] `java/jna/src/test/java/.../JnaEmbeddedBackupTest.java` -- covers BKUP-01, BKUP-04
- [ ] `java/jna/src/test/java/.../JnaServerBackupTest.java` -- covers BKUP-02, BKUP-04
- [ ] `java/panama/src/test/java/.../PanamaEmbeddedBackupTest.java` -- covers BKUP-01, BKUP-04
- [ ] `java/panama/src/test/java/.../PanamaServerBackupTest.java` -- covers BKUP-02, BKUP-04

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Java | All compilation/tests | Yes | 26 | -- |
| Gradle | Build system | Yes | 9.4.1 | -- |
| Make | Test runner | Yes | 3.81 | -- |
| Rust shim (libchroma_shim) | FFI integration tests | Yes (built via `make build`) | -- | -- |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** None.

## Open Questions

1. **BackupManifest construction mechanism**
   - What we know: BackupManifest has only a package-private no-arg constructor (for Gson). BackupExecutor needs to construct one from computed values.
   - What's unclear: Whether to add a package-private all-args constructor, or build the manifest as a Map and serialize/deserialize.
   - Recommendation: Add a package-private all-args constructor to BackupManifest. Since BackupExecutor is in the same package, it can use it directly. This avoids the overhead and fragility of Map-based construction.

2. **EmbeddedSession persist path access**
   - What we know: ServerSession has `persistPath()` accessor. EmbeddedSession does not expose persist path.
   - What's unclear: Whether to add a `persistPath()` accessor to EmbeddedSession or let the backend lambda extract it from the config YAML.
   - Recommendation: The backend lambda should capture the persist path from the config at session creation time (it's known from the YAML). No need to add a new accessor. This keeps EmbeddedSession's public API unchanged.

3. **BackupResult and leaveClosed/leaveStopped**
   - What we know: When `leaveClosed=true` (embedded) or `leaveStopped=true` (server), backup does not restart. The returned BackupResult needs to communicate that no new session exists.
   - What's unclear: Whether `session()` returns null, or whether BackupResult should have an `isSessionAvailable()` predicate.
   - Recommendation: `session()` returns null when left closed/stopped. Javadoc documents this clearly. Simple and matches the nullable pattern.

## Sources

### Primary (HIGH confidence)
- `internal/runtime/backup.go` -- Go reference implementation, full algorithm source
- `internal/runtime/backup_test.go` -- Go test patterns (sentinel file, edge cases)
- `java/core/src/main/java/tech/amikos/chroma/local/core/BackupOptions.java` -- Existing Phase 6 type
- `java/core/src/main/java/tech/amikos/chroma/local/core/BackupManifest.java` -- Existing Phase 6 type
- `java/core/src/main/java/tech/amikos/chroma/local/core/BackupFileMetadata.java` -- Existing Phase 6 type
- `java/core/src/main/java/tech/amikos/chroma/local/core/EmbeddedSession.java` -- Current session (7 constructor params)
- `java/core/src/main/java/tech/amikos/chroma/local/core/ServerSession.java` -- Current session with backup stub
- `java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java` -- JNA backend
- `java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java` -- Panama backend

### Secondary (MEDIUM confidence)
- Phase 8 test patterns (JnaEmbeddedMaintenanceTest, PanamaEmbeddedMaintenanceTest) -- established integration test structure

## Project Constraints (from CLAUDE.md)

- **Conventional commits** required for all commits
- **Radical simplicity** -- keep architecture and implementation simple
- **No verbose comments** -- code and function/variable names should be self-explanatory
- **No scope creep** -- do exactly what is asked, nothing more
- **Build commands:** `make test-java` for Java tests, `make test-all` for full suite
- **Linting:** `gradle --no-daemon :core:check :jna:check :panama:check`
- **No cgo** -- pure Go FFI via purego (not relevant to Java changes but project-wide principle)
- **Facade pattern** -- root Go package re-exports; Java equivalent is the core module providing shared types

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all dependencies already in project, no new libraries needed
- Architecture: HIGH -- Go reference implementation is complete and well-tested, decisions locked in CONTEXT.md
- Pitfalls: HIGH -- identified from direct code analysis of existing session patterns and Go edge cases

**Research date:** 2026-03-27
**Valid until:** 2026-04-27 (stable domain, no external dependency changes expected)
