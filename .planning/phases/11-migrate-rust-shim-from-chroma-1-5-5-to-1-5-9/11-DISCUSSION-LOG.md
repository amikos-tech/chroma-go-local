# Phase 11: Migrate Rust shim from Chroma 1.5.5 to 1.5.9 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in `11-CONTEXT.md` — this log preserves the alternatives considered.

**Date:** 2026-08-02
**Phase:** 11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9
**Areas discussed:** Additive Chroma APIs, embedded delete region, Rust and protobuf support guidance, Phase 11 verification boundary

---

## Additive Chroma 1.5.9 APIs

| Option | Description | Selected |
|---|---|---|
| Defer new APIs | Preserve the current C/Go/JNA/Panama surface and record upstream additions for later | ✓ |
| Expose them now | Add full opt-in APIs across every binding | |

**User's choice:** Defer the APIs and add a later phase for API expansion.
**Notes:** Phase 13 was added for the deferred cross-language API work. Existing raw YAML/schema behavior remains default-compatible without becoming a new typed API.

---

## Embedded delete region

| Option | Description | Selected |
|---|---|---|
| Pass an empty region | Match Chroma's local in-process precedent and preserve public behavior | ✓ |
| Read region from configuration | Add private configuration plumbing for telemetry labels | |

**User's choice:** Pass an empty region unless it breaks API calls.
**Notes:** The researched region is telemetry-only; `String::new()` is private to the shim and does not change any public call. A real embedded delete regression will protect the behavior.

---

## Rust and protobuf support guidance

| Option | Description | Selected |
|---|---|---|
| Measured minimum only | Publish only the tested oldest Rust compiler | |
| CI pin only | Publish only the reproducible release compiler | |
| Hybrid policy | Publish the measured MSRV and retain an exact CI/release pin | ✓ |

**User's choice:** Hybrid policy.
**Notes:** Spikes established Rust 1.88.0 as the MSRV, Rust 1.93.1 as the CI/release pin, and `protoc` 31.1 as the source-build generator requirement.

---

## Phase 11 verification boundary

| Option | Description | Selected |
|---|---|---|
| Full OS matrix as a special Phase 11 exit gate | Add a phase-specific cross-platform control | |
| Fresh-data validation plus normal green PR CI | Require existing Linux/macOS/Windows CI, with cross-version proof deferred to Phase 12 | ✓ |

**User's choice:** Fresh-data validation plus normal green CI.
**Notes:** The normal PR CI already runs the OS matrix. Its fresh-data results must not be described as persisted-data upgrade proof.

---

## Spike evidence folded in

- Spike 001 validated targeted Chroma resolution and the required `fastrace` 0.7.8 reconciliation; an unconstrained `cargo update` was rejected as excessive dependency churn.
- Spike 002 measured the Rust 1.88.0 MSRV and verified the upstream/CI `protoc` 31.1 source-build requirement.

## the agent's Discretion

- Exact implementation of MSRV enforcement and the embedded delete regression, within the locked decisions above.

## Deferred Ideas

- Phase 13 — expose the deferred Chroma 1.5.9 APIs with backward-compatible Go, JNA, and Panama bindings.
