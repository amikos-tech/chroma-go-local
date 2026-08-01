# Phase 12: Validate Chroma 1.5.9 Cross-Version Data and Binding Compatibility - Research

**Researched:** 2026-08-01
**Domain:** Persisted-data migration, backup/maintenance safety, cross-language FFI validation
**Confidence:** HIGH for the required test shape; LOW for compatibility outcome until a real 1.5.5 fixture is exercised

## Summary

The Phase 11 feasibility probe proves that the 1.5.9-backed shim can compile and operate on fresh data. It does not prove the user-critical case: opening a directory created by the released 1.5.5-backed shim.

Static upstream inspection suggests a low format-change risk because no local SQLite migrations or changes in the primary local persistence managers were found between the tags. That is useful evidence, not a substitute for a fixture test. SQLite metadata, HNSW files, WAL state, and maintenance operations must be exercised together.

**Primary recommendation:** Make a small, deterministic 1.5.5 fixture the center of this phase. Back it up before any 1.5.9 write, validate read behavior first, then mutate/restart, run maintenance operations, and document rather than assume reverse rollback support.

<phase_requirements>
## Phase Requirements

| ID | Description | Research support |
|----|-------------|------------------|
| COMPAT-01 | Open, read, query, mutate, and reopen 1.5.5 data with 1.5.9 | This is the main gap left by the fresh-data feasibility probe |
| COMPAT-02 | Verify backup and maintenance behavior on upgraded data | These operations touch persisted files and stop/restart ownership boundaries |
| COMPAT-03 | Verify all bindings on supported CI platforms | The local probe covered all bindings on one environment only |
| COMPAT-04 | Document backup, rollback, toolchain, and deferred APIs | Reverse compatibility and operational recovery must not be implied without evidence |
</phase_requirements>

## What Is Known

### Proven in the disposable 1.5.9 probe

- the migrated Rust graph compiles
- fresh embedded/server data works through the existing Go surface
- 42 Rust unit tests and 2 Rust FFI integration tests pass
- JNA and Panama smoke suites pass

### Suggested by source inspection

- no local SQLite migrations were added between Chroma 1.5.5 and 1.5.9
- the examined local persistent HNSW, local segment manager, and SQLite log paths did not change
- the resolved HNSW revision did not change in the probe

### Not yet proven

- a 1.5.5 data directory opens correctly under 1.5.9
- data remains correct after the first 1.5.9 write and restart
- backup/restore and maintenance work on upgraded data
- a directory touched by 1.5.9 can be reopened by 1.5.5
- every supported CI operating system and Java binding behaves the same

## Fixture Contract

The fixture should be deliberately small but cover different persisted surfaces. A useful minimum is:

| Fixture element | Purpose |
|-----------------|---------|
| Two named collections with captured UUIDs | Verify collection identity and enumeration |
| At least six records | Exercise more than a single HNSW node and allow filtered subsets |
| Fixed-dimensional embeddings with known nearest neighbors | Detect query-result drift |
| Documents including Unicode and empty-adjacent cases | Exercise document storage/encoding |
| String, integer, float, and boolean metadata | Exercise SQLite metadata typing and filters |
| One updated record | Carry an overwrite/update state across versions |
| One deleted record | Carry tombstone/WAL effects across versions |
| Non-empty WAL state where the public API permits it | Exercise prune/compaction inputs |

The fixture must include a human-readable manifest containing:

- exact producer revision and Chroma tag
- creation command
- collection IDs and expected counts
- record IDs and expected documents/metadata
- representative query inputs and expected ordered IDs or distance tolerances
- checksums for persisted fixture files or archive

### Fixture production options

| Option | Benefit | Cost/Risk |
|--------|---------|-----------|
| Commit a small generated fixture plus its generator | Fast and deterministic in ordinary CI | Binary changes require provenance and checksum review |
| Build 1.5.5 and generate on every test run | Maximum reproducibility from source | Doubles a large Rust build and depends on old graph availability |
| Generate in a dedicated scheduled/release job | Keeps regular CI lighter | Compatibility can regress between scheduled runs |

Recommendation: commit a minimal fixture and a reproducible generator, then add a separate verification that can regenerate and compare its manifest when dependency/network cost is acceptable. The discuss stage should lock this choice before planning.

## Compatibility Sequence

Order matters because the first 1.5.9 write may make reverse rollback impossible.

```text
1. Generate pristine data with 1.5.5
2. Copy/backup the pristine directory
3. Open the working copy with 1.5.9 in read paths
4. Compare IDs, counts, metadata, filters, and queries
5. Mutate with 1.5.9
6. Stop and reopen with 1.5.9
7. Run maintenance operations and recheck data
8. Test 1.5.5 rollback separately from the untouched and touched copies
9. Restore the pre-upgrade backup and verify recovery
```

The untouched backup is the control. Never use it as the working directory for mutation or maintenance tests.

## Required Test Matrix

### Version and state transitions

