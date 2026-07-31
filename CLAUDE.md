# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A local Chroma runtime package with:
- Go wrapper API (purego, no cgo)
- Rust FFI shim
- Java scaffold bindings (`core`, `jna`, `panama`)

## Requirements

- Go 1.21+
- Rust 1.70+
- Java 17+ (JNA path), Java 22+ (Panama path)
- Gradle 9+
- golangci-lint, ShellCheck 0.9 or newer, and yamllint (complete linting)

Go 1.21+ remains the project build/runtime minimum. Workflow linting runs the actionlint version pinned in `.actionlint-version`. Go 1.21+ with automatic toolchain switching is the normal path; Go 1.24+ must be installed locally only when switching is unavailable or disabled, including `GOTOOLCHAIN=local` or an older pinned toolchain selected through `GOTOOLCHAIN`.

## Build Commands

```bash
make build          # Build Rust shim (debug)
make build-release  # Build Rust shim (release)
make test           # Build debug + run Go tests
make test-release   # Build release + run Go tests
make build-java     # Build Java modules (no tests)
make test-java      # Run Java smoke tests (JNA + Panama)
make test-all       # Go + Rust + Java smoke tests (Java skipped only if Gradle missing)
make lint           # Run Go, Rust, Actions, embedded-shell, and YAML linters
make lint-workflows # Run Actions, embedded-shell, and repository-wide YAML lint
make fmt            # Format all code (Go + Rust)
make clean          # Clean build artifacts
```

## Testing

Go tests require the Rust shim and are wired by Makefile:
- `make test` builds debug shim and runs Go tests
- `make test-release` builds release shim and runs Go tests
- `CHROMA_LIB_PATH` is auto-set by Makefile

Java smoke tests are available via:
- `make test-java` (runs `:jna:test` and `:panama:test`)
- `make test-all` fails on Java test failures when Gradle is present

## Architecture

```
Go Package (root)                   Internal Implementation
├── chroma.go     ─── facade ───►   internal/runtime/
├── config.go         (type         ├── chroma.go      (server lifecycle)
├── embedded.go        aliases      ├── config.go      (builder options)
├── errors.go          + thin       ├── embedded.go    (embedded mode)
├── backup.go          wrappers)    ├── errors.go      (error types)
├── rebuild.go                      ├── backup.go      (backup API)
├── compaction.go                   ├── rebuild.go     (rebuild API)
├── wal_prune.go                    ├── compaction.go  (compaction API)
└── doc.go                          └── wal_prune.go   (WAL prune API)
                                    internal/library/
Rust Shim (shim/)                   ├── library.go     (FFI loading)
└── src/lib.rs ◄────────────────    ├── library_unix.go
    (chroma_* symbols)              └── library_windows.go

Java scaffold (java/)
├── core   (shared API models)
├── jna    (Java 17 fallback)
└── panama (Java 22 primary)
```

- **No cgo**: Uses purego for pure Go FFI
- **Runtime artifact name**: `chroma_shim` (`libchroma_shim.so`, `libchroma_shim.dylib`, `chroma_shim.dll`)
- **Configuration**: YAML-based embedded startup config
- **Resource cleanup**: explicit close semantics in Go and Java runtime/session wrappers, with Go finalizers as a fallback safety net
- **Facade pattern**: Root package re-exports all symbols via type aliases (`type X = runtime.X`) and thin wrapper functions; zero implementation logic at root

## Key Patterns

Builder pattern for configuration:
```go
server, err := chroma.NewServer(
    chroma.WithPort(8000),
    chroma.WithPersistPath("./chroma_data"),
)
```

YAML string config alternative:
```go
server, err := chroma.StartServer(chroma.StartServerConfig{
    ConfigString: yamlString,
})
```

Facade pattern (root package):
```go
// Type aliases forward types from internal/runtime
type Server = runtime.Server
type ServerOption = runtime.ServerOption

// Thin wrappers forward functions
var NewServer = runtime.NewServer
var StartServer = runtime.StartServer
```

The root package contains zero logic -- all implementation lives in `internal/runtime/` and `internal/library/`.

## Linting

- Go: `golangci-lint run ./...` (config in `.golangci.yml`)
- Rust: `cargo clippy --locked -- -D warnings`
- Actions syntax/expressions: the actionlint version pinned in `.actionlint-version`
- Embedded workflow shell: ShellCheck through actionlint, with the repository's SC2129 exception
- YAML: `yamllint -c .yamllint .` across the repository
- Java (separate target): `gradle --no-daemon :core:check :jna:check :panama:check`

`make lint-workflows` and `pwsh -File .\scripts\dev-windows.ps1 -Task lint-workflows` both run the pinned actionlint module through Go, then repository-wide yamllint. Automatic toolchain switching lets Go 1.21+ select the toolchain requested by the module. A local Go 1.24+ toolchain is needed only when switching is unavailable or disabled. A direct `go install` of the pinned module has the same conditional requirement. Official prebuilt actionlint binaries can run directly without Go, but installing one does not change what the repository targets invoke.

Install ShellCheck 0.9 or newer and yamllint with `sudo apt install shellcheck yamllint` on Debian/Ubuntu, `brew install shellcheck yamllint` on macOS, or the following on Windows. CI and local installations may use newer versions; printed tool versions are diagnostic-only, not exact-version gates.

```powershell
winget install --id koalaman.shellcheck
py -m pip install --user yamllint
```

On Windows, `scripts/dev-windows.ps1 -Task lint` runs Go, Rust, Actions, embedded-shell, and YAML checks in that order. The dedicated `-Task lint-workflows` entry point runs only the workflow lint contract. CI mirrors it in a standalone Ubuntu 24.04 `workflow-lint` job.

`.yamllint` obtains repository-local exclusions from `.gitignore`; only paths that file actually matches are skipped. Do not assume an arbitrary relocated `CARGO_TARGET_DIR` is excluded.
