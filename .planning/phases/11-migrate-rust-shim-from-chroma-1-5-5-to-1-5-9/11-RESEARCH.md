# Phase 11: Migrate Rust shim from Chroma 1.5.5 to 1.5.9 - Research

**Researched:** 2026-08-01
**Domain:** Chroma internal Rust crates, Cargo dependency resolution, no-cgo FFI compatibility
**Confidence:** HIGH for source and dependency changes; MEDIUM for toolchain floor until the final lockfile is produced

## Summary

The upgrade is feasible and should remain a narrow Rust-shim migration. A temporary copy of this repository was moved from Chroma 1.5.5 to 1.5.9, its lockfile conflict was reconciled, and the one shim source break was adapted. Rust compilation, Go tests, Rust tests, and both Java binding smoke suites then passed against fresh data.

The main implementation work is small, but the upstream change surface is not: the two tags span 205 commits and 481 changed files. The phase therefore needs to preserve the evidence and explicit boundaries below rather than treating the change as nine tag substitutions.

**Primary recommendation:** Upgrade the dependency graph and known delete signature in Phase 11, keep the C/Go/Java API stable, correct the documented toolchain contract, and defer persisted-data proof to Phase 12.

<phase_requirements>
## Phase Requirements

| ID | Description | Research support |
|----|-------------|------------------|
| UPG-01 | Move all nine direct Chroma dependencies and the lockfile to 1.5.9 | The pin-only update does not resolve; `fastrace` must be reconciled with Chroma's exact version |
| UPG-02 | Adapt the `Frontend::delete` region parameter without changing FFI behavior | This is the only direct shim compile break found by the probe |
| UPG-03 | Preserve public binding compatibility and explicitly disposition additive upstream APIs | No exported shim signature needs to change for the migration |
| UPG-04 | Make Rust and protobuf guidance match the resolved graph | The documented Rust 1.70+ floor is already below the current lockfile's effective floor |
</phase_requirements>

## Evidence Baseline

The assessment compares the official Chroma tags directly:

| Item | Chroma 1.5.5 | Chroma 1.5.9 |
|------|--------------|--------------|
| Tag commit | `eca66b7a58b5b2f478a5b8bae0ce7ce5f7a53f9a` | `11f3c743774db23f04a134eda1651644c25a0b35` |
| Local pin location | `shim/Cargo.toml:19-27` | Target state for the same nine entries |

Comparison links:

