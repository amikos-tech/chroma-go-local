---
status: testing
phase: 10-server-maintenance
source: [10-01-SUMMARY.md, 10-02-SUMMARY.md]
started: 2026-03-28T10:00:00Z
updated: 2026-03-28T10:00:00Z
---

## Current Test

number: 1
name: Core module compiles with MaintenanceResult and MaintenanceExecutor
expected: |
  `gradle --no-daemon :core:compileJava` succeeds with no errors.
  MaintenanceResult.java and MaintenanceExecutor.java exist in core module.
awaiting: user response

## Tests

### 1. Core module compiles with MaintenanceResult and MaintenanceExecutor
expected: `gradle --no-daemon :core:compileJava` succeeds. MaintenanceResult.java and MaintenanceExecutor.java exist in java/core/src/main/java/tech/amikos/chroma/local/core/.
result: [pending]

### 2. All three modules compile (core + JNA + Panama)
expected: `gradle --no-daemon :core:check :jna:check :panama:check` succeeds with no compilation or test errors.
result: [pending]

### 3. JNA server maintenance integration tests pass
expected: `gradle --no-daemon :jna:test --tests '*ServerMaintenanceTest*'` runs 11 tests, all pass. Tests cover rebuild, compact, compactAll, pruneWAL, pruneAllWAL, throws-after-close, and 5 null-rejection tests.
result: [pending]

### 4. Panama server maintenance integration tests pass
expected: `gradle --no-daemon :panama:test --tests '*ServerMaintenanceTest*'` runs 11 tests, all pass. Identical coverage to JNA.
result: [pending]

### 5. Existing backup tests still pass after constructor change
expected: `gradle --no-daemon :jna:test --tests '*ServerBackupTest*'` and `gradle --no-daemon :panama:test --tests '*ServerBackupTest*'` both pass — the ServerSession constructor expansion from 7 to 12 params did not break backup wiring.
result: [pending]

### 6. Full Java test suite passes
expected: `make test-java` exits 0. All JNA and Panama tests pass including tests from prior phases (lifecycle, backup, embedded maintenance).
result: [pending]

## Summary

total: 6
passed: 0
issues: 0
pending: 6
skipped: 0
blocked: 0

## Gaps

[none yet]
