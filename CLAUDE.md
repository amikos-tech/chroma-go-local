# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A minimal Go wrapper for running Chroma (vector database) as an embedded server using a Rust FFI shim and purego (no cgo required).

## Requirements

- Go 1.21+
- Rust 1.70+
- golangci-lint (for linting)

## Build Commands

```bash
make build          # Build Rust shim (debug)
make build-release  # Build Rust shim (release)
make test           # Build debug + run Go tests
make test-release   # Build release + run Go tests
make lint           # Run all linters (Go + Rust)
make fmt            # Format all code (Go + Rust)
make clean          # Clean build artifacts
```

## Testing

Tests require the Rust shim to be built first. The Makefile handles this automatically:
- `make test` builds debug and runs tests
- `make test-release` builds release and runs tests
- `CHROMA_LIB_PATH` is set automatically by Makefile

Tests are integration tests that start actual servers and make HTTP requests.

## Architecture

```
Go Package (chroma/)          Rust Shim (shim/)
├── chroma.go    ─────────►   src/lib.rs (FFI exports)
│   (Server lifecycle)            ├── chroma_server_start
├── config.go                     ├── chroma_server_stop
│   (Builder pattern)             ├── chroma_server_port
├── library.go                    ├── chroma_server_address
│   (purego FFI loading)          └── ...
└── errors.go
    (Error codes)
```

- **No cgo**: Uses purego for pure Go FFI
- **Tokio runtime**: Managed per Server instance in Rust
- **Configuration**: YAML-based with environment overrides (CHROMA_ prefix)
- **Resource cleanup**: Go runtime finalizers for server instances

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

## Linting

- Go: `golangci-lint run ./...` (config in `.golangci.yml`)
- Rust: `cargo clippy -- -D warnings` (warnings as errors)
