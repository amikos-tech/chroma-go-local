# Go API Surface (Current)

This document summarizes the Go API surface currently implemented in this repository for both server and embedded modes, with practical examples.

## 1. Initialization

You must initialize the shared library before using server or embedded clients.

```go
if err := chroma.Init(""); err != nil {
    panic(err)
}
fmt.Println("shim version:", chroma.Version())
v, err := chroma.VersionWithError()
fmt.Println(v, err)
```

`Init("")` uses `CHROMA_LIB_PATH` when no explicit path is provided.

Runtime artifact naming as of `v0.3.1`:
- Linux: `libchroma_shim.so`
- macOS: `libchroma_shim.dylib`
- Windows: `chroma_shim.dll`

## 2. Server Mode API

Implemented server lifecycle APIs:

- `NewServer(opts ...ServerOption) (*Server, error)`
- `StartServer(config StartServerConfig) (*Server, error)`
- `(*Server).Port() int`
- `(*Server).Address() string`
- `(*Server).URL() string`
- `(*Server).Stop() error`
- `(*Server).Close() error`
- `(*Server).Backup(options ...BackupOption) (*BackupManifest, error)`
- `(*Server).CompactCollection(request CompactCollectionRequest) (*CompactionResult, error)` (supports `TenantID` + `DatabaseName` scope together)
- `(*Server).CompactAll(request CompactAllRequest) (*CompactionResult, error)` (supports `TenantID` + `DatabaseName` scope together; `result.Collections[i].Error` reports per-collection failures)

Example:

```go
srv, err := chroma.NewServer(
    chroma.WithPort(8000),
    chroma.WithPersistPath("./chroma_data"),
    chroma.WithAllowReset(true),
)
if err != nil {
    panic(err)
}
defer srv.Close()

fmt.Println("running at", srv.URL())
```

Server backup example:

```go
manifest, err := srv.Backup(
    chroma.WithDestination("./backups/server-2026-02-25"),
    chroma.WithIncludeMetadata(),
)
if err != nil {
    panic(err)
}
fmt.Println("backup manifest:", manifest.ManifestPath)
```

Server compaction example:

```go
result, err := srv.CompactCollection(chroma.CompactCollectionRequest{
    Name:         "docs",
    TenantID:     "team_a",
    DatabaseName: "prod_db",
})
if err != nil {
    panic(err)
}
fmt.Println("compacted collections:", result.CollectionCount)
```

Compaction semantics:

- `CompactCollection` and `CompactAll` run explicit compaction via Chroma's local compaction manager.
- You can pass both `TenantID` and `DatabaseName` in the same request.
- For `CompactCollection`, collection name lookup is performed inside that tenant+database scope.
- When omitted, tenant/database scope defaults to `default_tenant` and `default_database`.
- For each collection, compaction runs backfill then log purge (WAL cleanup).
- This is not a full HNSW rebuild from scratch and does not change collection configuration/schema.
- In server mode, the server is unavailable while compaction runs (stop -> compact in embedded mode -> restart).
- `CompactAll` continues across collections and reports per-collection failures in `result.Collections[i].Error`.
- `result.CollectionCount` is attempted collections, not only successful collections.
- `pending_ops_before`/`pending_ops_after` are advisory; if unavailable they are omitted and surfaced via `pending_ops_before_error`/`pending_ops_after_error`.

Backup constraints (applies to server and embedded backup):

- `DestinationPath` must not exist or must be an empty directory.
- `<destination>/persist` must not be inside the source persist path (symlink-aware check).
- Symlinks inside the source persist tree are rejected and cause backup to fail.
- `WithLeaveStopped()` is server-only; `WithLeaveClosed()` is embedded-only.
- `WithIncludeMetadata()` adds per-file entries with `path`, `size_bytes`, `mode`, `sha256`, and `modified_at`.

## 3. Embedded Mode API

### 3.1 Start and Stop

- `NewEmbedded(opts ...EmbeddedOption) (*Embedded, error)`
- `StartEmbedded(config StartEmbeddedConfig) (*Embedded, error)`
- `(*Embedded).Backup(options ...BackupOption) (*BackupManifest, error)`
- `(*Embedded).Close() error`

```go
embedded, err := chroma.NewEmbedded(
    chroma.WithEmbeddedPersistPath("./chroma_data"),
    chroma.WithEmbeddedAllowReset(true),
)
if err != nil {
    panic(err)
}
defer embedded.Close()
```

Embedded backup example:

```go
manifest, err := embedded.Backup(
    chroma.WithDestination("./backups/embedded-2026-02-25"),
    chroma.WithIncludeMetadata(),
)
if err != nil {
    panic(err)
}
fmt.Println("snapshot dir:", manifest.SnapshotPath)
```

### 3.2 Health and Runtime Status

