# Phase 6: Core Foundation Types - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-22
**Phase:** 06-core-foundation-types
**Areas discussed:** Result type design, Config builder shape, FFI safety infra, Session type hierarchy, Option builder pattern, Test strategy, Error handling contract

---

## Result type design

### Nullable field representation

| Option | Description | Selected |
|--------|-------------|----------|
| Boxed Long | Use `Long` instead of `long` — null means absent | ✓ |
| OptionalLong accessors | Store as Long internally, expose OptionalLong getters | |
| Sentinel values | Use primitive long with -1 as absent sentinel | |

**User's choice:** Boxed Long
**Notes:** Simple, no extra dependency, standard Java pattern. Matches what Gson produces naturally.

### JSON deserialization strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Manual parsing | javax.json or string parsing — zero external deps | |
| Gson dependency | com.google.code.gson:gson — clean @SerializedName annotations | ✓ |
| Jackson dependency | Industry standard but heavier footprint | |

**User's choice:** Gson dependency
**Notes:** Clean deserialization, handles Long nullability automatically. Adds runtime dep but simplifies parsing for nested types.

### Result POJO style

| Option | Description | Selected |
|--------|-------------|----------|
| Final-field classes | Traditional immutable classes with private final fields and getters | ✓ |
| Java records | Compact syntax, automatic equals/hashCode/toString | |

**User's choice:** Final-field classes
**Notes:** Works with Gson out of the box, compatible with Java 17+.

### Getter naming convention

| Option | Description | Selected |
|--------|-------------|----------|
| Accessor style | collectionId(), recordsScanned() — modern Java convention | ✓ |
| JavaBean style | getCollectionId(), getRecordsScanned() — traditional convention | |

**User's choice:** Accessor style
**Notes:** Modern, concise, matches newer Java APIs.

---

## Config builder shape

### YAML output strategy

| Option | Description | Selected |
|--------|-------------|----------|
| String formatting | StringBuilder producing YAML directly — mirrors Go | |
| SnakeYAML dependency | Proper YAML serialization via library | ✓ |

**User's choice:** SnakeYAML dependency
**Notes:** Handles edge cases (special chars, multiline) correctly.

### Validation level

| Option | Description | Selected |
|--------|-------------|----------|
| Structural only | Validate types and always-wrong constraints only | |
| Permissive | No validation — let FFI call fail | |
| Strict | Validate everything possible at build time | ✓ |

**User's choice:** Strict
**Notes:** Full upfront validation including port range, path checks, address format, mutually exclusive options.

### Raw YAML escape hatch

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, with override | rawYaml(String) on both builders — overrides all fields | ✓ |
| No escape hatch | Force config through builder fields | |
| Separate factory method | ServerConfig.fromYaml(String) as static factory | |

**User's choice:** Yes, with override
**Notes:** Mirrors Go's WithRawYAML behavior exactly.

### Builder inheritance

| Option | Description | Selected |
|--------|-------------|----------|
| Independent builders | Each builder self-contained, no shared base | ✓ |
| Shared abstract base | AbstractConfigBuilder with common fields | |

**User's choice:** Independent builders
**Notes:** Shared fields are minimal (3-4), not worth the inheritance complexity.

---

## FFI safety infra

### Lock location

| Option | Description | Selected |
|--------|-------------|----------|
| Abstract base in core | AbstractChromaRuntime holds ReentrantLock, backends extend | ✓ |
| Utility class in core | Static FfiLock utility class | |
| Each backend owns lock | No shared infrastructure | |

**User's choice:** Abstract base in core

### Lock scope

| Option | Description | Selected |
|--------|-------------|----------|
| Global static lock | Single static lock for all instances — mirrors Go's ffiMu | ✓ |
| Per-instance lock | Each runtime gets its own lock | |

**User's choice:** Global static lock
**Notes:** User asked about multi-instance capability. Analysis showed Rust LAST_ERROR is `static Mutex<Option<String>>` (global, not thread-local). Go uses global `ffiMu`. User agreed with global lock but noted desire for future per-handle error isolation to enable concurrent instances on different directories.

