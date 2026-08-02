---
phase: 11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9
plan: "01"
subsystem: dependencies
tags: [rust, cargo, chroma-1.5.9, fastrace, native-abi]

requires:
  - phase: 10-server-maintenance
    provides: Stable Rust C ABI shared by Go, JNA, and Panama maintenance bindings
provides:
  - Nine direct Chroma dependencies pinned to official tag 1.5.9
  - Reproducible constrained lockfile with fastrace 0.7.8
  - Rust 1.88 source-build MSRV declaration
  - Artifact-derived pre-migration baseline of 47 native chroma_* exports
affects: [11-02-private-source-adaptation, 11-03-toolchain-validation, 11-04-abi-ci-evidence, phase-12-compatibility]

tech-stack:
  added: [Chroma 1.5.9 dependency graph, fastrace 0.7.8]
  patterns: [targeted Cargo resolution, locked metadata verification, artifact-derived ABI baselining]

key-files:
  created:
    - .planning/phases/11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9/11-abi-before.exports
    - .planning/phases/11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9/11-abi-before-inspection.md
  modified:
    - shim/Cargo.toml
    - shim/Cargo.lock

key-decisions:
  - "Resolved the migration only through the package-specific fastrace 0.7.8 update, preserving 212 unrelated locked packages."
  - "Declared Rust 1.88 as the source-build MSRV while leaving the successful locked all-targets compile gate to Plan 11-02 after its known private delete adaptation."

patterns-established:
  - "Dependency migrations use Cargo metadata predicates over exact package names and sources, not formatting-sensitive lockfile text counts."
  - "Native ABI baselines come from the built dynamic library and retain the exact source SHA, inspector command, and list checksum."

requirements-completed: [UPG-01, UPG-04]

duration: 4min
completed: 2026-08-02
---

# Phase 11 Plan 01: Chroma 1.5.9 Dependency Graph Summary

**All nine direct Chroma packages now resolve from official tag 1.5.9 with fastrace 0.7.8, backed by a checksumed 47-symbol pre-migration native ABI baseline.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-08-02T11:42:09Z
- **Completed:** 2026-08-02T11:46:14Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Built the pre-migration Chroma 1.5.5 dylib and captured 47 actual `chroma_*` exports with the exact commit, command, platform, and SHA-256 provenance.
- Migrated all nine direct Chroma git pins together to tag 1.5.9 and generated the lockfile with only the constrained `fastrace` 0.7.8 update procedure.
- Proved the locked graph semantically: every expected package resolves from tag 1.5.9, no expected package resolves from 1.5.5, and `fastrace` is exactly 0.7.8.
- Declared Rust 1.88 as the source-build MSRV without prematurely claiming the compile gate that requires Plan 11-02's private delete-signature adaptation.

## Task Commits

Each task was committed atomically:

1. **Task 1: Capture the actual pre-migration chroma_* native export set** - `0adc8e0` (chore)
2. **Task 2: Pin the official Chroma 1.5.9 graph and verify it structurally** - `9e37829` (chore)

## Files Created/Modified

- `.planning/phases/11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9/11-abi-before.exports` - Sorted list of 47 exports observed in the pre-migration dylib.
- `.planning/phases/11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9/11-abi-before-inspection.md` - Reproducible artifact inspection record and checksum.
- `shim/Cargo.toml` - Rust 1.88 declaration and nine synchronized Chroma 1.5.9 pins.
- `shim/Cargo.lock` - Cargo-generated Chroma 1.5.9 graph constrained to `fastrace` 0.7.8.

## Decisions Made

- Used only `cargo update --manifest-path shim/Cargo.toml -p fastrace --precise 0.7.8`; a bare resolver refresh remained prohibited.
- Kept compilation explicitly deferred because the new graph is expected to expose the already-planned private `Frontend::delete` signature break. Plan 11-02 owns the first successful Rust 1.88 locked all-targets check.
- Retained the full duplicate-package graph as reviewed resolver output; the lockfile diff matches the validated targeted migration shape with 212 packages unchanged.

## Verification

- `make build` passed before dependency edits and produced `shim/target/debug/libchroma_shim.dylib`.
- `/usr/bin/nm -gjU` produced 47 normalized, sorted `chroma_*` exports; the committed list SHA-256 is `d1360de902b9d17c30c36e6785f2c808386a4c6fd3d102e6580bae2baf33c3f8`.
- The task-level Cargo metadata predicate returned `false` before the migration and `true` against the final locked graph.
- Manifest checks found exactly nine direct tag-1.5.9 pins, no tag-1.5.5 references, `rust-version = "1.88"`, and `fastrace` 0.7.8.
- `cargo tree --manifest-path shim/Cargo.toml --duplicates` completed and the generated diff was reviewed for unrelated churn.
- `git diff --check HEAD~2..HEAD` passed.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

Two local verification wrappers initially used zsh special parameters (`status` and `path`). Both commands were rerun with neutral variable names and passed with the intended evidence. Neither attempt affected repository files.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 11-02 can now adapt the private `Frontend::delete(request, region)` call against the committed 1.5.9 graph.
- The Rust 1.88 `--all-targets --locked` compile gate remains intentionally pending until that adaptation is present.
- The pre-migration artifact baseline is ready for the migrated before/after ABI comparison in Plan 11-04.

## Self-Check: PASSED

- All four created or modified task files exist.
- Task commits `0adc8e0` and `9e37829` exist in repository history.
- The summary records both plan requirement IDs verbatim.

---
*Phase: 11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9*
*Completed: 2026-08-02*