- [Official 1.5.5 to 1.5.9 comparison](https://github.com/chroma-core/chroma/compare/1.5.5...1.5.9)
- [Chroma 1.5.6 release](https://github.com/chroma-core/chroma/releases/tag/1.5.6)
- [Chroma 1.5.7 release](https://github.com/chroma-core/chroma/releases/tag/1.5.7)
- [Chroma 1.5.8 release](https://github.com/chroma-core/chroma/releases/tag/1.5.8)
- [Chroma 1.5.9 release](https://github.com/chroma-core/chroma/releases/tag/1.5.9)

### Quantified upstream change surface

| Measure | Result |
|---------|--------|
| Commits between tags | 205 |
| All upstream files changed | 481 |
| All upstream line delta | +48,732 / -10,537 |
| Files changed under the nine directly consumed crate directories | 103 |
| Direct-crate line delta | +19,803 / -3,176 |
| Commit subjects classified as enhancements | 104 |
| Commit subjects classified as bug fixes | 35 |
| Commit subjects classified as performance | 2 |
| Commit subjects classified as maintenance | 55 |
| Commit subjects classified as release | 9 |

The subject classification is a planning aid, not an upstream semantic guarantee. The source and compile probe are the stronger evidence for local impact.

## Local Dependency Surface

The shim consumes these nine Chroma Rust crates directly, all pinned to the same tag:

1. `chroma-frontend`
2. `chroma-config`
3. `chroma-system`
4. `chroma-types`
5. `chroma-log`
6. `chroma-index`
7. `chroma-segment`
8. `chroma-sysdb`
9. `chroma-sqlite`

The resolved Chroma internal package line moves from 0.13.2 to 0.14.0. A reconciled probe lockfile changed by 222 additions and 38 removals and grew from 769 to 781 package identities. New transitive identities included `combine`, `failsafe`, `fastbloom`, `foldhash`, JNI support, `reqwest` 0.13, and platform verifier packages.

### Lockfile blocker found by the probe

A naive edit of the nine tags fails dependency resolution:

- The current lockfile selects `fastrace` 0.7.16.
- Chroma 1.5.9 pins `fastrace` exactly to 0.7.8.
- Cargo must be allowed to reconcile that package while updating the Chroma git sources.

This is a normal lockfile migration, but it must be an explicit plan task. A pin-only diff is not a buildable result.

## Source Compatibility Findings

### One direct shim compile break

The local embedded delete path currently calls:

```rust
frontend.delete(request).await
```

Chroma 1.5.9 requires:

```rust
frontend.delete(request, region).await
```

The affected local call is in `run_embedded_delete_records` at `shim/src/lib.rs:2614`. Chroma's own in-process property-test helper passes an empty string, and the temporary local probe also compiled and passed with `String::new()`. The implementation plan should still record why the selected value is correct for local embedded mode and add a delete regression test; it should not smuggle in a cloud-only regional assumption.

No other direct shim call-site break was found by `cargo check --all-targets` after resolving the lockfile.

### Additive Chroma capabilities

Chroma 1.5.9 adds capabilities that do not need to be exposed for this dependency migration:

- get a collection by UUID (`getCollectionById` in client-facing APIs)
- collection `fork_count`
- `ReadLevel::IndexAndBoundedWal`
- sparse-index algorithm selection with `Wand` and `MaxScore`
- MaxScore tenant configuration

Recommendation: record these as deferred unless discussion identifies an immediate user-facing requirement. Adding API surface would increase the regression area without being necessary to consume 1.5.9.

### Behavioral and configuration changes to retain in test coverage

- Embedding validation now rejects NaN and infinite values.
- Configuration adds optional/defaulted region, stdout tracing, log scouting, and MaxScore tenant controls.
- These additions appear default-compatible, but config parsing tests should cover existing Go and Java generated YAML.

## Persistence Assessment

Static comparison found no local SQLite migration added between 1.5.5 and 1.5.9 and no changes in the key local persistence paths examined:

- Python `local_persistent_hnsw.py`
- Rust `local_segment_manager.rs`
- Rust `sqlite_log.rs`

The HNSW git revision already resolved by this repository is also unchanged by the successful 1.5.9 probe. This lowers format-risk, but it does not prove that an existing mixed SQLite/HNSW directory survives a real upgrade. That proof belongs to Phase 12.

## Toolchain Finding

The repository currently documents Rust 1.70+ in `README.md`, while CI pins Rust 1.93.1. A direct build with Rust 1.85 fails because packages in the existing 1.5.5 lockfile already require Rust 1.88. This mismatch predates the Chroma 1.5.9 upgrade.

Phase 11 should determine the minimum supported Rust version from the final lockfile and align contributor guidance and validation with that evidence. Do not attribute the entire floor change to Chroma 1.5.9 unless the final before/after lock analysis proves it.

The README also says protobuf compiler 31.x matches Chroma 1.5.5. The phase should verify Chroma 1.5.9's protobuf requirement and update that wording even if the numeric version remains unchanged.

## Completed Feasibility Probe

A disposable repository copy was changed as follows:

1. Pin all nine direct Chroma crates to tag 1.5.9.
2. Reconcile the `fastrace` lockfile conflict.
3. Pass an empty local region to the changed delete method.

Results:

| Check | Result |
|-------|--------|
| `cargo check --all-targets` | Passed |
| `make test` | Passed against fresh data; expected skips/warnings remained |
| `make test-rust` | Passed: 42 unit tests and 2 FFI integration tests |
| `make test-java` | Passed for both JNA and Panama |

These results prove source feasibility and fresh-database behavior on one local environment. They do not prove persisted-data upgrade, reverse rollback, or the full CI operating-system matrix.

## Recommended Plan Boundaries

### Task 1: Dependency and source migration

- update the nine Chroma tags together
- reconcile only the lockfile changes required by the new graph
- adapt the delete region parameter with a focused regression test
- run locked Cargo checks and inspect duplicate/version changes

### Task 2: Compatibility and toolchain contract

- compare exported symbols and public Go/JNA/Panama API surfaces before and after
- verify existing configuration builders still parse with new defaults
- establish and document the actual Rust and protobuf requirements
- explicitly list additive Chroma APIs as included or deferred

### Task 3: Fresh-data verification

- run Rust unit/FFI, Go, JNA, and Panama tests
- run repository lint targets
- leave cross-version fixtures and destructive data operations to Phase 12

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Lockfile resolves but carries unintended duplicate major versions | Medium | Medium | Review `cargo tree --duplicates` and explain material additions |
| Delete compiles with the wrong region semantics | Low | High | Follow the local-mode upstream precedent and add a real delete test |
| Public ABI changes accidentally while adapting internals | Low | High | Diff exported symbols and run all binding smoke tests |
| Toolchain promise remains lower than the locked graph | High | Medium | Test the claimed minimum and align README/CI guidance |
| New upstream features expand phase scope | Medium | Medium | Default to documented deferral; require an explicit requirement to expose them |
| Fresh-data success is mistaken for upgrade proof | High | High | Make Phase 12 a release gate and keep its fixture criteria explicit |

## Questions for Discuss Phase

1. Confirm that additive 1.5.9 APIs remain deferred unless there is a concrete consumer.
2. Confirm the local empty-region behavior for embedded delete, or identify a real repository configuration source for it.
3. Decide whether to advertise the measured minimum Rust version or simply pin the supported CI/development version.
4. Decide whether Phase 11 must finish on every supported CI platform before Phase 12 starts, or whether Phase 12 owns the complete matrix.

## Planner Checklist

- [ ] Every one of UPG-01 through UPG-04 maps to at least one task and verification command.
- [ ] The lockfile update is reviewed as a dependency migration, not accepted as generated noise.
- [ ] The delete regression test reaches the real embedded frontend.
- [ ] ABI/API comparison is explicit for C, Go, JNA, and Panama.
- [ ] Fresh-data tests are not presented as cross-version persistence proof.
- [ ] Phase 12 remains blocked on the migrated shim and owns the persisted fixture.
