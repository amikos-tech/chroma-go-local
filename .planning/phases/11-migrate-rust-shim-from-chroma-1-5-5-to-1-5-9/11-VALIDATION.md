---
phase: 11
slug: migrate-rust-shim-from-chroma-1-5-5-to-1-5-9
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-08-02
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Cargo test/check, Go `testing`, and Gradle/JUnit (JNA + Panama) |
| **Config file** | `shim/Cargo.toml`, `go.mod`, and `java/build.gradle.kts` |
| **Quick run command** | After Plan 11-02 Task 2 adapts the private delete region: `cargo +1.88.0 check --manifest-path shim/Cargo.toml --all-targets --locked` |
| **Full suite command** | `make test && make test-rust && make test-java && make lint` |
| **Estimated runtime** | ~10 minutes locally; CI evidence may take up to 30 minutes |

---

## Sampling Rate

- **After every task commit:** Run the task's fastest mapped command.
- **After every plan wave:** Run `make test`, `make test-rust`, and `make test-java`; run `make lint` when its local tools are available, recording exact missing-tool output otherwise.
- **Before `$gsd-verify-work`:** The complete fresh-data local suite and exact-SHA normal-PR CI evidence must be green.
- **Max feedback latency:** 10 minutes for local checks; 30 minutes for CI metadata polling.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 11-01-01 | 01 | 1 | UPG-03 | T-11-02 | A real pre-migration native `chroma_*` export baseline, or a precise inspector-unavailability record that keeps the ABI comparison pending, exists before dependency edits | ABI/artifact | `test -s 11-abi-before-inspection.md` plus either a non-empty export list or `UNAVAILABLE` record | ❌ Wave 0 | ⬜ pending |
| 11-01-02 | 01 | 1 | UPG-01 | T-11-01 | All nine named Chroma packages resolve from the official `1.5.9` git source in locked Cargo metadata; this Wave 1 task does not require a successful compile before the known adaptation | dependency integration | `cargo metadata --manifest-path shim/Cargo.toml --locked --format-version=1` plus the plan's JSON predicate | ✅ | ⬜ pending |
| 11-02-01 | 02 | 2 | UPG-02 | T-11-03 | A public embedded delete removes only the requested record through Go → C → Rust | integration | `make build` then the focused Go delete/survivor test | ❌ Wave 0 | ⬜ pending |
| 11-02-02 | 02 | 2 | UPG-02, UPG-04 | T-11-04 | After the known private delete-region adaptation, the committed graph compiles without lockfile mutation using the documented MSRV; any unexpected upstream change stays private and compatibility-preserving, or escalates before changing ABI/public APIs | compile + review | `cargo +1.88.0 check --manifest-path shim/Cargo.toml --all-targets --locked` | ✅ | ⬜ pending |
| 11-03-01 | 03 | 3 | UPG-03 | T-11-02 | The rebuilt native library has exactly the pre-migration `chroma_*` export set | ABI/artifact | platform export-list capture followed by `diff -u` | ❌ Wave 0 | ⬜ pending |
| 11-03-02 | 03 | 3 | UPG-03 | T-11-02 | Go, JNA, and Panama load and test the rebuilt shim without public API changes | integration/smoke | `make test && make test-java` | ✅ | ⬜ pending |
| 11-03-03 | 03 | 3 | UPG-04 | T-11-01 | Source-builder documentation and CI/release pins state Rust 1.88.0, Rust 1.93.1, and protoc 31.1 accurately | workflow lint + review | `make lint-workflows` and semantic document review | ✅ | ⬜ pending |
| 11-04-01 | 04 | 4 | UPG-03 | T-11-03 | Exact migrated SHA has successful normal-PR CI evidence with every returned job successful, or an equivalent maintainer-supplied record | CI integration | `gh run view "$RUN_ID" --json status,conclusion,headSha,jobs` when authenticated; otherwise manual evidence template | ✅ CI; manual fallback | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/runtime/embedded_integration_edge_test.go` — add a focused public embedded delete test that proves both deletion and survivor retrieval.
- [ ] Task-local temporary export baseline and comparison procedure — capture real `chroma_*` exports before migration and diff after rebuilding; do not add production ABI infrastructure.
- [ ] Plan task JSON predicate — prove all nine expected Chroma packages resolve from tag `1.5.9` through locked Cargo metadata.
- [ ] `11-04-SUMMARY.md` evidence template — support both authenticated GitHub CLI metadata and authorized manual CI evidence.
- [ ] Source-adaptation task — include the Rule-1/private-fix versus Rule-4/escalation boundary for unexpected Chroma 1.5.9 compilation changes.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Exact-SHA normal-PR CI evidence when `gh` is unavailable, unauthenticated, or offline | UPG-03 | The executor cannot fetch remote CI metadata in that environment | An authorized maintainer records PR URL, head SHA, CI run URL/ID, UTC retrieval time, and every returned job name/status/conclusion in `11-04-SUMMARY.md`. |
| Windows native export inspection tool selection | UPG-03 | Available inspector (`llvm-nm` or `dumpbin`) depends on the runner image | Use an available PE export inspector, save the sorted `chroma_*` list, and compare it with the pre-migration baseline; record any unavailable tool precisely. |
| Contributor guidance semantic accuracy | UPG-04 | Text linting cannot prove that source-builder and prebuilt-consumer distinctions are understandable | Verify that docs say Rust 1.88.0 and protoc 31.1 apply to source builders, Rust 1.93.1 remains the exact CI/release pin, and prebuilt consumers require neither tool. |

---

## Validation Sign-Off

- [x] All planned tasks have automated verification or Wave 0 dependencies.
- [x] Sampling continuity: no three consecutive tasks lack automated feedback.
- [x] Wave 0 covers every missing verification artifact.
- [x] No watch-mode flags are required; CI polling has a 30-minute deadline.
- [x] Local feedback latency is under 10 minutes.
- [x] `nyquist_compliant: true` set in frontmatter.

**Approval:** pending
