---
gsd_state_version: 1.0
milestone: v0.5.0
milestone_name: Java API Surface
status: Phase 11 in progress — Plan 2 ready to execute
stopped_at: Completed 11-01-PLAN.md
last_updated: "2026-08-02T11:48:45.421Z"
last_activity: 2026-08-02
progress:
  total_phases: 8
  completed_phases: 5
  total_plans: 15
  completed_plans: 12
  percent: 80
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-21)

**Core value:** Java and Go APIs must provide equivalent access to all Chroma runtime capabilities
**Current focus:** Phase 11 — migrate-rust-shim-from-chroma-1-5-5-to-1-5-9

## Current Position

Phase: 11 (migrate-rust-shim-from-chroma-1-5-5-to-1-5-9) — EXECUTING
Plan: 2 of 4

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
| Phase 11 P01 | 4min | 2 tasks | 4 files |

## Accumulated Context

### Roadmap Evolution

- Phase 11 added: Migrate Rust shim from Chroma 1.5.5 to 1.5.9
- Phase 12 added: Validate Chroma 1.5.9 cross-version data and binding compatibility
- Phase 12 edited: defined COMPAT-01 through COMPAT-04, five success criteria, and phase-local compatibility research
- Phase 11 edited: defined UPG-01 through UPG-04, five success criteria, and phase-local upgrade research
- Phase 13 added: Expose deferred Chroma 1.5.9 APIs with backward-compatible Go, JNA, and Panama bindings

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
- [Phase 11]: Resolved Chroma 1.5.9 only through the targeted fastrace 0.7.8 update, preserving unrelated locked packages. — Constrained dependency drift to the validated migration graph.
- [Phase 11]: Declared Rust 1.88 as the source-build MSRV and deferred the successful locked all-targets compile gate to Plan 11-02. — The known private delete-signature adaptation must land before compilation can pass.

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

### Quick Tasks Completed

| # | Description | Date | Commit | Status | Directory |
|---|-------------|------|--------|--------|-----------|
| 260730-ggr | Fix issue #100: release workflow signs with refs/heads/main identity instead of refs/tags/<version> when dispatched from main | 2026-07-30 | 4a5d7aa | Verified | [260730-ggr-fix-issue-100-release-workflow-signs-wit](./quick/260730-ggr-fix-issue-100-release-workflow-signs-wit/) |
| 260730-p2k | Fix issue #96: replace arduino/setup-protoc (deprecated Node 20 runtime) with chroma-core/setup-protoc fork at all 4 call sites | 2026-07-30 | 425fb9c | N/A | [260730-p2k-replace-arduino-setup-protoc-with-chroma](./quick/260730-p2k-replace-arduino-setup-protoc-with-chroma/) |
| 260730-pii | Install protoc inline from its GitHub release with sha256 verification; vendoring approach superseded during code review (GPL-3.0 bundle + release-backfill regression) | 2026-07-30 | bc8200c, 2223396 | Verified — PR #105 | [260730-pii-address-setup-protoc-fork-availability-u](./quick/260730-pii-address-setup-protoc-fork-availability-u/) |
| 260730-sz8 | Partial fix for issue #97: CDN cache purge step silently skipped when Cloudflare credentials are unset; now always runs and emits a ::warning:: annotation naming the missing credential. Setting real CF_ZONE_ID/CLOUDFLARE_API_TOKEN values and fixing the Cloudflare cache-rule TTL are still outstanding. | 2026-07-30 | e22f1a0 | Verified — PR #107 | [260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea](./quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/) |
| 260731-dgm | Address issue #97 review findings: make all Cloudflare purge failures diagnostic and non-fatal, require success:true, add remediation, lint workflows in CI, and correct the prior shell rationale | 2026-07-31 | fb0b734, ab47d82 | Verified — PR #107 | [260731-dgm-address-cdn-purge-reliability-diagnostic](./quick/260731-dgm-address-cdn-purge-reliability-diagnostic/) |
| 260731-eho | Address CDN purge retry-body pollution and diagnostics, and introduce standalone workflow linting; its intended yamllint apt pin did not displace preinstalled yamllint 1.38.0 and is corrected by 260731-fqz | 2026-07-31 | c3a0e3a | Verified — PR #107 | [260731-eho-address-cdn-purge-and-ci-workflow-lint-f](./quick/260731-eho-address-cdn-purge-and-ci-workflow-lint-f/) |
| 260731-fqz | Replace the ineffective yamllint apt path with a package-install-free, version-visible workflow lint contract and simplify Cloudflare cache purging into a structurally best-effort step | 2026-07-31 | 4174a8b | Verified — PR #107 | [260731-fqz-address-ci-lint-reproducibility-and-rele](./quick/260731-fqz-address-ci-lint-reproducibility-and-rele/) |
| 260731-j64 | Address release purge diagnostics, lint documentation and Windows parity, resilient lint tool versions, and workflow lint hardening | 2026-07-31 | 4743852 | Needs Review — PR #107 | [260731-j64-address-release-purge-diagnostics-lint-d](./quick/260731-j64-address-release-purge-diagnostics-lint-d/) |
| 260731-nkz | Validate and address release workflow review findings 1-10 | 2026-07-31 | f9d4e03 | Needs Review — PR #107 | [260731-nkz-validate-and-address-release-workflow-re](./quick/260731-nkz-validate-and-address-release-workflow-re/) |
| 260731-sjq | Protect published `v*` tags from updates and deletions for issue #99 | 2026-07-31 | ruleset 20138470 | Verified | [260731-sjq-review-and-address-issue-99-by-protectin](./quick/260731-sjq-review-and-address-issue-99-by-protectin/) |

## Session Continuity

Last activity: 2026-08-02

Last session: 2026-08-02T11:48:45.413Z
Stopped at: Completed 11-01-PLAN.md
Resume file: None
