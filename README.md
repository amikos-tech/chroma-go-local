# local-go-chroma

A minimal Go wrapper for running Chroma from Go using a Rust FFI shim and [purego](https://github.com/ebitengine/purego) (no cgo required).

It supports both:
- server mode (starts the HTTP frontend)
- embedded mode (direct in-process calls, no HTTP port)

## Requirements

- Go 1.21+
- Rust 1.70+

## Building

```bash
# Build debug version
make build

# Build release version
make build-release
```

## Prebuilt Shim Artifacts

Tag pushes matching `v*` trigger the release workflow in `.github/workflows/release.yml`, which publishes downloadable shim archives in the GitHub [Releases](https://github.com/amikos-tech/chroma-go-local/releases).

Archive naming is stable:

- `chroma-go-shim-linux-<arch>.tar.gz`
- `chroma-go-shim-macos-<arch>.tar.gz`
- `chroma-go-shim-windows-<arch>.tar.gz`
- `chroma-go-shim_SHA256SUMS.txt` (combined checksums for all archives)

Architecture note: archive `<arch>` is derived from the GitHub runner architecture. With the current hosted matrix, releases are expected to be `amd64`; add ARM64 runners to publish native `arm64` archives.

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

    chroma "github.com/amikos-tech/local-go-chroma"
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

## Testing

```bash
make test-go       # Run Go tests (unit + integration + property tests)
make test-rust     # Run Rust shim tests (unit + proptests + FFI integration)
make test-all      # Run both Go and Rust tests
make test-release  # Run Go tests with release build
```

## CI

GitHub Actions runs a cross-platform matrix (`ubuntu-latest`, `macos-latest`, `windows-latest`) on pushes to `main` and pull requests. Each matrix job runs:

1. `cargo build --locked` in `shim/`
2. `go test -v ./...` with platform-specific `CHROMA_LIB_PATH`
3. `golangci-lint run ./...`
4. `cargo clippy --locked -- -D warnings` in `shim/`

Release tags (`v*`) run a separate workflow that builds release shim archives and publishes them together with `chroma-go-shim_SHA256SUMS.txt`.

### Benchmarks

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
├── examples/
│   └── basic/      # Example usage
└── shim/
    ├── Cargo.toml  # Rust dependencies
    └── src/
        └── lib.rs  # Rust FFI shim
```

## License

MIT
