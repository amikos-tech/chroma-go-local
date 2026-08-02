# Phase 11: Migrate Rust shim from Chroma 1.5.5 to 1.5.9 - Context

**Gathered:** 2026-08-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Upgrade the Rust shim's nine direct Chroma dependencies from 1.5.5 to 1.5.9, make the one required embedded-delete source adaptation, and align the dependency/toolchain contract while preserving every existing C FFI symbol and public Go, JNA, and Panama API. Validate fresh-data behavior through the normal cross-platform PR CI; persisted-data upgrade evidence belongs to Phase 12.

</domain>

<decisions>
## Implementation Decisions

### Additive Chroma 1.5.9 APIs
- **D-01:** Defer all explicit additive upstream APIs: collection lookup by UUID, `fork_count`, `ReadLevel::IndexAndBoundedWal`, sparse-index selection, and MaxScore controls.
- **D-02:** Keep default-compatible raw YAML/schema parsing, but add no new C, Go, JNA, or Panama typed API surface in this phase.

### Embedded delete region
- **D-03:** Pass `String::new()` for the new `Frontend::delete(request, region)` argument in local embedded mode. This is private shim state and must not change C, Go, JNA, or Panama calls.
- **D-04:** Add a real embedded delete regression that verifies deleted and remaining records; the region itself is telemetry-only and not externally observable.

### Dependency and toolchain contract
- **D-05:** Update all nine Chroma tags together, then explicitly run `cargo update -p fastrace --precise 0.7.8`. Treat the generated lockfile as a dependency migration and inspect it; never use an unconstrained `cargo update`.
- **D-06:** Adopt the hybrid toolchain policy established by the validated spikes: Rust 1.88.0 is the source-build MSRV, Rust 1.93.1 remains the exact CI/release pin, and `protoc` 31.1 is the source-build generator requirement. Prebuilt-library consumers need neither Rust nor `protoc`.
- **D-07:** Revalidate `cargo +1.88.0 check --all-targets --locked` whenever the committed lockfile changes materially.

### Verification boundary
- **D-08:** Phase 11 requires fresh-data validation and a green normal PR CI matrix. That matrix already exercises Linux, macOS, and Windows, but it must not be described as proof that existing 1.5.5 data upgrades safely.
- **D-09:** Phase 12 remains the release gate for cross-version persisted-data, mutation/reopen, maintenance, and binding compatibility evidence.

### the agent's Discretion
- Exact Cargo command sequencing and how lockfile/duplicate-package review is presented in plan tasks.
- Whether the measured MSRV is encoded in `shim/Cargo.toml`, a dedicated CI check, or both, provided the public documentation and locked validation remain aligned.
- Exact test data and assertions for the embedded delete regression.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` — Phase 11 goal, success criteria, Phase 12 boundary, and the deferred Phase 13 API-expansion phase.
- `.planning/REQUIREMENTS.md` — UPG-01 through UPG-04 and the Phase 12 COMPAT requirements.
- `.planning/phases/11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9/11-RESEARCH.md` — upstream source analysis, feasibility baseline, API inventory, and risk register.

### Validated spike evidence
- `.planning/spikes/001-chroma-1-5-9-lock-resolution/README.md` — verified `fastrace` reconciliation, targeted lockfile procedure, lockfile delta, and the bare-`cargo update` landmine.
- `.planning/spikes/002-chroma-1-5-9-toolchain-floor/README.md` — measured Rust 1.88.0 MSRV, Rust 1.93.1 CI/release pin, and verified `protoc` 31.1 source-build evidence.
- `.planning/spikes/CONVENTIONS.md` — reusable isolated-spike and locked-validation conventions.

### Implementation and contributor contract
- `shim/Cargo.toml` — nine direct Chroma git pins and shim dependency declarations.
- `shim/Cargo.lock` — committed resolution to migrate and validate with `--locked`.
- `shim/src/lib.rs` — `run_embedded_delete_records`, the only known direct source break; existing exported C symbol implementations.
- `README.md` — contributor Rust/protobuf guidance that must be corrected.
- `.github/workflows/ci.yml` — existing three-OS fresh-data PR matrix and Rust/protobuf pins.
- `.github/workflows/release.yml` — release Rust/protobuf pins that must remain consistent with documentation.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `shim/Cargo.toml` and `shim/Cargo.lock` — a committed, reproducible Cargo graph; the spike provides the safe update procedure instead of a new dependency-management mechanism.
- `shim/src/lib.rs` — the embedded FFI delete path already centralizes the required upstream signature adaptation in one local call site.
- Existing Go, JNA, and Panama bindings — public contracts already call stable `chroma_*` exports and require no expansion for this migration.

### Established Patterns
- The project is intentionally no-`cgo`: Go uses `purego`, and Java uses JNA/Panama against the same Rust C ABI.
- All language bindings must remain in sync when an exported FFI contract changes; this migration deliberately makes no such change.
- CI runs fresh-data Go, Rust, JNA, and Panama coverage on Linux, macOS, and Windows.

### Integration Points
- `run_embedded_delete_records` must call the 1.5.9 `Frontend::delete(request, region)` signature with the locked local-mode empty region.
- The Cargo manifest and lockfile must move together, with `fastrace` resolved to 0.7.8 and duplicate-package output reviewed.
- README and workflow guidance must state the validated source-build and reproducible-CI toolchain values consistently.

</code_context>

<specifics>
## Specific Ideas

- The targeted 1.5.9 lockfile probe changed 222 lines by addition and 38 by removal; an unconstrained `cargo update` instead refreshed 276 packages and introduced unrelated Rust 1.94.1 requirements.
- Rust 1.85.0 was rejected by the targeted lockfile, while Rust 1.88.0 passed `cargo check --all-targets --locked`.
- The official `protoc` 31.1 release was SHA-256 verified and passed the same clean Rust 1.88.0 all-targets build.

</specifics>

<deferred>
## Deferred Ideas

- **Phase 13: Expose deferred Chroma 1.5.9 APIs with backward-compatible Go, JNA, and Panama bindings.** It owns any future cross-language wrapper API for UUID collection lookup, `fork_count`, bounded-WAL reads, sparse-index algorithms, and MaxScore controls.

</deferred>

---

*Phase: 11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9*
*Context gathered: 2026-08-02*
