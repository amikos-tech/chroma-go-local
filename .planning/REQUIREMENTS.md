# Requirements: chroma-go-local v0.5.0

**Defined:** 2026-03-21
**Core Value:** Java and Go APIs must provide equivalent access to all Chroma runtime capabilities

## v1 Requirements

### Foundation

- [ ] **FOUND-01**: Core module contains all shared interfaces, builders, and result types with no FFI dependencies
- [ ] **FOUND-02**: `ServerConfigBuilder` produces valid YAML for server startup with fluent API (port, listenAddress, persistPath, allowReset, etc.)
- [ ] **FOUND-03**: `EmbeddedConfigBuilder` produces valid YAML for embedded startup with fluent API (persistPath, sqliteFilename, allowReset)
- [ ] **FOUND-04**: Result POJOs defined for all maintenance operations (BackupManifest, RebuildCollectionResult, CompactionResult, WALPruneResult)
- [ ] **FOUND-05**: FFI serialization lock pattern established to protect global error slot
- [ ] **FOUND-06**: String ownership helpers distinguish owned vs borrowed native pointers

### Server Lifecycle

- [ ] **SRVR-01**: `ChromaRuntime.startServer(configYaml)` returns `ServerSession` in both JNA and Panama
- [ ] **SRVR-02**: `ServerSession` implements AutoCloseable with idempotent close and two-step teardown (stop + free)
- [ ] **SRVR-03**: `ServerSession.port()`, `address()`, `url()` return server connection details
- [ ] **SRVR-04**: Integration tests verify server start, accessor values, stop, and close in both backends

### Embedded Maintenance

- [ ] **EMNT-01**: `EmbeddedSession.rebuildCollection(name, options)` returns RebuildCollectionResult in both backends
- [ ] **EMNT-02**: `EmbeddedSession.compactCollection(request)` and `compactAll(request)` return CompactionResult in both backends
- [ ] **EMNT-03**: `EmbeddedSession.pruneCollectionWAL(name, options)` and `pruneAllWAL(options)` return WALPruneResult in both backends
- [ ] **EMNT-04**: Option builders (RebuildOptions, WALPruneOptions) validate inputs at build time
- [ ] **EMNT-05**: Integration tests verify each embedded maintenance operation in both backends

### Backup

- [ ] **BKUP-01**: `EmbeddedSession.backup(options)` performs directory copy with manifest and returns BackupManifest
- [ ] **BKUP-02**: `ServerSession.backup(options)` performs stop-backup-restart cycle and returns BackupManifest
- [ ] **BKUP-03**: `BackupOptions` builder supports destination, includeMetadata, leaveClosed/leaveStopped
- [ ] **BKUP-04**: Integration tests verify backup creates valid output directory in both backends

### Server Maintenance

- [ ] **SMNT-01**: `ServerSession.rebuildCollection(name, options)` uses stop-embed-op-restart pattern
- [ ] **SMNT-02**: `ServerSession.compactCollection(request)` and `compactAll(request)` use stop-embed-op-restart pattern
- [ ] **SMNT-03**: `ServerSession.pruneCollectionWAL(name, options)` and `pruneAllWAL(options)` use stop-embed-op-restart pattern
- [ ] **SMNT-04**: Integration tests verify server maintenance operations in both backends

## v2 Requirements

### Post-Parity Improvements

- **FUTURE-01**: `ChromaErrorCode` enum on `ChromaException` for programmatic error handling
- **FUTURE-02**: `BackupEngine`-style backup management (list, purge old, restore)
- **FUTURE-03**: Embedded data operations (CRUD for collections, documents, queries)
- **FUTURE-04**: Java-native OpenTelemetry integration through config builder

## Out of Scope

| Feature | Reason |
|---------|--------|
| JDBC/DataSource interface | ChromaDB is not relational; JDBC model does not map to vector operations |
| Async/CompletableFuture API | All FFI calls serialize through global lock; async would be misleading |
| Connection pooling | Embedded mode is single in-process runtime; pooling has no benefit |
| Auto-reconnect on server crash | Invalid native handle cannot be reconnected; must restart explicitly |
| Checked exceptions | ChromaException is already unchecked; changing would break existing users |
| New Rust shim exports | Java reuses existing chroma_* FFI symbols; no shim changes allowed |
| Maven Central publishing | Separate milestone after API stabilizes |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| FOUND-01 | Pending | Pending |
| FOUND-02 | Pending | Pending |
| FOUND-03 | Pending | Pending |
| FOUND-04 | Pending | Pending |
| FOUND-05 | Pending | Pending |
| FOUND-06 | Pending | Pending |
| SRVR-01 | Pending | Pending |
| SRVR-02 | Pending | Pending |
| SRVR-03 | Pending | Pending |
| SRVR-04 | Pending | Pending |
| EMNT-01 | Pending | Pending |
| EMNT-02 | Pending | Pending |
| EMNT-03 | Pending | Pending |
| EMNT-04 | Pending | Pending |
| EMNT-05 | Pending | Pending |
| BKUP-01 | Pending | Pending |
| BKUP-02 | Pending | Pending |
| BKUP-03 | Pending | Pending |
| BKUP-04 | Pending | Pending |
| SMNT-01 | Pending | Pending |
| SMNT-02 | Pending | Pending |
| SMNT-03 | Pending | Pending |
| SMNT-04 | Pending | Pending |

**Coverage:**
- v1 requirements: 23 total
- Mapped to phases: 0
- Unmapped: 23 ⚠️

---
*Requirements defined: 2026-03-21*
*Last updated: 2026-03-21 after initial definition*