| Producer | Consumer | State | Expected gate |
|----------|----------|-------|---------------|
| 1.5.5 | 1.5.9 | Pristine/read-only | Must open and match the manifest |
| 1.5.5 | 1.5.9 | Mutate then restart | Must preserve old and new records |
| 1.5.5 | 1.5.9 | Maintenance applied | Must remain queryable and internally consistent |
| 1.5.5 backup | 1.5.9 | Restored copy | Must reproduce pristine expectations |
| 1.5.9-touched | 1.5.5 | Reverse open | Observe and document; do not promise success in advance |

If reverse open fails cleanly, release guidance must say rollback means restoring the pre-upgrade backup, not pointing 1.5.5 at data already modified by 1.5.9.

### Data behavior assertions

- collection list, name, UUID, and count
- record get by stable IDs
- documents and typed metadata equality
- metadata and document filters
- nearest-neighbor query ordering with an explicit distance tolerance
- add, update/upsert, and delete after upgrade
- close/reopen durability after every mutation group

Avoid byte-for-byte comparison of database and HNSW files after valid operations. Logical results and integrity checks are the compatibility contract.

### Maintenance and recovery assertions

| Operation | Before/after assertion |
|-----------|------------------------|
| Backup | Destination and manifest exist; restored copy opens and matches expected data |
| Rebuild collection | Collection remains queryable with stable logical results |
| Compact collection/all | Counts and representative queries remain stable |
| Prune collection/all WAL | Intended WAL reduction occurs without losing records |
| Server maintenance | Server stops, temporary embedded operation completes, server restarts and responds |
| Failure cleanup | Handles close once, errors remain actionable, original backup remains untouched |

### Binding ownership

| Surface | Minimum responsibility |
|---------|------------------------|
| Rust FFI tests | Direct lifecycle, error slot, and native operation coverage |
| Go tests | Fixture read/query/mutate/reopen and public maintenance APIs |
| Java JNA | Start/open, backup/maintenance callbacks, close and restart smoke against upgraded data |
| Java Panama | The same lifecycle contract as JNA on Java 22+ |
| CI OS matrix | Linux, macOS, and Windows paths used by current workflows |

Java does not currently expose all embedded record CRUD operations. Do not invent a new Java CRUD API for this phase. Seed and inspect data through the existing Go/Rust paths or the server HTTP API, then use Java for the lifecycle and maintenance surfaces it already owns.

## Failure Classification

The tests and release notes should distinguish these outcomes:

| Failure | Meaning | Release response |
|---------|---------|------------------|
| 1.5.9 cannot open pristine 1.5.5 data | Forward compatibility failure | Block release |
| Open succeeds but logical results differ | Silent compatibility failure | Block release and preserve fixture |
| First 1.5.9 mutation corrupts/reorders expected state | Write compatibility failure | Block release |
| Maintenance breaks only upgraded data | Operational compatibility failure | Block release or explicitly remove affected operation from support |
| 1.5.5 cannot reopen 1.5.9-touched data | Reverse rollback unsupported | Require restore-from-backup guidance; not automatically a forward-release blocker |
| One binding or OS fails | Distribution compatibility gap | Block that artifact/platform until resolved |

## Recommended Plan Boundaries

### Task 1: Reproducible 1.5.5 fixture

- choose and document fixture production strategy
- create the producer and manifest/checksum path
- assert the fixture is actually produced by the 1.5.5 dependency graph

### Task 2: Forward upgrade and restart verification

- validate pristine reads before any write
- mutate through 1.5.9 and verify restart durability
- retain the untouched control copy

### Task 3: Maintenance, bindings, and rollback

- run backup, rebuild, compaction, WAL pruning, and server restart cases
- cover Rust, Go, JNA, and Panama on the CI matrix
- test the two rollback cases and write operational guidance

## Release Gate

Phase 12 is complete only when:

- every COMPAT requirement has an automated test or a version-controlled, reviewed manual procedure where automation is impossible
- the pristine fixture and expected manifest are reproducible
- forward upgrade and post-write restart pass
- backup restore is proven
- all supported bindings and CI platforms are green
- release notes make no reverse-rollback claim stronger than the test evidence

## Questions for Discuss Phase

1. Choose committed fixture plus generator, per-run generation, or a dedicated compatibility job.
2. Confirm whether reverse opening with 1.5.5 is informational or a hard release gate. Recommendation: informational, with backup restore as the supported rollback.
3. Confirm the minimum representative dataset and which query/filter results must be stable.
4. Decide whether cross-platform fixture generation is required or whether one canonical fixture can be consumed on every platform.

## Planner Checklist

- [ ] The plan creates data with a real 1.5.5 build, not a mocked schema.
- [ ] Read-only checks happen before any 1.5.9 mutation.
- [ ] The pristine copy and working copy cannot be confused.
- [ ] Logical assertions cover SQLite metadata, vector index results, and WAL-backed mutations.
- [ ] Maintenance operations run against upgraded data, not only a new database.
- [ ] Java coverage uses existing APIs and does not expand public CRUD scope.
- [ ] Rollback documentation matches the actual reverse-open and restore results.