- `(*Embedded).Heartbeat() (uint64, error)`
- `(*Embedded).MaxBatchSize() (uint32, error)`
- `(*Embedded).Healthcheck() (*EmbeddedHealthCheckResponse, error)`
- `(*Embedded).IndexingStatus(request EmbeddedIndexingStatusRequest) (*EmbeddedIndexingStatusResponse, error)`
- `(*Embedded).CompactCollection(request CompactCollectionRequest) (*CompactionResult, error)`
- `(*Embedded).CompactAll(request CompactAllRequest) (*CompactionResult, error)` (`result.Collections[i].Error` reports per-collection failures)
- `(*Embedded).Reset() error`

```go
heartbeat, _ := embedded.Heartbeat()
maxBatch, _ := embedded.MaxBatchSize()
health, _ := embedded.Healthcheck()

fmt.Println(heartbeat, maxBatch, health.IsExecutorReady, health.IsLogClientReady)
```

```go
status, err := embedded.IndexingStatus(chroma.EmbeddedIndexingStatusRequest{
    CollectionID: collectionID,
})
if err != nil {
    // local backends may return unimplemented for indexing status
    fmt.Println("indexing status unavailable:", err)
} else {
    fmt.Println(status.OpIndexingProgress, status.TotalOps)
}
```

```go
compacted, err := embedded.CompactAll(chroma.CompactAllRequest{
    TenantID:     "team_a",
    DatabaseName: "my_db",
})
if err != nil {
    panic(err)
}
fmt.Println(compacted.CollectionCount, compacted.DurationMS)
```

### 3.3 Tenant APIs

- `(*Embedded).CreateTenant(request EmbeddedCreateTenantRequest) error`
- `(*Embedded).GetTenant(request EmbeddedGetTenantRequest) (*EmbeddedTenant, error)`
- `(*Embedded).UpdateTenant(request EmbeddedUpdateTenantRequest) error`

```go
_ = embedded.CreateTenant(chroma.EmbeddedCreateTenantRequest{Name: "team_a"})

tenant, _ := embedded.GetTenant(chroma.EmbeddedGetTenantRequest{Name: "team_a"})
fmt.Println("tenant:", tenant.Name)

_ = embedded.UpdateTenant(chroma.EmbeddedUpdateTenantRequest{
    TenantID:     "team_a",
    ResourceName: "projects/demo",
})
```

### 3.4 Database APIs

- `(*Embedded).CreateDatabase(request EmbeddedCreateDatabaseRequest) error`
- `(*Embedded).ListDatabases(request EmbeddedListDatabasesRequest) ([]EmbeddedDatabase, error)`
- `(*Embedded).GetDatabase(request EmbeddedGetDatabaseRequest) (*EmbeddedDatabase, error)`
- `(*Embedded).DeleteDatabase(request EmbeddedDeleteDatabaseRequest) error`

```go
_ = embedded.CreateDatabase(chroma.EmbeddedCreateDatabaseRequest{Name: "my_db"})

dbs, _ := embedded.ListDatabases(chroma.EmbeddedListDatabasesRequest{})
fmt.Println("databases:", len(dbs))

db, _ := embedded.GetDatabase(chroma.EmbeddedGetDatabaseRequest{Name: "my_db"})
fmt.Println("database:", db.Name)
```

### 3.5 Collection APIs

- `(*Embedded).CreateCollection(request EmbeddedCreateCollectionRequest) (*EmbeddedCollection, error)`
- `(*Embedded).ListCollections(request EmbeddedListCollectionsRequest) ([]EmbeddedCollection, error)`
- `(*Embedded).GetCollection(request EmbeddedGetCollectionRequest) (*EmbeddedCollection, error)`
- `(*Embedded).CountCollections(request EmbeddedCountCollectionsRequest) (uint32, error)`
- `(*Embedded).UpdateCollection(request EmbeddedUpdateCollectionRequest) error` (name and/or metadata)
- `(*Embedded).DeleteCollection(request EmbeddedDeleteCollectionRequest) error`
- `(*Embedded).ForkCollection(request EmbeddedForkCollectionRequest) (*EmbeddedCollection, error)`

`EmbeddedCreateCollectionRequest` fields:
- `Name`
- `TenantID` (optional)
- `DatabaseName` (optional)
- `Metadata map[string]any` (optional)
- `Configuration map[string]any` (optional)
- `Schema map[string]any` (optional)
- `GetOrCreate` (optional)

`EmbeddedCollection` includes:
- `ID`, `Name`, `Tenant`, `Database`
- `Metadata`
- `ConfigurationJSON` (JSON key `configuration_json`)
- `Schema`

`EmbeddedUpdateCollectionRequest` fields:
- `CollectionID`
- `NewName` (optional)
- `NewMetadata map[string]any` (optional; replaces existing collection metadata; nil values are rejected)
- `DatabaseName` (optional)

At least one of `NewName` or `NewMetadata` is required.

