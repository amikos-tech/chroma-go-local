# local-go-chroma

A minimal Go wrapper for running Chroma from Go using a Rust FFI shim and [purego](https://github.com/ebitengine/purego) (no cgo required).

It supports both:
- server mode (starts the HTTP frontend)
- embedded mode (direct in-process calls, no HTTP port)

## Requirements

- Go 1.21+
- Rust 1.70+
- `golangci-lint` (for `lint` checks)

## Supported Platform Matrix

The table below captures current support and CI coverage for this repository.

| OS | Arch | CI coverage | Release shim archive | Notes |
|---|---|---|---|---|
| Linux | amd64 | yes | yes | Fully exercised in CI. |
| macOS | amd64 | no | no | Not in current hosted CI matrix. |
| Windows | amd64 | yes | yes | Use the documented PowerShell workflow for local dev. |
| Linux | arm64 | no | no | Not in current hosted CI matrix. |
| macOS | arm64 | yes | yes | Fully exercised in CI. |
| Windows | arm64 | no | no | Toolchain is documented, but CI/release artifacts are not yet published. |

See [Prebuilt Shim Artifacts](#prebuilt-shim-artifacts) for archive naming and [Windows toolchain setup](#windows-toolchain-setup) for local Windows prerequisites.

## Integration Direction (`chroma-go` PersistentClient)

This repository is the low-level runtime layer (`purego` + Rust shim). It is intended to be consumed by `github.com/amikos-tech/chroma-go` for a downstream `PersistentClient`.

Design intent:

- `local-go-chroma` remains independent and does not import `chroma-go`
- `chroma-go` depends on `local-go-chroma` to embed Chroma in Go apps
- integration and compatibility tests for `PersistentClient` should live in `chroma-go`

## Building

```bash
# Build debug version
make build

# Build release version
make build-release
```

## Windows Developer Workflow (PowerShell)

Use the PowerShell helper on Windows for native build/test/lint parity:

```powershell
pwsh -File .\scripts\dev-windows.ps1 -Task help
```

On Windows, prefer the PowerShell workflow for `test`, `test-release`, and `bench-go`; these Make targets are intentionally guarded on Windows Make hosts to avoid path translation issues.

### Windows toolchain setup

1. Install Go 1.21+.
2. Install Rust with an MSVC target toolchain:

```powershell
# x64 Windows
rustup toolchain install stable-x86_64-pc-windows-msvc
rustup default stable-x86_64-pc-windows-msvc
```

```powershell
# ARM64 Windows
rustup toolchain install stable-aarch64-pc-windows-msvc
rustup default stable-aarch64-pc-windows-msvc
```

3. Install `protoc` 31.x (matches Chroma `1.4.1` toolchain and this repo's CI).
4. Install `golangci-lint`.
5. Install `goimports`:

```powershell
go install golang.org/x/tools/cmd/goimports@latest
```

### Common Windows commands

```powershell
# Build debug shim
pwsh -File .\scripts\dev-windows.ps1 -Task build

# Run Go tests (builds debug shim and sets CHROMA_LIB_PATH automatically)
pwsh -File .\scripts\dev-windows.ps1 -Task test

# Run Rust tests
pwsh -File .\scripts\dev-windows.ps1 -Task test-rust

# Run linters (golangci-lint + cargo clippy)
pwsh -File .\scripts\dev-windows.ps1 -Task lint
```

## Prebuilt Shim Artifacts

Tag pushes matching `v*` trigger the release workflow in `.github/workflows/release.yml`, which publishes downloadable shim archives in the GitHub [Releases](https://github.com/amikos-tech/chroma-go-local/releases).

Archive naming is stable:

- `chroma-go-shim-linux-<arch>.tar.gz`
- `chroma-go-shim-macos-<arch>.tar.gz`
- `chroma-go-shim-windows-<arch>.tar.gz`
- `chroma-go-shim_SHA256SUMS.txt` (combined checksums for all archives)

Architecture note: archive `<arch>` is derived from the GitHub runner architecture. In the current hosted matrix for this repository, Linux/Windows builds are `amd64` and macOS builds are `arm64`. Runner mappings can change over time.

Library filename mapping inside each archive:

| OS | Library filename |
|---|---|
| Linux | `libchroma_go_shim.so` |
| macOS | `libchroma_go_shim.dylib` |
| Windows | `chroma_go_shim.dll` |

Example usage:

```bash
# Linux/macOS
tar -xzf chroma-go-shim-linux-amd64.tar.gz
export CHROMA_LIB_PATH="$(pwd)/libchroma_go_shim.so"
```

```powershell
# Windows PowerShell
tar -xzf chroma-go-shim-windows-amd64.tar.gz
$env:CHROMA_LIB_PATH = (Resolve-Path .\chroma_go_shim.dll).Path
```

Verify release checksums:

```bash
# Linux
sha256sum -c chroma-go-shim_SHA256SUMS.txt

# macOS
shasum -a 256 -c chroma-go-shim_SHA256SUMS.txt
```

```powershell
# Windows PowerShell
Get-Content chroma-go-shim_SHA256SUMS.txt | ForEach-Object {
    if (-not $_) { return }
    $expected, $file = $_ -split '  ', 2
    $actual = (Get-FileHash -Algorithm SHA256 $file).Hash.ToLowerInvariant()
    if ($actual -eq $expected) { "OK: $file" } else { throw "MISMATCH: $file" }
}
```

## Usage

```go
package main

import (
    "fmt"
    "os"

    chroma "github.com/amikos-tech/chroma-go-local"
)

func main() {
    // Initialize - set CHROMA_LIB_PATH or pass path directly
    if err := chroma.Init(""); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to initialize: %v\n", err)
        os.Exit(1)
    }

    // Start server with builder pattern
    server, err := chroma.NewServer(
        chroma.WithPort(8000),
        chroma.WithPersistPath("./chroma_data"),
        chroma.WithAllowReset(true),
    )
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
        os.Exit(1)
    }
    defer server.Close()

    fmt.Printf("Server running at %s\n", server.URL())

    // Use any Chroma client to connect to the server...
}
```

## Embedded Mode (No HTTP Port)

```go
embedded, err := chroma.NewEmbedded(
    chroma.WithEmbeddedPersistPath("./chroma_data"),
    chroma.WithEmbeddedAllowReset(true),
)
if err != nil {
    panic(err)
}
defer embedded.Close()

collection, err := embedded.CreateCollection(chroma.EmbeddedCreateCollectionRequest{
    Name: "docs",
})
if err != nil {
    panic(err)
}

err = embedded.Add(chroma.EmbeddedAddRequest{
    CollectionID: collection.ID,
    IDs:          []string{"doc-1"},
    Embeddings:   [][]float32{{0.1, 0.2, 0.3}},
})
if err != nil {
    panic(err)
}

result, err := embedded.Query(chroma.EmbeddedQueryRequest{
    CollectionID:    collection.ID,
    QueryEmbeddings: [][]float32{{0.1, 0.2, 0.3}},
    NResults:        1,
})
if err != nil {
    panic(err)
}
fmt.Println(result.IDs)
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithPort(port)` | Server port | 8000 |
| `WithListenAddress(addr)` | Bind address | "127.0.0.1" |
| `WithPersistPath(path)` | Data directory | "./chroma" |
| `WithAllowReset(bool)` | Enable reset endpoint | false |
| `WithMaxPayloadSize(bytes)` | Max request size | 40 MB |
| `WithCORSAllowOrigins(origins...)` | CORS allowed origins | none |
| `WithSQLiteFilename(name)` | SQLite DB filename | "chroma.sqlite3" |
| `WithOpenTelemetry(endpoint, service)` | Enable OTel tracing | disabled |
| `WithRawYAML(yaml)` | Raw YAML config (overrides all) | - |

### Alternative: YAML config file

```go
server, err := chroma.StartServer(chroma.StartServerConfig{
    ConfigPath: "./config.yaml",
})
```

### Alternative: Inline YAML string

```go
server, err := chroma.StartServer(chroma.StartServerConfig{
    ConfigString: `
port: 8000
persist_path: "./chroma_data"
allow_reset: true
`,
})
```

## API

For a detailed, example-heavy reference of the currently implemented Go APIs, see [`GO_API_SURFACE.md`](GO_API_SURFACE.md).

| Function | Description |
|----------|-------------|
| `Init(libPath string) error` | Initialize the library. Uses `CHROMA_LIB_PATH` env if path is empty. |
| `Version() string` | Returns the shim version. |
| `NewServer(opts ...ServerOption) (*Server, error)` | Start a server with builder options. |
| `StartServer(config StartServerConfig) (*Server, error)` | Start a server with YAML config. |
| `(*Server) Port() int` | Get the server port. |
| `(*Server) Address() string` | Get the server listen address. |
| `(*Server) URL() string` | Get the full server URL. |
| `(*Server) Stop() error` | Gracefully stop the server. |
| `(*Server) Close() error` | Stop and free resources. |
| `NewEmbedded(opts ...EmbeddedOption) (*Embedded, error)` | Start in-process embedded mode. |
| `StartEmbedded(config StartEmbeddedConfig) (*Embedded, error)` | Start embedded mode from YAML config. |
| `(*Embedded) Heartbeat() (uint64, error)` | Read in-process heartbeat nanoseconds. |
| `(*Embedded) MaxBatchSize() (uint32, error)` | Get embedded max batch size. |
| `(*Embedded) CreateTenant(request EmbeddedCreateTenantRequest) error` | Create a tenant in embedded mode. |
| `(*Embedded) GetTenant(request EmbeddedGetTenantRequest) (*EmbeddedTenant, error)` | Get a tenant by name. |
| `(*Embedded) UpdateTenant(request EmbeddedUpdateTenantRequest) error` | Update tenant properties. |
| `(*Embedded) Healthcheck() (*EmbeddedHealthCheckResponse, error)` | Get embedded readiness state. |
| `(*Embedded) CreateDatabase(request EmbeddedCreateDatabaseRequest) error` | Create a database in embedded mode. |
| `(*Embedded) ListDatabases(request EmbeddedListDatabasesRequest) ([]EmbeddedDatabase, error)` | List databases in embedded mode. |
| `(*Embedded) GetDatabase(request EmbeddedGetDatabaseRequest) (*EmbeddedDatabase, error)` | Get a database by name. |
| `(*Embedded) DeleteDatabase(request EmbeddedDeleteDatabaseRequest) error` | Delete a database by name. |
| `(*Embedded) CreateCollection(request EmbeddedCreateCollectionRequest) (*EmbeddedCollection, error)` | Create a collection without HTTP. |
| `(*Embedded) ListCollections(request EmbeddedListCollectionsRequest) ([]EmbeddedCollection, error)` | List collections for a database. |
| `(*Embedded) GetCollection(request EmbeddedGetCollectionRequest) (*EmbeddedCollection, error)` | Get a collection by name. |
| `(*Embedded) CountCollections(request EmbeddedCountCollectionsRequest) (uint32, error)` | Count collections for a database. |
| `(*Embedded) UpdateCollection(request EmbeddedUpdateCollectionRequest) error` | Update a collection (rename-focused). |
| `(*Embedded) DeleteCollection(request EmbeddedDeleteCollectionRequest) error` | Delete a collection by name. |
| `(*Embedded) ForkCollection(request EmbeddedForkCollectionRequest) (*EmbeddedCollection, error)` | Fork a collection (may be unimplemented in local mode). |
| `(*Embedded) CountRecords(request EmbeddedCountRecordsRequest) (uint32, error)` | Count records for a collection. |
| `(*Embedded) GetRecords(request EmbeddedGetRecordsRequest) (*EmbeddedGetRecordsResponse, error)` | Get records from a collection (supports `where` and `where_document`). |
| `(*Embedded) UpdateRecords(request EmbeddedUpdateRecordsRequest) error` | Update existing records by id. |
| `(*Embedded) UpsertRecords(request EmbeddedUpsertRecordsRequest) error` | Upsert records by id. |
| `(*Embedded) DeleteRecords(request EmbeddedDeleteRecordsRequest) error` | Delete records by ids and/or filters. |
| `(*Embedded) Add(request EmbeddedAddRequest) error` | Add records without HTTP. |
| `(*Embedded) Query(request EmbeddedQueryRequest) (*EmbeddedQueryResponse, error)` | Query records without HTTP (supports `where` and `where_document`). |
| `(*Embedded) IndexingStatus(request EmbeddedIndexingStatusRequest) (*EmbeddedIndexingStatusResponse, error)` | Get collection indexing status (may be unimplemented in local backend). |
| `(*Embedded) Reset() error` | Reset local state when enabled. |
| `(*Embedded) Close() error` | Free embedded resources. |

### Metadata Value Rules (Embedded Record APIs)

For `EmbeddedAddRequest.Metadatas`, `EmbeddedUpdateRecordsRequest.Metadatas`, and `EmbeddedUpsertRecordsRequest.Metadatas`:

- Supported scalar values: `bool`, integer types, float types, `string`
- Supported arrays: homogeneous arrays of one supported scalar type
- Unsupported values: nested objects/maps, mixed-type arrays, structs

`UpdateRecords` and `UpsertRecords` allow `nil` metadata values to clear keys. Float metadata values are encoded with an explicit decimal representation to avoid integer/float array ambiguity at the Go/Rust boundary.

## Testing

```bash
make test-go       # Run Go tests (unit + integration + property tests)
make test-rust     # Run Rust shim tests (unit + proptests + FFI integration)
make test-all      # Run both Go and Rust tests
make test-release  # Run Go tests with release build
```

```powershell
# Windows PowerShell equivalents
pwsh -File .\scripts\dev-windows.ps1 -Task test
pwsh -File .\scripts\dev-windows.ps1 -Task test-rust
pwsh -File .\scripts\dev-windows.ps1 -Task test-all
pwsh -File .\scripts\dev-windows.ps1 -Task test-release
```

## CI

GitHub Actions runs a cross-platform matrix (`ubuntu-latest`, `macos-latest`, `windows-latest`) on pushes to `main` and pull requests. Each matrix job runs:

1. `cargo build --locked` in `shim/`
2. `go test -v ./...` with platform-specific `CHROMA_LIB_PATH`
3. `golangci-lint run ./...`
4. `cargo clippy --locked -- -D warnings` in `shim/`

Release tags (`v*`) run a separate workflow that builds release shim archives and publishes them together with `chroma-go-shim_SHA256SUMS.txt`.

## Troubleshooting

### Dynamic loading (`Init` / `CHROMA_LIB_PATH`)

If `Init("")` fails, validate all of the following first:

- `CHROMA_LIB_PATH` should be absolute for clarity. Relative paths that include separators are also supported and resolved by the loader.
- The library filename matches your platform (see [Prebuilt Shim Artifacts](#prebuilt-shim-artifacts)).
- The library exists at that exact path.

Quick verification:

```bash
# Linux/macOS
echo "$CHROMA_LIB_PATH"
ls -l "$CHROMA_LIB_PATH"
```

```powershell
# Windows PowerShell
echo $env:CHROMA_LIB_PATH
Test-Path $env:CHROMA_LIB_PATH
```

### Linux and macOS

- If the loader reports file-not-found, confirm extension and filename are correct for the platform.
- If using release downloads, re-check archive checksums before loading.
- On macOS, if the downloaded file is quarantined by Gatekeeper, remove quarantine metadata:

```bash
xattr -dr com.apple.quarantine /path/to/libchroma_go_shim.dylib
```

### Windows

- Prefer the PowerShell helper commands in this README (`scripts/dev-windows.ps1`) instead of `make` for test/lint/bench flows.
- Ensure the Rust MSVC target is active and `protoc` 31.x is installed before running tests.
- If path issues appear, set `CHROMA_LIB_PATH` via `Resolve-Path` as shown in [Prebuilt Shim Artifacts](#prebuilt-shim-artifacts).

### Build and test failures

- `protoc` version mismatches are a common source of build failures; use `31.x`.
- If Rust or Go dependencies are corrupted locally, clear build outputs and rerun:

```bash
make clean
make test-go
```

## Benchmarks

```bash
make bench-go      # Run Go benchmarks
make bench-rust    # Run Rust criterion benchmarks
make bench         # Run both benchmark suites
```

## Project Structure

```
.
├── chroma.go       # Main Go wrapper
├── config.go       # Server config builder with WithXXX options
├── embedded.go     # Embedded (in-process) API
├── library.go      # Library loading via purego
├── errors.go       # Error codes and handling
├── chroma_test.go  # Tests
├── embedded_test.go # Embedded integration test
├── Makefile        # Build orchestration
├── scripts/
│   └── dev-windows.ps1 # Windows build/test/lint helper
├── examples/
│   └── basic/      # Example usage
└── shim/
    ├── Cargo.toml  # Rust dependencies
    └── src/
        └── lib.rs  # Rust FFI shim
```

## License

MIT
