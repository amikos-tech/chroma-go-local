---
phase: 11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9
plan: "03"
subsystem: compatibility
tags: [rust, chroma-1.5.9, native-abi, purego, jna, panama]

requires:
  - phase: 11-02-private-delete-adaptation
    provides: Compiling Chroma 1.5.9 shim with stable private delete semantics
provides:
  - Contributor contract for Rust 1.88.0 and protoc 31.1 source builds
  - Byte-identical 47-symbol before/after native ABI evidence
  - Fresh-data Go, Rust, JNA, and Panama validation against the rebuilt shim
affects: [11-04-exact-sha-ci-evidence, phase-12-cross-version-compatibility, phase-13-additive-api-bindings]

tech-stack:
  added: []
  patterns: [role-specific toolchain documentation, artifact-derived ABI comparison, uncached fresh-data binding verification]

key-files:
  created:
    - .planning/phases/11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9/11-abi-after.exports
    - .planning/phases/11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9/11-abi-export-comparison.md
  modified:
    - README.md
    - AGENTS.md

key-decisions:
  - "Documented Rust 1.88.0 as the source-build MSRV while retaining Rust 1.93.1 as the exact CI/release compiler pin."
  - "Kept all additive Chroma 1.5.9 APIs deferred to Phase 13 and all persisted-data compatibility claims reserved for Phase 12."
  - "Required uncached supplemental Go and Java runs after repository targets reported cached or up-to-date results."

patterns-established:
  - "Toolchain documentation separates source builders, reproducible CI/release builds, and prebuilt consumers."
  - "Native ABI evidence records the actual dynamic-library inspector, source SHAs, export-list checksums, and zero-output diff."

requirements-completed: [UPG-03, UPG-04]

duration: 6min
completed: 2026-08-02
---

# Phase 11 Plan 03: Toolchain and Native Compatibility Summary

**Rust 1.88.0/protoc 31.1 source-build guidance now matches the Chroma 1.5.9 graph, with an identical 47-symbol native ABI and fresh Go, Rust, JNA, and Panama evidence.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-08-02T12:32:02Z
- **Completed:** 2026-08-02T12:37:55Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Published one consistent toolchain contract: Rust 1.88.0 and `protoc` 31.1 for source builders, exact Rust 1.93.1 for CI/releases, and neither tool for prebuilt consumers.
- Captured 47 actual `chroma_*` exports from the rebuilt Mach-O dylib; the before and after lists are byte-identical with SHA-256 `d1360de902b9d17c30c36e6785f2c808386a4c6fd3d102e6580bae2baf33c3f8`.
- Passed the repository Go, Rust, Java, and lint targets, plus uncached fresh-data runs for Go, JNA, and Panama against the rebuilt Chroma 1.5.9 shim.
- Preserved the compatibility boundary: Phase 12 owns persisted 1.5.5-data upgrade evidence, while Phase 13 owns all five deferred additive API groups.

## Task Commits

Each task was committed atomically:

1. **Task 1: Align contributor and project-agent toolchain guidance without changing workflows** - `73e2569` (docs)
2. **Task 2: Compare built exports and run the rebuilt-shim fresh-data suites** - `c11cba3` (test)

## Files Created/Modified

- `README.md` - Defines source-builder, CI/release, prebuilt-consumer, fresh-data, and Phase 12/13 boundaries.
- `AGENTS.md` - Replaces the stale Rust floor with the measured Chroma 1.5.9 source-build contract while retaining all project instructions.
- `.planning/phases/11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9/11-abi-after.exports` - Sorted 47-symbol export set observed in the rebuilt dylib.
- `.planning/phases/11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9/11-abi-export-comparison.md` - Reproducible before/after comparison with commands, SHAs, hashes, timestamp, and `RESULT: identical`.

## Decisions Made

- Kept `.github/workflows/ci.yml` and `.github/workflows/release.yml` unchanged because both already use exact Rust 1.93.1 and `protoc` 31.1 pins.
- Named every Phase 13 deferral explicitly: UUID collection lookup, `fork_count`, `ReadLevel::IndexAndBoundedWal`, sparse-index selection, and MaxScore controls.
- Forced supplemental uncached runs instead of treating Go cache hits or Gradle `UP-TO-DATE` output as fresh-data execution evidence.

## Documentation Review Checklist

- [x] Source builders are told to use Rust 1.88.0 or newer and `protoc` 31.1 for the committed Chroma 1.5.9 graph.
- [x] CI and release builds are documented as retaining the exact Rust 1.93.1 reproducibility pin.
- [x] Released/prebuilt native-library consumers are told they need neither Rust nor `protoc`.
- [x] Phase 11 is explicitly fresh-data-only and makes no persisted Chroma 1.5.5 compatibility claim.
- [x] Phase 12 owns persisted-data upgrade, mutation/reopen, and maintenance compatibility.
- [x] Phase 13 owns all five named additive API groups; no C, Go, JNA, or Panama typed API was added.
- [x] Default-compatible raw YAML and schema parsing remain within the existing API boundary.

## Verification

- `make build` passed and produced `shim/target/debug/libchroma_shim.dylib`.
- `/usr/bin/nm -gjU` inspected the actual dylib; normalization produced 47 sorted `chroma_*` names.
- `diff -u 11-abi-before.exports 11-abi-after.exports` exited 0 with no output; both files have the same SHA-256.
- `make test` passed against the rebuilt dylib; `GOFLAGS='-count=1' make test` then passed as an uncached fresh-data run.
- `make test-rust` passed against the locked Chroma 1.5.9 graph.
- `make test-java` passed and invoked both `:jna:test` and `:panama:test` with `CHROMA_LIB_PATH` set to the rebuilt dylib.
- `gradle --no-daemon --rerun-tasks :jna:test :panama:test` passed with all eight tasks executed, proving uncached JNA and Panama coverage against that dylib.
- `make lint-workflows` passed during Task 1.
- Lint prerequisites were available exactly at `/opt/homebrew/bin/shellcheck` (ShellCheck 0.11.0) and `/opt/homebrew/bin/yamllint` (yamllint 1.38.0).
- `make lint` passed: golangci-lint reported zero issues, Rust clippy completed with warnings denied, and actionlint/ShellCheck/yamllint completed successfully.
- `git diff --check` passed.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The first repository Go and Java target runs reused local caches. Both required targets passed, then uncached supplemental runs were executed to produce fresh-data evidence rather than relying on cached results.
- Expected test diagnostics about unavailable local `scout_logs`, intentional address-conflict cases, JNA native access, and reflective Gson field access did not fail their suites.

## Known Stubs

None found in the files created or modified by this plan.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 11-04 can use the exact migrated SHA to obtain the independent Linux, macOS, and Windows normal-PR CI evidence.
- The local ABI gate is closed with actual-library proof; there is no pending inspector fallback.
- Phase 12 remains the release gate for persisted-data upgrade, mutation/reopen, maintenance, and cross-version binding compatibility.

## Self-Check: PASSED

- All four task files exist.
- Task commits `73e2569` and `c11cba3` exist in repository history.
- The export lists are byte-identical and the comparison record says `RESULT: identical`.
- Requirements `UPG-03` and `UPG-04` are copied verbatim from the plan frontmatter.

---
*Phase: 11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9*
*Completed: 2026-08-02*
