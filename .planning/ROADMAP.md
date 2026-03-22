# Roadmap: chroma-go-local

## Milestones

- **v0.4.0 Go Subtree Reorganization** - Phases 1-5 (complete)
- **v0.5.0 Java API Surface** - Phases 6-10 (in progress)

## Phases

<details>
<summary>v0.4.0 Go Subtree Reorganization (Phases 1-5)</summary>

- [x] **Phase 1: Layout Design** - Lock directory structure, create skeleton packages, confirm `internal/` is anchored at module root
- [x] **Phase 2: File Migration** - Move all Go implementation files into `internal/runtime/` and `internal/library/`, co-locate tests
- [x] **Phase 3: Root Facade** - Write thin facade at repo root that re-exports every public symbol via type aliases and function vars
- [x] **Phase 4: Build and Test** - Update Makefile, CI, lint config, and test layout; verify all `make` targets and cross-compile pass
- [x] **Phase 5: Compatibility and Docs** - Run `go-apidiff` against v0.3.4, add `compat_test.go`, update CLAUDE.md and docs

</details>

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

### v0.5.0 Java API Surface

- [ ] **Phase 6: Core Foundation Types** - Define all shared interfaces, builders, result POJOs, and FFI safety patterns in the core module with no FFI dependencies
- [ ] **Phase 7: Server Lifecycle** - Implement server start/stop/close and connection accessors in both JNA and Panama backends
- [ ] **Phase 8: Embedded Maintenance** - Implement rebuild, compaction, and WAL prune operations on EmbeddedSession in both backends
- [ ] **Phase 9: Backup API** - Implement backup with filesystem copy, manifest generation, and option builder for both embedded and server modes
- [ ] **Phase 10: Server Maintenance** - Implement stop-embed-op-restart orchestration for all maintenance operations on ServerSession

## Phase Details

### Phase 6: Core Foundation Types
**Goal**: The core module contains all shared interfaces, builders, result types, and FFI safety infrastructure so that backend modules (JNA, Panama) can implement against stable contracts without any FFI dependency in core
**Depends on**: Nothing (first phase of v0.5.0 milestone)
**Requirements**: FOUND-01, FOUND-02, FOUND-03, FOUND-04, FOUND-05, FOUND-06
**Success Criteria** (what must be TRUE):
  1. `ServerConfigBuilder` produces YAML that matches Go's config.go format field-for-field; a golden test compares output against known-good YAML strings
  2. `EmbeddedConfigBuilder` produces YAML for embedded startup; existing `startEmbedded(yaml)` callers can replace hand-written YAML with the builder
  3. All result POJOs (BackupManifest, RebuildCollectionResult, CompactionResult, WALPruneResult) are defined in core and can be constructed and serialized without FFI
  4. FFI serialization lock pattern and string ownership helpers (readOwnedString / readBorrowedString) are established and retrofitted into existing JNA and Panama call sites
  5. `gradle :core:build` succeeds with zero JNA or Panama imports in the core module
**Plans**: 3 plans

Plans:
- [x] 06-01-PLAN.md -- Build deps (Gson + SnakeYAML), JsonUtil, and all result POJOs
- [x] 06-02-PLAN.md -- Option/request types with Builders and config builders with YAML output
- [x] 06-03-PLAN.md -- AbstractChromaRuntime FFI safety, ServerSession, ChromaRuntime extension

### Phase 7: Server Lifecycle
**Goal**: Users can start a Chroma server from Java, retrieve its connection details, and cleanly shut it down using try-with-resources in both JNA and Panama backends
**Depends on**: Phase 6
**Requirements**: SRVR-01, SRVR-02, SRVR-03, SRVR-04
**Success Criteria** (what must be TRUE):
  1. `ChromaRuntime.startServer(configYaml)` returns a `ServerSession` that is listening on the configured port in both JNA and Panama backends
  2. `ServerSession.port()`, `address()`, and `url()` return correct connection details that match the startup configuration
  3. `ServerSession.close()` performs two-step teardown (stop then free) and is idempotent -- calling close twice does not crash the JVM
  4. Integration tests in both JNA and Panama verify server start, accessor values, stop, and close lifecycle
