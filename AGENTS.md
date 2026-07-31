# AGENTS.md

Guidance for coding agents working in this repository.

## Project Summary

- `local-go-chroma` is a Go wrapper for running Chroma as an embedded server.
- It uses a Rust FFI shim plus `purego` (no `cgo`).
- Primary languages: Go, Rust, and Java.

## Requirements

- Go 1.21+
- Rust 1.70+
- Java 17+ (JNA) and Java 22+ (Panama)
- `golangci-lint`, ShellCheck, and yamllint for complete linting

Go 1.21+ remains the project build/runtime minimum. The workflow-lint tooling pins actionlint v1.7.11 in `.actionlint-version`; running that module through Go requires Go 1.24+ only for linting.

## Common Commands

- Build debug shim: `make build`
- Build release shim: `make build-release`
- Build Java modules: `make build-java`
- Run tests (debug): `make test`
- Run tests (release): `make test-release`
- Run Java smoke tests: `make test-java`
- Run all Go, Rust, Actions, embedded-shell, and YAML linters: `make lint`
- Run only workflow and YAML linters: `make lint-workflows`
- Windows full lint: `pwsh -File .\scripts\dev-windows.ps1 -Task lint`
- Windows workflow lint: `pwsh -File .\scripts\dev-windows.ps1 -Task lint-workflows`
- Format code: `make fmt`

Notes:
- `make test` and `make test-release` set `CHROMA_LIB_PATH` automatically.
- Prefer Make targets over ad-hoc commands for reproducibility.

## Linting Contract

- `make lint` runs `golangci-lint`, Rust clippy, actionlint, ShellCheck for embedded workflow shell, and repository-wide yamllint.
- Make and `scripts/dev-windows.ps1 -Task lint-workflows` read the actionlint version from `.actionlint-version` and invoke the same Go module with the same ShellCheck/SC2129 settings.
- The pinned actionlint v1.7.11 Go path, including `go install`, requires Go 1.24+. An official prebuilt actionlint binary can be run directly without Go, but the repository targets still use `go run`; this tooling detail does not raise the Go 1.21+ library baseline.
- Install ShellCheck and yamllint with `sudo apt install shellcheck yamllint` on Debian/Ubuntu, `brew install shellcheck yamllint` on macOS, or `winget install --id koalaman.shellcheck` plus `py -m pip install --user yamllint` on Windows.
- `.yamllint` follows `.gitignore` for repository-local exclusions. A relocated `CARGO_TARGET_DIR` is excluded only when its exact path is also Git-ignored.
- CI runs the workflow contract in a standalone Ubuntu 24.04 `workflow-lint` job via `make lint-workflows`.

## Code Map

- `chroma.go`: server lifecycle and public Go API
- `rebuild.go`: collection rebuild maintenance API and server orchestration
- `config.go`: server config and builder options (`With...`)
- `library.go`: dynamic library loading and symbol binding via `purego`
- `errors.go`: error handling types and codes
- `shim/src/lib.rs`: Rust FFI exports and runtime-backed server operations
- `chroma_test.go`: integration-style tests against real server instances
- `java/core`: shared Java runtime API surface
- `java/jna`: Java 17 JNA bindings
- `java/panama`: Java 22 Panama bindings

## Implementation Rules

- Preserve the no-`cgo` design.
- Keep Go and Rust FFI contracts in sync when changing signatures.
- Maintain resource cleanup behavior (`Stop`, `Close`, and finalizers).
- Keep public API changes backward compatible unless explicitly requested.
- Add or update tests for behavior changes.
- Prefer public API call sites in functional-option form (`WithX(...)`) over nested option structs when introducing or refactoring APIs.
- Validate functional options in the entrypoint method by looping over all provided options and returning clear errors before any side effects.

## Validation Before Handoff

- Run relevant checks for touched areas:
- `make test`
- `make lint`

If a full run is not possible, document exactly what was not executed and why.
