---
phase: 06-core-foundation-types
plan: 01
subsystem: api
tags: [gson, java, pojo, json-deserialization, maintenance-results]

requires: []
provides:
  - "Gson 2.13.2 and SnakeYAML 2.6 dependencies in core module"
  - "JsonUtil shared Gson instance with LOWER_CASE_WITH_UNDERSCORES policy"
  - "7 result POJO classes for rebuild, compaction, WAL prune, backup operations"
  - "Package-private constructors and accessor-style methods on all POJOs"
affects: [06-core-foundation-types, 07-server-lifecycle, 08-embedded-maintenance, 09-backup-api, 10-server-maintenance]

tech-stack:
  added: [com.google.code.gson:gson:2.13.2, org.yaml:snakeyaml:2.6]
  patterns: [final-field-pojo-with-gson, package-private-constructor, accessor-style-methods, boxed-long-for-optional-numerics]

key-files:
  created:
    - java/core/src/main/java/tech/amikos/chroma/local/core/JsonUtil.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/RebuildCollectionResult.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/CompactionCollectionResult.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/CompactionResult.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/WALPruneCollectionResult.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/WALPruneResult.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/BackupFileMetadata.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/BackupManifest.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/RebuildCollectionResultTest.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/CompactionResultTest.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/WALPruneResultTest.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/BackupManifestTest.java
  modified:
    - java/core/build.gradle.kts
    - java/jna/build.gradle.kts

key-decisions:
  - "Used sourceCompatibility/targetCompatibility instead of strict toolchain for JDK portability"
  - "Warnings list in RebuildCollectionResult defaults to null (not empty list) when absent from JSON"

patterns-established:
  - "Final-field POJO pattern: public final class, private final fields, package-private no-arg constructor, accessor methods (name() not getName())"
  - "Boxed Long for optional numerics: Go *uint64 maps to Java Long (null = absent), primitive long for required numerics"
  - "JsonUtil shared Gson: package-private utility with LOWER_CASE_WITH_UNDERSCORES for all snake_case JSON mapping"

requirements-completed: [FOUND-01, FOUND-04]

duration: 9min
completed: 2026-03-22
---

# Phase 06 Plan 01: Core Foundation Types - Result POJOs Summary

**Gson/SnakeYAML dependencies added to core module with 7 result POJOs for rebuild, compaction, WAL prune, and backup operations, all deserializing from snake_case JSON via shared LOWER_CASE_WITH_UNDERSCORES policy**

## Performance

- **Duration:** 9 min
- **Started:** 2026-03-22T18:05:17Z
- **Completed:** 2026-03-22T18:14:25Z
- **Tasks:** 2
- **Files modified:** 14

## Accomplishments
- Added Gson 2.13.2 and SnakeYAML 2.6 as implementation dependencies to the core module
- Created JsonUtil with shared Gson instance using LOWER_CASE_WITH_UNDERSCORES naming policy for automatic snake_case mapping
- Implemented all 7 result POJO classes matching Go struct field names and types exactly
- 15 new tests across 4 test classes covering full JSON, missing optional fields, nested lists, and boxed Long null semantics

## Task Commits

Each task was committed atomically:

1. **Task 1: Add dependencies and create JsonUtil** - `ebc0d14` (feat)
2. **Task 2 RED: Failing tests for result POJOs** - `0ba3f50` (test)
3. **Task 2 GREEN: Implement 7 result POJOs** - `80fa161` (feat)

## Files Created/Modified
- `java/core/build.gradle.kts` - Added Gson 2.13.2, SnakeYAML 2.6 deps; switched to sourceCompatibility/targetCompatibility
- `java/jna/build.gradle.kts` - Switched to sourceCompatibility/targetCompatibility for JDK portability
- `java/core/src/main/java/tech/amikos/chroma/local/core/JsonUtil.java` - Package-private shared Gson instance
- `java/core/src/main/java/tech/amikos/chroma/local/core/RebuildCollectionResult.java` - 12 fields matching Go rebuild.go
- `java/core/src/main/java/tech/amikos/chroma/local/core/CompactionCollectionResult.java` - Boxed Long for optional pending ops
- `java/core/src/main/java/tech/amikos/chroma/local/core/CompactionResult.java` - Nested List<CompactionCollectionResult>
- `java/core/src/main/java/tech/amikos/chroma/local/core/WALPruneCollectionResult.java` - 5 boxed Long fields for sequence numbers
- `java/core/src/main/java/tech/amikos/chroma/local/core/WALPruneResult.java` - Nested List<WALPruneCollectionResult>
- `java/core/src/main/java/tech/amikos/chroma/local/core/BackupFileMetadata.java` - File metadata with String modifiedAt
- `java/core/src/main/java/tech/amikos/chroma/local/core/BackupManifest.java` - Nested List<BackupFileMetadata>
- `java/core/src/test/java/tech/amikos/chroma/local/core/RebuildCollectionResultTest.java` - 4 tests
- `java/core/src/test/java/tech/amikos/chroma/local/core/CompactionResultTest.java` - 4 tests
- `java/core/src/test/java/tech/amikos/chroma/local/core/WALPruneResultTest.java` - 3 tests
- `java/core/src/test/java/tech/amikos/chroma/local/core/BackupManifestTest.java` - 4 tests

## Decisions Made
- Used `sourceCompatibility`/`targetCompatibility` instead of strict `toolchain { languageVersion }` in core and jna build.gradle.kts to allow any JDK >= 17 to compile (the environment has JDK 18/22/26 but not 17)
- Warnings list in RebuildCollectionResult defaults to null when absent from JSON (consistent with Gson's default behavior for missing fields)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed Java toolchain detection for local build**
- **Found during:** Task 1 (build.gradle.kts compilation)
- **Issue:** Gradle toolchain `languageVersion.set(JavaLanguageVersion.of(17))` requires exact JDK 17 installed, but only JDK 18, 22, 25, 26 are available. This is a pre-existing environment issue (`make build-java` also failed before any changes).
- **Fix:** Changed core and jna modules from strict `toolchain { languageVersion }` to `sourceCompatibility`/`targetCompatibility` which allows any JDK >= 17 to cross-compile to Java 17 bytecode.
- **Files modified:** java/core/build.gradle.kts, java/jna/build.gradle.kts
- **Verification:** `gradle :core:compileJava` and `gradle :core:build` succeed
- **Committed in:** ebc0d14 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary to unblock all Java compilation. The change is functionally equivalent -- produces Java 17 bytecode from any JDK >= 17.

## Issues Encountered
None beyond the toolchain issue documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 7 result POJOs are ready for consumption by session methods in Phases 7-10
- JsonUtil provides the shared Gson instance for future option type `toJson()` serialization
- Core module compiles and tests pass (`gradle :core:build` exits 0)
- Plans 06-02 and 06-03 can proceed: config builders, FFI safety abstractions, session types

## Self-Check: PASSED

- All 12 created files verified on disk
- All 3 commits (ebc0d14, 0ba3f50, 80fa161) verified in git log
- `gradle :core:build` exits 0 with all 20 tests passing (15 new + 5 existing)
- Zero JNA/Panama imports in core module source

---
*Phase: 06-core-foundation-types*
*Completed: 2026-03-22*
