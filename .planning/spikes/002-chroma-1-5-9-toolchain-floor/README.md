---
spike: 002
name: chroma-1-5-9-toolchain-floor
type: standard
validates: "Given the final spike lockfile, when toolchain requirements are validated, then the Rust MSRV, CI pin, and protoc requirement can be stated accurately."
verdict: VALIDATED
related: [001-chroma-1-5-9-lock-resolution]
tags: [rust, msrv, protoc, phase-11]
---

# Spike 002: Chroma 1.5.9 Toolchain Floor

## What This Validates

Given the targeted Chroma 1.5.9 lockfile from Spike 001, when the shim builds with a locked graph under candidate Rust and protobuf versions, then the project can state an evidence-backed MSRV and reproducible CI toolchain.

## Research

| Approach | Pros | Cons | Status |
|---|---|---|---|
| Infer from Chroma's `rust-toolchain.toml` (1.92.0) | Authoritative for how upstream develops Chroma | Does not describe the lowest compiler that builds this wrapper's locked graph | Supporting evidence only |
| State the existing CI pin (1.93.1) as the minimum | Reproducible for CI and releases | Overstates the source-build floor | Rejected as the MSRV |
| Measure the wrapper's final `Cargo.lock` | Directly tests the project promise | Must be repeated whenever the locked graph materially changes | Chosen |

Chroma 1.5.9's `rust-toolchain.toml` selects Rust 1.92.0, and its Dockerfile pins `PROTOC_VERSION=31.1`. This repository's CI and release workflows pin Rust 1.93.1 and use `protoc` 31.1. Cargo documents that package `rust-version` fields represent compatibility requirements and that `--locked` rejects a changed dependency graph. See [Rust version support](https://doc.rust-lang.org/cargo/reference/rust-version.html) and [cargo build](https://doc.rust-lang.org/cargo/commands/cargo-build.html).

## How to Run

Starting with the targeted 1.5.9 lockfile and the local delete-region adaptation from Spike 001:

```sh
# Fails: exact locked packages require Rust 1.88.
cargo +1.85.0 check --manifest-path shim/Cargo.toml --all-targets --locked

# Passes: measured MSRV.
cargo +1.88.0 check --manifest-path shim/Cargo.toml --all-targets --locked

# Passes with the upstream/CI protobuf generator release.
PATH="/path/to/protoc-31.1/bin:$PATH" \
  cargo +1.88.0 check --manifest-path shim/Cargo.toml --all-targets --locked
```

## What to Expect

- Rust 1.85.0 fails before compilation because the locked AWS, `darling`, `home`, and `time` packages require Rust 1.88.0.
- Rust 1.88.0 passes `cargo check --all-targets --locked`.
- Rust 1.89.0 and 1.92.0 also pass, but they are not the MSRV.
- `libprotoc 31.1`, downloaded from the official release and verified against the CI SHA-256, passes the same Rust 1.88.0 locked check.

## Investigation Trail

1. The existing project documentation claimed Rust 1.70+, but the targeted 1.5.9 lockfile rejected Rust 1.85.0 due to package-declared Rust 1.88 requirements.
2. Rust 1.88.0 was installed and compiled every shim target successfully with `--locked`.
3. Chroma upstream selects Rust 1.92.0 for its own development, while this repository's CI/release jobs pin Rust 1.93.1. Those are reproducibility pins, not the wrapper's measured minimum.
4. The local machine's `libprotoc 35.1` passed, but that was insufficient evidence for the intended contributor setup. The official universal macOS `protoc` 31.1 archive was SHA-256-verified and passed a clean Rust 1.88.0 all-targets build.

## Results

**VALIDATED.** The Phase 11 hybrid contract should be:

- **MSRV:** Rust 1.88.0 for source builds of the committed 1.5.9 lockfile.
- **CI/release compiler:** Rust 1.93.1, retained as the reproducible build pin.
- **Protobuf generator:** `protoc` 31.1 for source builds; prebuilt-library consumers need neither Rust nor `protoc`.
- **Drift control:** rerun the Rust 1.88.0 `--locked` check whenever the committed lockfile changes materially.
