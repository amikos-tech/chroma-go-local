---
status: complete
phase: 06-core-foundation-types
source: [06-01-SUMMARY.md, 06-02-SUMMARY.md, 06-03-SUMMARY.md]
started: 2026-03-26T10:00:00Z
updated: 2026-03-26T10:15:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Core module builds with zero errors
expected: `gradle :core:build` completes successfully — all compilation and tests pass with exit code 0.
result: pass

### 2. Zero JNA or Panama imports in core module
expected: Searching core/src/main/java for any import referencing `com.sun.jna` or `java.lang.foreign` returns zero matches. Core module has no FFI dependencies.
result: pass

### 3. Result POJOs deserialize from snake_case JSON
expected: RebuildCollectionResult, CompactionResult, WALPruneResult, and BackupManifest can be constructed from snake_case JSON strings via JsonUtil. Optional fields (boxed Long) are null when absent from JSON.
result: pass

### 4. ServerConfigBuilder produces valid YAML
expected: `ServerConfigBuilder.builder().port(8000).persistPath("/tmp/data").build().toYaml()` produces YAML with all required fields (chroma_server_auth_provider, is_persistent, persist_directory, chroma_server_http_port, chroma_otel_collection_endpoint, anonymized_telemetry). Golden tests compare output semantically against known-good YAML.
result: pass

### 5. EmbeddedConfigBuilder produces valid YAML
expected: `EmbeddedConfigBuilder.builder().persistPath("/tmp/data").build().toYaml()` produces YAML with persist_directory, allow_reset, and anonymized_telemetry fields matching Go's DefaultEmbeddedConfig() format.
result: pass

### 6. Builder validation rejects invalid inputs
expected: Calling `RebuildOptions.builder().build()` without a required name throws IllegalStateException. `WALPruneOptions.watermark()` with negative values throws. `BackupOptions.builder().build()` without destinationPath throws. Builders enforce invariants at build time.
result: pass

### 7. ServerSession close is idempotent
expected: Calling `close()` on a ServerSession twice does not throw — second call is a no-op. After close, calling `port()`, `address()`, or maintenance methods throws IllegalStateException.
result: pass

### 8. AbstractChromaRuntime FFI lock serialization
expected: FFI template methods (callFfiHandle, callFfiJson, callFfiVoid) acquire a global ReentrantLock before invoking the FFI callback and release it afterward. Concurrent calls are serialized. If FFI returns an error, a ChromaException is thrown after the lock is released.
result: pass

## Summary

total: 8
passed: 8
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none]
