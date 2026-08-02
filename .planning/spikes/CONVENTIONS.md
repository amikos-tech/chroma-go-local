# Spike Conventions

Patterns established while derisking the Rust shim migration.

## Stack

- Use the repository's Cargo shim directly; do not introduce a separate framework for build-feasibility spikes.
- Run experiments in a disposable `git archive` copy so project source and lockfiles remain unchanged until the real phase is approved.

## Structure

- Keep one README per binary build question under `.planning/spikes/NNN-name/`.
- Record exact compiler, generator, Cargo, and `--locked` commands in the spike README.

## Patterns

- Treat a lockfile migration as a targeted dependency operation. Avoid a bare `cargo update`; constrain the package update and inspect the resulting diff.
- Establish an MSRV by showing the prior candidate fails and the claimed version passes the same locked all-targets command.

## Tools & Libraries

- Cargo's `--locked` mode verifies reproducible resolution.
- Use the CI-pinned protobuf release and verify its SHA-256 before treating it as source-build evidence.
