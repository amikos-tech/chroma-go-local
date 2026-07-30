---
gsd_state_version: 1.0
milestone: v0.5.0
milestone_name: Java API Surface
status: "Phase 10 shipped — PR #82"
stopped_at: Completed 10-02-PLAN.md
last_updated: "2026-03-28T13:39:05.763Z"
progress:
  total_phases: 5
  completed_phases: 5
  total_plans: 11
  completed_plans: 11
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-21)

**Core value:** Java and Go APIs must provide equivalent access to all Chroma runtime capabilities
**Current focus:** Phase 10 — server-maintenance (complete)

## Current Position

Phase: 10
Plan: Not started

## Performance Metrics

| Phase | Plan | Duration | Tasks | Files |
|-------|------|----------|-------|-------|
| 06 | 01 | 9min | 2 | 14 |
| Phase 06 P03 | 5min | 2 tasks | 7 files |
| Phase 06 P02 | 5min | 2 tasks | 13 files |
| 07 | 01 | 3min | 2 | 3 |
| 07 | 02 | 2min | 1 | 2 |
| 08 | 01 | 3min | 2 | 4 |
| 08 | 02 | 9min | 2 | 2 |
| Phase 09 P02 | 4min | 2 tasks | 4 files |
| Phase 10 P01 | 6min | 2 tasks | 6 files |
| 10 | 02 | 6min | 2 | 2 |

## Accumulated Context

### Decisions

- [v0.4.0]: Go subtree reorganization complete -- internal/runtime/ and internal/library/ with root facade
- [v0.5.0-scope]: Reuse existing chroma_* FFI symbols -- no Rust shim changes
- [v0.5.0-scope]: Java builder pattern for server configuration (not Go-style functional options)
- [v0.5.0-scope]: Both JNA and Panama backends kept in sync for full API
- [v0.5.0-roadmap]: 5 phases (6-10) derived from dependency chain: core types -> server lifecycle + embedded maintenance -> backup + server maintenance
- [Phase 06]: Used sourceCompatibility/targetCompatibility instead of strict toolchain for JDK portability
- [Phase 06]: Maintenance methods throw UnsupportedOperationException rather than using null callback slots -- simpler Phase 6 design, Phases 7-10 replace with actual wiring
- [Phase 06]: SnakeYAML BLOCK flow style with semantic golden tests for YAML output verification
- [Phase 06]: WALPruneOptions watermark() API takes both high and low in single call to prevent incomplete pairs
- [Phase 07]: serverFree and embeddedFree bypass callFfiVoid to avoid FFI lock in finally blocks
- [Phase 07]: Skipped port-already-bound and concurrent start tests -- flaky across OSes
- [Phase 07]: Used ServerConfigBuilder in integration tests to validate builder output against real FFI
- [Phase 08]: BiFunction<Long, String, T> callback slots for all 5 maintenance operations
- [Phase 08]: EmbeddedSession constructor expanded from 2 to 7 parameters
- [Phase 08]: Smoke tier tests only -- D-09 data-seeded tests deferred pending FUTURE-03 collection CRUD
- [Phase 08]: EmbeddedConfigBuilder used for test YAML instead of hand-written strings
- [Phase 10]: MaintenanceResult preserves operation result on partial failure (restart error) per D-03
- [Phase 10]: MaintenanceExecutor mirrors Go rebuild.go error matrix with Java exception semantics
- [Phase 10]: Server maintenance methods invalidate session after callback matching backup pattern
- [Phase 10]: Split null-option rejection tests per operation due to ServerSession closing on IllegalArgumentException
- [Phase 10]: HTTP data seeding via Chroma v2 REST API for server maintenance test verification

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

### Quick Tasks Completed

| # | Description | Date | Commit | Status | Directory |
|---|-------------|------|--------|--------|-----------|
| 260730-ggr | Fix issue #100: release workflow signs with refs/heads/main identity instead of refs/tags/<version> when dispatched from main | 2026-07-30 | 4a5d7aa | Verified | [260730-ggr-fix-issue-100-release-workflow-signs-wit](./quick/260730-ggr-fix-issue-100-release-workflow-signs-wit/) |

## Session Continuity

Last activity: 2026-07-30 - Completed quick task 260730-ggr: Fix issue #100: release workflow signs with refs/heads/main identity instead of refs/tags/<version> when dispatched from main

Last session: 2026-03-28T09:41:20Z
Stopped at: Completed 10-02-PLAN.md
Resume file: None
