---
phase: 06-core-foundation-types
plan: 02
subsystem: api
tags: [java, builder-pattern, yaml, json-serialization, snakeyaml, config-builders, option-types]

requires:
  - phase: 06-core-foundation-types/01
    provides: "Gson + SnakeYAML dependencies, JsonUtil shared instance"
provides:
  - "5 option/request types with nested Builders and toJson() serialization"
  - "ServerConfigBuilder producing YAML matching Go's DefaultServerConfig().toYAML()"
  - "EmbeddedConfigBuilder producing YAML matching Go's DefaultEmbeddedConfig().toYAML()"
  - "rawYaml() escape hatch on both config builders"
  - "Strict validation at build() time for all builders"
affects: [07-server-lifecycle, 08-embedded-maintenance, 09-backup-api, 10-server-maintenance]

tech-stack:
  added: []
  patterns: [nested-builder-with-validation, config-builder-yaml-output, semantic-yaml-golden-tests, option-type-json-serialization]

key-files:
  created:
    - java/core/src/main/java/tech/amikos/chroma/local/core/RebuildOptions.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/WALPruneOptions.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/BackupOptions.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/ServerConfigBuilder.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/EmbeddedConfigBuilder.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/RebuildOptionsTest.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/WALPruneOptionsTest.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/BackupOptionsTest.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/CompactRequestTest.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/ServerConfigBuilderTest.java
    - java/core/src/test/java/tech/amikos/chroma/local/core/EmbeddedConfigBuilderTest.java
  modified:
    - java/core/src/main/java/tech/amikos/chroma/local/core/CompactCollectionRequest.java
    - java/core/src/main/java/tech/amikos/chroma/local/core/CompactAllRequest.java

key-decisions:
  - "SnakeYAML BLOCK flow style with default scalar representation for YAML output; golden tests parse semantically to avoid quoting brittleness"
  - "WALPruneOptions watermark() API takes both high and low in a single call, preventing incomplete pair configuration at the API level"
  - "Boxed Long for nullable policy fields in WALPruneOptions matches Go's *uint64 omitempty pattern"

patterns-established:
  - "Nested Builder pattern: public static class Builder with fluent setters, validation at build(), private constructor on outer class"
  - "Option type toJson() via JsonUtil.toJson() with LOWER_CASE_WITH_UNDERSCORES for automatic snake_case mapping"
  - "Config builder YAML output: LinkedHashMap for field order control, SnakeYAML DumperOptions.FlowStyle.BLOCK"
  - "Semantic YAML golden tests: parse output with SnakeYAML and compare map values instead of string equality"

requirements-completed: [FOUND-01, FOUND-02, FOUND-03]

duration: 5min
completed: 2026-03-22
---

# Phase 06 Plan 02: Option/Request Types and Config Builders Summary

**5 option/request types with nested Builders producing snake_case JSON for FFI, plus ServerConfigBuilder and EmbeddedConfigBuilder producing YAML matching Go defaults field-for-field**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-22T18:17:56Z
- **Completed:** 2026-03-22T18:23:32Z
- **Tasks:** 2
- **Files modified:** 13

## Accomplishments
- Created RebuildOptions, WALPruneOptions, BackupOptions, CompactCollectionRequest, CompactAllRequest with nested Builders and strict validation
- Built ServerConfigBuilder producing YAML with all 6 required fields plus conditional CORS and OTel sections
- Built EmbeddedConfigBuilder producing YAML with 3 fields matching Go's DefaultEmbeddedConfig()
- Both config builders support rawYaml() escape hatch overriding all other fields
- 49 tests across 6 test classes covering validation rejection, JSON serialization, and semantic YAML verification

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Failing tests for option/request types** - `be2a275` (test)
2. **Task 1 GREEN: Implement 5 option/request types** - `01507dd` (feat)
3. **Task 2 RED: Failing tests for config builders** - `4b6d0c6` (test)
4. **Task 2 GREEN: Implement config builders** - `5503665` (feat)

## Files Created/Modified
- `java/core/src/main/java/tech/amikos/chroma/local/core/RebuildOptions.java` - Rebuild options with name, keepBackup=true default, precheck
- `java/core/src/main/java/tech/amikos/chroma/local/core/WALPruneOptions.java` - WAL prune options with boxed Long policies, watermark pair validation
- `java/core/src/main/java/tech/amikos/chroma/local/core/BackupOptions.java` - Backup options with destinationPath, includeMetadata, leave flags
- `java/core/src/main/java/tech/amikos/chroma/local/core/CompactCollectionRequest.java` - Compact single collection with name, tenantId, databaseName
- `java/core/src/main/java/tech/amikos/chroma/local/core/CompactAllRequest.java` - Compact all with optional tenantId, databaseName
- `java/core/src/main/java/tech/amikos/chroma/local/core/ServerConfigBuilder.java` - Fluent builder producing YAML for server config
- `java/core/src/main/java/tech/amikos/chroma/local/core/EmbeddedConfigBuilder.java` - Fluent builder producing YAML for embedded config
- `java/core/src/test/java/tech/amikos/chroma/local/core/RebuildOptionsTest.java` - 8 tests
- `java/core/src/test/java/tech/amikos/chroma/local/core/WALPruneOptionsTest.java` - 12 tests
- `java/core/src/test/java/tech/amikos/chroma/local/core/BackupOptionsTest.java` - 5 tests
- `java/core/src/test/java/tech/amikos/chroma/local/core/CompactRequestTest.java` - 7 tests
- `java/core/src/test/java/tech/amikos/chroma/local/core/ServerConfigBuilderTest.java` - 11 tests
- `java/core/src/test/java/tech/amikos/chroma/local/core/EmbeddedConfigBuilderTest.java` - 6 tests

## Decisions Made
- Used SnakeYAML BLOCK flow style with default scalar representation; golden tests parse YAML semantically (map comparison) to avoid quoting brittleness between Go's %q and SnakeYAML's output
- WALPruneOptions.watermark(highBytes, lowBytes) takes both values in one call, preventing incomplete watermark pair at the API level
- Used boxed Long for nullable numeric fields in WALPruneOptions to match Go's *uint64 omitempty semantics (null = absent in Gson output)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 5 option/request types ready for session maintenance methods in Phases 7-10
- Config builders ready for startServer(yaml) and startEmbedded(yaml) in Phases 7-8
- Plan 06-03 can proceed: AbstractChromaRuntime FFI safety, ServerSession, ChromaRuntime extension

## Self-Check: PASSED

- All 13 created/modified files verified on disk
- All 4 commits (be2a275, 01507dd, 4b6d0c6, 5503665) verified in git log
- `gradle :core:build` exits 0 with all tests passing
- Zero JNA/Panama imports in core module source