### String ownership helpers

| Option | Description | Selected |
|--------|-------------|----------|
| Abstract methods on base | readBorrowedString() and readOwnedString() abstract in base class | ✓ |
| Callback-based pattern | StringReader functional interface | |

**User's choice:** Abstract methods on base class

---

## Session type hierarchy

### Shared interface for maintenance

| Option | Description | Selected |
|--------|-------------|----------|
| No shared interface | Independent types, matching Go's separate structs | ✓ |
| Shared MaintenanceCapable | Interface with rebuild/compact/prune/backup methods | |
| Shared AutoCloseable base | Only lifecycle contract shared | |

**User's choice:** No shared interface
**Notes:** Semantics differ (embedded runs directly, server does stop-embed-op-restart). Callers always know which mode.

### ChromaRuntime.startServer() addition

| Option | Description | Selected |
|--------|-------------|----------|
| Add to ChromaRuntime | startServer(String configYaml) returning ServerSession | ✓ |
| Overloaded with builder | Both raw YAML and config object overloads | |
| Builder-only | Only config object, raw YAML through escape hatch | |

**User's choice:** Add to ChromaRuntime interface

### ServerSession implementation

| Option | Description | Selected |
|--------|-------------|----------|
| Concrete class in core | Final class wrapping handle + callbacks, like EmbeddedSession | ✓ |
| Interface in core | Backends implement their own | |

**User's choice:** Concrete class in core

### Phase 6 scope depth

| Option | Description | Selected |
|--------|-------------|----------|
| Full definition upfront | All callback slots defined now, backends wire in Phases 7-10 | ✓ |
| Skeleton now, grow later | Handle + close only, add methods per phase | |

**User's choice:** Full definition upfront

---

## Option builder pattern

### Option type structure

| Option | Description | Selected |
|--------|-------------|----------|
| Builder per option type | Nested Builder with fluent API and strict validation | ✓ |
| Immutable POJOs with factories | Static factory methods, constructor for full customization | |
| Mixed approach | Builders for complex, constructors for simple | |

**User's choice:** Builder per option type

### Serialization responsibility

| Option | Description | Selected |
|--------|-------------|----------|
| Core produces JSON | Option types have toJson() using Gson | ✓ |
| Backends serialize | Each backend converts to JSON independently | |

**User's choice:** Core produces JSON

---

## Test strategy

### Golden YAML test format

| Option | Description | Selected |
|--------|-------------|----------|
| Inline expected strings | Multi-line strings in test methods | ✓ |
| Fixture files | src/test/resources/*.yaml files | |

**User's choice:** Inline expected strings

### Result POJO test data

| Option | Description | Selected |
|--------|-------------|----------|
| Hand-crafted JSON | Representative JSON strings in tests | ✓ |
| Captured from Go tests | Actual JSON output from Go test runs | |
| Both | Hand-crafted + cross-language fixtures | |

**User's choice:** Hand-crafted JSON

---

## Error handling contract

### lastError() integration

| Option | Description | Selected |
|--------|-------------|----------|
| Integrated in base class | callFfi() template: lock → call → check → readError → unlock | ✓ |
| Lock and error separate | withFfiLock() + lastError() composed by backends | |

**User's choice:** Integrated in base class

### Exception hierarchy

| Option | Description | Selected |
|--------|-------------|----------|
| Three-tier | IAE for bad input, ISE for lifecycle, CE for native failures | ✓ |
| ChromaException only | All errors wrapped in ChromaException | |

**User's choice:** Three-tier

---

## Claude's Discretion

- Exact Gson configuration (custom TypeAdapter vs annotation-based)
- Internal structure of callFfi() overloads
- Test class organization within core module
- SnakeYAML Dumper options

## Deferred Ideas

- Per-handle error isolation for concurrent multi-instance support (user explicitly requested as future phase)
- ChromaErrorCode enum on ChromaException (tracked as FUTURE-01)