```go
col, _ := embedded.CreateCollection(chroma.EmbeddedCreateCollectionRequest{
    Name:         "docs",
    DatabaseName: "my_db",
    Metadata: map[string]any{
        "owner": "qa",
        "active": true,
    },
    Configuration: map[string]any{
        "hnsw": map[string]any{
            "space": "cosine",
        },
    },
    GetOrCreate:  true,
})

copyCol, _ := embedded.CreateCollection(chroma.EmbeddedCreateCollectionRequest{
    Name:         "docs_schema_copy",
    DatabaseName: "my_db",
    Schema:       col.Schema,
    GetOrCreate:  true,
})

_ = embedded.UpdateCollection(chroma.EmbeddedUpdateCollectionRequest{
    CollectionID: col.ID,
    NewName:      "docs_v2",
    DatabaseName: "my_db",
})

_ = embedded.UpdateCollection(chroma.EmbeddedUpdateCollectionRequest{
    CollectionID: col.ID,
    NewMetadata: map[string]any{
        "owner": "platform",
    },
    DatabaseName: "my_db",
})

_ = copyCol
```

### 3.6 Record APIs

- `(*Embedded).Add(request EmbeddedAddRequest) error`
- `(*Embedded).Query(request EmbeddedQueryRequest) (*EmbeddedQueryResponse, error)`
- `(*Embedded).GetRecords(request EmbeddedGetRecordsRequest) (*EmbeddedGetRecordsResponse, error)`
- `(*Embedded).CountRecords(request EmbeddedCountRecordsRequest) (uint32, error)`
- `(*Embedded).UpdateRecords(request EmbeddedUpdateRecordsRequest) error`
- `(*Embedded).UpsertRecords(request EmbeddedUpsertRecordsRequest) error`
- `(*Embedded).DeleteRecords(request EmbeddedDeleteRecordsRequest) error`

```go
_ = embedded.Add(chroma.EmbeddedAddRequest{
    CollectionID: col.ID,
    DatabaseName: "my_db",
    IDs:          []string{"doc-1", "doc-2"},
    Embeddings:   [][]float32{{0.1, 0.2, 0.3}, {0.2, 0.2, 0.1}},
    Documents:    []string{"first", "second"},
    Metadatas: []map[string]any{
        {"labels": []string{"alpha", "beta"}, "scores": []float64{1.1, 2.2}},
        {"labels": []string{"beta", "gamma"}, "scores": []float64{3.3, 4.4}},
    },
})

result, _ := embedded.Query(chroma.EmbeddedQueryRequest{
    CollectionID:    col.ID,
    DatabaseName:    "my_db",
    QueryEmbeddings: [][]float32{{0.1, 0.2, 0.3}},
    NResults:        1,
})
fmt.Println(result.IDs)
```

Metadata values in `Add`/`UpdateRecords`/`UpsertRecords` support:

- scalar values: `bool`, `int`/`int64`, `float32`/`float64`, `string`
- homogeneous arrays of those scalar types

Nil metadata values are accepted for `UpdateRecords` and `UpsertRecords` to clear keys. Float values are encoded with an explicit decimal to avoid integer/float array ambiguity at the Go/Rust FFI boundary.
Numeric metadata values decode back as `float64` in `EmbeddedGetRecordsResponse.Metadatas` because Go JSON unmarshaling uses `map[string]any`.

## 4. Filter Support (`where`, `where_document`)

`Query`, `GetRecords`, and `DeleteRecords` support both metadata and document filters.

Metadata filter example (`where`):

```go
resp, _ := embedded.GetRecords(chroma.EmbeddedGetRecordsRequest{
    CollectionID: col.ID,
    DatabaseName: "my_db",
    Where: map[string]any{
        "$and": []any{
            map[string]any{"source": "blog"},
            map[string]any{"lang": "en"},
        },
    },
    Include: []string{"documents", "metadatas"},
})
fmt.Println(resp.IDs)
```

Document filter example (`where_document`):

```go
q, _ := embedded.Query(chroma.EmbeddedQueryRequest{
    CollectionID:    col.ID,
    DatabaseName:    "my_db",
    QueryEmbeddings: [][]float32{{0.1, 0.2, 0.3}},
    WhereDocument: map[string]any{
        "$contains": "updated",
    },
    NResults: 3,
})
fmt.Println(q.IDs)
```

Filtered delete example:

```go
_ = embedded.DeleteRecords(chroma.EmbeddedDeleteRecordsRequest{
    CollectionID: col.ID,
    DatabaseName: "my_db",
    WhereDocument: map[string]any{
        "$contains": "stale",
    },
})
```

`DeleteRecords` requires at least one of:

- `IDs`
- `Where`
- `WhereDocument`

## 5. Not Yet Exposed in This Go Surface

The following upstream capabilities are not currently bridged in this repo:

- `search` (separate from vector `query`)
- `get_collection_by_crn`
- `attach_function`
- `get_attached_function`
- `detach_function`
