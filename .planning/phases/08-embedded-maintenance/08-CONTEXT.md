# Phase 8: Embedded Maintenance - Context

**Gathered:** 2026-03-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire the 5 embedded maintenance FFI calls (rebuild_collection, compact_collection, compact_all, prune_wal_collection, prune_wal_all) through `EmbeddedSession` in both JNA and Panama backends, with integration tests covering both smoke and data-seeded scenarios. No new Rust shim exports — reuses existing `chroma_embedded_*` symbols.

</domain>

<decisions>
## Implementation Decisions

### EmbeddedSession API shape
- **D-01:** Callback slots pattern — add functional interface fields (e.g., `BiFunction<Long, String, RebuildCollectionResult>`) to `EmbeddedSession` constructor, backends inject lambdas at construction time
- **D-02:** Consistent with `ServerSession` pattern — both session types use callback slots, keeping core module FFI-free
- **D-03:** `EmbeddedSession` constructor signature expands with maintenance callbacks; existing `(long handle, LongConsumer closeAction)` gains additional function parameters

### FFI call wiring
- **D-04:** Eager symbol binding at `init()` — all 5 `chroma_embedded_*` maintenance symbols bound during backend initialization, same as existing server symbols
- **D-05:** All 5 FFI functions share `(handle, request_json) -> result_json` signature — maps to `callFfiJson` template method
- **D-06:** Backends create lambdas that call `callFfiJson(() -> nativeCall(handle, json), ResultType.class)` and inject into `EmbeddedSession` at construction

### Integration test strategy
- **D-07:** Two-tier testing: smoke tests on empty/default embedded instances AND data-seeded tests that verify ops produce empirically correct results
- **D-08:** Smoke tests: start embedded, call each maintenance op, verify result types are well-formed with expected fields
- **D-09:** Data-seeded tests: create collection, add records, then verify maintenance ops produce measurable results (e.g., `records_scanned > 0` after rebuild, WAL prune shows actual effect)
- **D-10:** Identical test structure in both `:jna:test` and `:panama:test` modules — same test cases, same assertions (consistent with Phase 7 D-07)

### Error handling
- **D-11:** Java-side validation for obvious bad inputs (null/empty collection name) — throws `IllegalArgumentException` before hitting FFI
- **D-12:** Option builder validation at `build()` time (already implemented in Phase 6 core types)
- **D-13:** Runtime errors (nonexistent collection, etc.) delegated to Rust FFI — returned via `LAST_ERROR` and wrapped as `ChromaException`
- **D-14:** Follows Phase 6 three-tier exception rule (D-21): `IllegalArgumentException` for bad inputs, `IllegalStateException` for closed session, `ChromaException` for FFI failures

### Claude's Discretion
- Exact functional interface types for maintenance callbacks (BiFunction vs custom)
- Whether to use a single callback type for all JSON-in/JSON-out maintenance calls or typed callbacks per operation
- Test data creation approach (Chroma client API or direct FFI calls)
- Order of implementation (JNA first vs Panama first vs parallel)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Core module contracts (Phase 6 output)
- `java/core/src/main/java/tech/amikos/chroma/local/core/AbstractChromaRuntime.java` — `callFfiJson` template method for maintenance calls
- `java/core/src/main/java/tech/amikos/chroma/local/core/EmbeddedSession.java` — Current minimal session; must expand with callback slots
- `java/core/src/main/java/tech/amikos/chroma/local/core/RebuildOptions.java` — Option builder with `toJson()` for rebuild requests
- `java/core/src/main/java/tech/amikos/chroma/local/core/RebuildCollectionResult.java` — Result POJO for rebuild
- `java/core/src/main/java/tech/amikos/chroma/local/core/CompactCollectionRequest.java` — Request type for compact single collection
- `java/core/src/main/java/tech/amikos/chroma/local/core/CompactAllRequest.java` — Request type for compact all
- `java/core/src/main/java/tech/amikos/chroma/local/core/CompactionResult.java` — Result POJO for compaction
- `java/core/src/main/java/tech/amikos/chroma/local/core/WALPruneOptions.java` — Option builder for WAL prune
- `java/core/src/main/java/tech/amikos/chroma/local/core/WALPruneResult.java` — Result POJO for WAL prune

### Backend implementations (wiring targets)
- `java/jna/src/main/java/tech/amikos/chroma/local/jna/JnaChromaRuntime.java` — JNA backend; must bind 5 new symbols and wire EmbeddedSession callbacks
- `java/panama/src/main/java/tech/amikos/chroma/local/panama/PanamaChromaRuntime.java` — Panama backend; same changes as JNA

### Rust FFI shim (symbol reference)
- `shim/src/lib.rs` — `chroma_embedded_rebuild_collection` (line 4443), `chroma_embedded_compact_collection` (line 4519), `chroma_embedded_compact_all` (line 4596), `chroma_embedded_prune_wal_collection` (line 4724), `chroma_embedded_prune_wal_all` (line 4801)

### Go reference implementation
- `internal/runtime/rebuild.go` — Go rebuild API, request JSON format
- `internal/runtime/compaction.go` — Go compaction API, request JSON format
- `internal/runtime/wal_prune.go` — Go WAL prune API, request JSON format
- `internal/runtime/embedded.go` — Go embedded struct with maintenance method wiring

### ServerSession reference (callback slot pattern)
- `java/core/src/main/java/tech/amikos/chroma/local/core/ServerSession.java` — Callback slot pattern with UnsupportedOperationException stubs for maintenance methods (Phase 8 wires these for server too, but that's Phase 10)

### Requirements
- `.planning/REQUIREMENTS.md` — EMNT-01 through EMNT-05

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `AbstractChromaRuntime.callFfiJson()` — template method that acquires lock, calls FFI, deserializes JSON result, handles errors. Direct fit for all 5 maintenance ops
- All option builders (`RebuildOptions`, `WALPruneOptions`, `CompactCollectionRequest`, `CompactAllRequest`) already have `toJson()` methods producing the request JSON
- All result POJOs (`RebuildCollectionResult`, `CompactionResult`, `WALPruneResult`) already have Gson deserialization via `JsonUtil`
- `ServerSession` callback slot pattern — proven approach to copy for `EmbeddedSession`

### Established Patterns
- JNA: `Function.getFunction(library, "symbol_name")` for symbol binding; `function.invokeLong(...)` / `function.invokePointer(...)` for calls
- Panama: `linker.downcallHandle(lookup.find("symbol").get(), descriptor)` for symbol binding; `handle.invoke(...)` for calls
- Both backends use `callFfiJson(() -> nativeCall(handle, json), Type.class)` pattern established in Phase 7

### Integration Points
- `EmbeddedSession` constructor must expand — this affects `doStartEmbedded()` in both backends
- Existing JNA/Panama embedded smoke tests coexist alongside new maintenance tests
- Makefile `test-java` target runs both `:jna:test` and `:panama:test`

</code_context>

<specifics>
## Specific Ideas

- All 5 FFI functions share identical `(handle: *mut c_void, request_json: *const c_char) -> *mut c_char` signature — a single helper pattern in each backend can wrap all 5 calls
- Data-seeded tests should verify empirically correct results (e.g., `records_scanned > 0` after rebuild, actual WAL prune effect) to ensure functional correctness beyond smoke
- `EmbeddedSession` callback expansion should be backward-compatible if possible — consider builder or overloaded constructor

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 08-embedded-maintenance*
*Context gathered: 2026-03-26*
