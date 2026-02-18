# Go API Surface (Current)

This document summarizes the Go API surface currently implemented in this repository for both server and embedded modes, with practical examples.

## 1. Initialization

You must initialize the shared library before using server or embedded clients.

```go
if err := chroma.Init(""); err != nil {
    panic(err)
}
fmt.Println("shim version:", chroma.Version())
```

`Init("")` uses `CHROMA_LIB_PATH` when no explicit path is provided.

## 2. Server Mode API

Implemented server lifecycle APIs:

- `NewServer(opts ...ServerOption) (*Server, error)`
- `StartServer(config StartServerConfig) (*Server, error)`
- `(*Server).Port() int`
- `(*Server).Address() string`
- `(*Server).URL() string`
- `(*Server).Stop() error`
- `(*Server).Close() error`

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

## 3. Embedded Mode API

### 3.1 Start and Stop

- `NewEmbedded(opts ...EmbeddedOption) (*Embedded, error)`
- `StartEmbedded(config StartEmbeddedConfig) (*Embedded, error)`
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

### 3.2 Health and Runtime Status

- `(*Embedded).Heartbeat() (uint64, error)`
- `(*Embedded).MaxBatchSize() (uint32, error)`
- `(*Embedded).Healthcheck() (*EmbeddedHealthCheckResponse, error)`
- `(*Embedded).IndexingStatus(request EmbeddedIndexingStatusRequest) (*EmbeddedIndexingStatusResponse, error)`
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
- `(*Embedded).UpdateCollection(request EmbeddedUpdateCollectionRequest) error` (rename-focused)
- `(*Embedded).DeleteCollection(request EmbeddedDeleteCollectionRequest) error`
- `(*Embedded).ForkCollection(request EmbeddedForkCollectionRequest) (*EmbeddedCollection, error)`

```go
col, _ := embedded.CreateCollection(chroma.EmbeddedCreateCollectionRequest{
    Name:         "docs",
    DatabaseName: "my_db",
    GetOrCreate:  true,
})

_ = embedded.UpdateCollection(chroma.EmbeddedUpdateCollectionRequest{
    CollectionID: col.ID,
    NewName:      "docs_v2",
})
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
})

result, _ := embedded.Query(chroma.EmbeddedQueryRequest{
    CollectionID:    col.ID,
    DatabaseName:    "my_db",
    QueryEmbeddings: [][]float32{{0.1, 0.2, 0.3}},
    NResults:        1,
})
fmt.Println(result.IDs)
```

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
