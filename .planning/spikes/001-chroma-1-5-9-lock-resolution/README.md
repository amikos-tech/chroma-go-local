---
spike: 001
name: chroma-1-5-9-lock-resolution
type: standard
validates: "Given the nine Chroma 1.5.9 pins, when Cargo resolves the graph, then the committed lockfile builds with --locked and the fastrace reconciliation is understood."
verdict: VALIDATED
related: [002-chroma-1-5-9-toolchain-floor]
tags: [cargo, chroma, lockfile, phase-11]
---

# Spike 001: Chroma 1.5.9 Lock Resolution

## What This Validates

Given the shim's nine direct Chroma dependencies at tag 1.5.9, when Cargo resolves the graph conservatively, then the lockfile contains the 1.5.9 graph and `fastrace` 0.7.8 without refreshing unrelated registry dependencies.

## Research

Cargo documents that a package-specific `cargo update` updates that package and only the transitive dependencies it must change; `--locked` then rejects any build that would change `Cargo.lock`. See [cargo update](https://doc.rust-lang.org/cargo/commands/cargo-update.html) and [cargo build](https://doc.rust-lang.org/cargo/commands/cargo-build.html).

| Approach | Pros | Cons | Status |
|---|---|---|---|
| Manifest tags only | Shows the actual resolver conflict | Cannot build: the old locked `fastrace` 0.7.16 conflicts with Chroma's exact 0.7.8 requirement | Rejected as incomplete |
| Bare `cargo update` | Resolves successfully | Refreshed 276 packages in this environment, including unrelated packages requiring Rust 1.94.1 | Rejected as migration procedure |
| `cargo update -p fastrace --precise 0.7.8` | Reconciles the necessary package while keeping 212 existing packages unchanged | Must be called out as an intentional migration step | Chosen |

## How to Run

Run the migration in a disposable copy, change all nine Chroma tags to `1.5.9`, then run:

```sh
cargo update --manifest-path shim/Cargo.toml -p fastrace --precise 0.7.8
cargo tree --manifest-path shim/Cargo.toml --duplicates
cargo +1.88.0 check --manifest-path shim/Cargo.toml --all-targets --locked
```

The compile command also requires the known local `Frontend::delete(request, String::new())` shim adaptation.

## What to Expect

- `fastrace` changes from 0.7.16 to 0.7.8.
- The Chroma git source changes from tag 1.5.5 (`eca66b7`) to 1.5.9 (`11f3c743`).
- The lockfile diff is 222 additions and 38 removals in this probe.
- `cargo check --all-targets --locked` passes once the delete call is adapted.

## Investigation Trail

1. A manifest-only `cargo check` failed because the old locked `fastrace` 0.7.16 could not satisfy Chroma 1.5.9's exact 0.7.8 dependency.
2. A bare `cargo update` resolved the graph but updated 276 packages, including unrelated AWS packages that require Rust 1.94.1. It is not a valid Phase 11 procedure.
3. Running `cargo update -p fastrace --precise 0.7.8` moved the Chroma graph to tag 1.5.9, added only the needed transitive identities, and retained 212 locked packages unchanged.
4. After the planned local delete-region adaptation, the targeted graph passed `cargo +1.88.0 check --all-targets --locked`.

## Results

**VALIDATED.** Phase 11 must treat the lockfile as a targeted dependency migration: update all nine direct Chroma tags together, explicitly reconcile `fastrace` to 0.7.8, inspect duplicate packages, and validate the resulting graph with `--locked`. Do not use a bare `cargo update`.