**Plans**: TBD

Plans:
- [ ] 07-01: TBD

### Phase 8: Embedded Maintenance
**Goal**: Users can perform rebuild, compaction, and WAL prune operations on an embedded Chroma instance through EmbeddedSession in both JNA and Panama backends
**Depends on**: Phase 6
**Requirements**: EMNT-01, EMNT-02, EMNT-03, EMNT-04, EMNT-05
**Success Criteria** (what must be TRUE):
  1. `EmbeddedSession.rebuildCollection(name, options)` returns a typed RebuildCollectionResult in both backends
  2. `EmbeddedSession.compactCollection(request)` and `compactAll(request)` return CompactionResult in both backends
  3. `EmbeddedSession.pruneCollectionWAL(name, options)` and `pruneAllWAL(options)` return WALPruneResult in both backends
  4. Option builders (RebuildOptions, WALPruneOptions) reject invalid inputs at build time with clear error messages
  5. Integration tests verify each maintenance operation produces valid results against a real embedded instance in both backends
**Plans**: TBD

Plans:
- [ ] 08-01: TBD

### Phase 9: Backup API
**Goal**: Users can back up Chroma data from both embedded and server modes, producing a directory with a manifest file that records backup metadata
**Depends on**: Phase 7, Phase 8
**Requirements**: BKUP-01, BKUP-02, BKUP-03, BKUP-04
**Success Criteria** (what must be TRUE):
  1. `EmbeddedSession.backup(options)` creates a backup directory at the specified destination and returns a BackupManifest with correct metadata
  2. `ServerSession.backup(options)` performs a stop-backup-restart cycle and returns a BackupManifest without corrupting the running server state
  3. `BackupOptions` builder supports destination, includeMetadata, and leaveClosed/leaveStopped flags with validation at build time
  4. Integration tests verify backup creates a valid output directory with expected contents in both JNA and Panama backends
**Plans**: TBD

Plans:
- [ ] 09-01: TBD

### Phase 10: Server Maintenance
**Goal**: Users can perform rebuild, compaction, and WAL prune operations on a server-mode Chroma instance, with the server automatically stopping and restarting around each maintenance operation
**Depends on**: Phase 7, Phase 8
**Requirements**: SMNT-01, SMNT-02, SMNT-03, SMNT-04
**Success Criteria** (what must be TRUE):
  1. `ServerSession.rebuildCollection(name, options)` stops the server, runs rebuild via a temporary embedded session, and restarts the server transparently
  2. `ServerSession.compactCollection(request)` and `compactAll(request)` use the stop-embed-op-restart pattern and return CompactionResult
  3. `ServerSession.pruneCollectionWAL(name, options)` and `pruneAllWAL(options)` use the stop-embed-op-restart pattern and return WALPruneResult
  4. Integration tests verify each server maintenance operation completes and the server is accessible again afterward in both backends
**Plans**: TBD

Plans:
- [ ] 10-01: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 6 -> 7 -> 8 -> 9 -> 10
(Phases 7 and 8 can execute in parallel after Phase 6; Phases 9 and 10 depend on both 7 and 8.)

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Layout Design | v0.4.0 | 1/1 | Complete | 2026-03-20 |
| 2. File Migration | v0.4.0 | 2/2 | Complete | 2026-03-20 |
| 3. Root Facade | v0.4.0 | 2/2 | Complete | - |
| 4. Build and Test | v0.4.0 | 3/3 | Complete | - |
| 5. Compatibility and Docs | v0.4.0 | 2/2 | Complete | - |
| 6. Core Foundation Types | v0.5.0 | 0/3 | Not started | - |
| 7. Server Lifecycle | v0.5.0 | 0/? | Not started | - |
| 8. Embedded Maintenance | v0.5.0 | 0/? | Not started | - |
| 9. Backup API | v0.5.0 | 0/? | Not started | - |
| 10. Server Maintenance | v0.5.0 | 0/? | Not started | - |
