# Spike Manifest

## Idea

Establish reproducible evidence for the Chroma 1.5.9 Rust dependency migration before Phase 11 planning: resolve the nine direct Chroma dependencies, inspect the resulting lockfile, and determine accurate source-build toolchain guidance.

## Requirements

- The real Phase 11 migration must preserve the existing C FFI and public Go, JNA, and Panama APIs.
- Dependency evidence must come from a reproducible locked Cargo graph, not a pin-only diff.
- Rust documentation must distinguish a measured MSRV from the exact CI/release pin.
- Persisted-data upgrade evidence remains Phase 12 scope.

## Spikes

| # | Name | Type | Validates | Verdict | Tags |
|---|------|------|-----------|---------|------|
| 001 | chroma-1-5-9-lock-resolution | standard | Given the nine Chroma 1.5.9 pins, when Cargo resolves the graph, then the committed lockfile builds with `--locked` and the `fastrace` reconciliation is understood | VALIDATED | cargo, chroma, lockfile, phase-11 |
| 002 | chroma-1-5-9-toolchain-floor | standard | Given the final spike lockfile, when toolchain requirements are validated, then the Rust MSRV, CI pin, and protoc requirement can be stated accurately | PENDING | rust, msrv, protoc, phase-11 |
