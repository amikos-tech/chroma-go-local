# Testing Patterns

**Analysis Date:** 2026-03-19

## Test Framework

**Go:**
- Runner: `testing` (standard library)
- Assertion Library: `github.com/stretchr/testify/require`
- Config: None (standard testing conventions)
- Run Commands:
  ```bash
  make test              # Run Go tests (requires debug build)
  make test-release      # Run Go tests with release build
  CHROMA_LIB_PATH=... go test -v ./...     # Run with custom lib path
  go test -run '^$' -bench . -benchmem ./  # Run benchmarks
  ```

**Rust:**
- Runner: `cargo test`
- Assertion Library: `assert!`, `assert_eq!` (standard library)
- Config: Criterion for benchmarks
- Run Commands:
  ```bash
  make test-rust         # Run Rust tests
  cd shim && cargo test --locked
  make bench-rust        # Run Criterion benchmarks
  cd shim && cargo bench --locked --bench ffi_bench
  ```

**Java:**
- Runner: JUnit Jupiter 5 (org.junit.jupiter:junit-jupiter)
- Assertion Library: org.junit.jupiter.api.Assertions
- Config: Gradle-based
- Run Commands:
  ```bash
  make test-java         # Run JNA and Panama tests
  gradle --no-daemon :jna:test :panama:test
  make lint-java         # Run checks
  gradle --no-daemon :core:check :jna:check :panama:check
  ```

## Test File Organization

**Location:**
- Go: Co-located with implementation files (same package)
  - Test files in root package directory (e.g., `chroma_test.go` next to `chroma.go`)
  - Example: `chroma_test.go`, `embedded_test.go`, `library_test.go`, `embedded_integration_edge_test.go`
- Rust: Inline in source with `#[cfg(test)]` and separate `benches/` directory
  - Tests in `shim/src/`
  - Benchmarks in `shim/benches/ffi_bench.rs`
- Java: Mirror directory structure
  - Source: `java/core/src/main/java/tech/amikos/chroma/local/core/`
  - Tests: `java/core/src/test/java/tech/amikos/chroma/local/core/`
  - JNA tests: `java/jna/src/test/java/tech/amikos/chroma/local/jna/`
  - Panama tests: `java/panama/src/test/java/tech/amikos/chroma/local/panama/`

**Naming:**
- Go: `*_test.go` suffix (e.g., `chroma_test.go`, `embedded_test.go`)
  - Specialized names for test categories: `embedded_integration_edge_test.go`, `embedded_benchmark_test.go`
- Rust: Test functions use `#[test]` attribute, benches use separate files
  - Naming: `test_*` convention inside test modules
- Java: `Test` suffix (e.g., `EmbeddedSessionTest.java`, `PanamaChromaRuntimeTest.java`)
  - Both JNA and Panama have separate test classes

**Structure:**
```
Go:
  chroma/
  ├── chroma.go                                  # Core API
  ├── chroma_test.go                             # Basic tests
  ├── embedded.go                                # Embedded mode
  ├── embedded_test.go                           # Embedded tests
  ├── embedded_integration_edge_test.go          # Integration tests
  ├── library_test.go                            # Library loading tests
  └── errors.go

Rust:
  shim/
  ├── src/
  │   └── lib.rs                                 # FFI shim
  ├── benches/
  │   └── ffi_bench.rs                          # Criterion benchmarks
  └── tests/                                     # Integration tests (if present)

Java:
  java/
  ├── core/
  │   ├── src/main/java/...EmbeddedSession.java
  │   └── src/test/java/...EmbeddedSessionTest.java
  ├── jna/
  │   └── src/test/java/...JnaChromaRuntimeTest.java
  └── panama/
      └── src/test/java/...PanamaChromaRuntimeTest.java
```

## Test Structure

**Go Suite Organization:**

Standard Go testing pattern with helper functions:

```go
func TestInitAndVersion(t *testing.T) {
	if err := Init(""); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	version := Version()
	if version == "" {
		t.Fatal("Version returned empty string")
	}
	require.NoError(t, err)
	require.Equal(t, version, versionWithError)
}
```

**Helper Functions:**
- Prefix with test function name context
- Use `t.Helper()` to exclude from stack trace
- Example from `embedded_integration_edge_test.go`:
  ```go
  func newEmbeddedForIntegrationTest(t *testing.T) *Embedded {
      t.Helper()
      if err := Init(""); err != nil {
          t.Fatalf("Failed to initialize: %v", err)
      }
      embedded, err := NewEmbedded(...)
      t.Cleanup(func() {
          if closeErr := embedded.Close(); closeErr != nil {
              t.Errorf("failed to close embedded runtime: %v", closeErr)
          }
      })
      return embedded
  }
  ```

**Setup/Teardown:**
- `t.Cleanup()` for resource cleanup (registered at test start)
- `defer` blocks for immediate cleanup in some tests
- Temporal isolation via `t.TempDir()` for file I/O

**Patterns:**
- `require.NoError(t, err)` - Assert nil error
- `require.ErrorIs(t, err, ErrLibraryNotLoaded)` - Assert specific error type
- `require.Eventually(t, condition, timeout, interval, message)` - Polling/retry pattern
- `require.Contains(t, string, substring)` - String assertion
- Manual assertions with `if ... { t.Errorf() }` pattern

**Rust Test Pattern:**

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_name() {
        // Test code
        assert_eq!(expected, actual);
    }
}
```

**Java Test Pattern:**

```java
class EmbeddedSessionTest {
    @Test
    void testName() {
        // Test code
        assertEquals(expected, actual);
        assertThrows(ExceptionType.class, () -> { /* code */ });
    }
}
```

## Mocking

**Go Framework:**
- Standard library `net/http` mocking via `httptest` package
- No dedicated mocking library in current tests
- Testing patterns use real implementations or test doubles

**Example HTTP Testing:**
```go
// From chroma_test.go - polling actual server
require.Eventually(t, func() bool {
    resp, err := http.Get("http://127.0.0.1:8765/api/v2/heartbeat")
    if err != nil {
        return false
    }
    _, _ = io.Copy(io.Discard, resp.Body)
    _ = resp.Body.Close()
    return resp.StatusCode == http.StatusOK
}, 10*time.Second, 100*time.Millisecond, "server heartbeat did not become ready")
```

**What to Mock:**
- File system operations (use `t.TempDir()` for temp isolation)
- External HTTP endpoints (manual polling with eventual consistency checks)
- FFI library loading (minimal mocking; mostly full integration tests)

**What NOT to Mock:**
- Core Chroma operations (test real embedded runtime)
- Collection operations (full integration via embedded mode)
- Database state (test actual persistence)

**Rust:**
- No external mocking libraries used
- Tests use real Tokio runtime
- Panic catching tests use standard unwinding

**Java:**
- No explicit mocking library (no Mockito, etc.)
- Tests use real implementations or constructor injection of collaborators
- `@TempDir` annotation for temporary directory isolation (JUnit 5)
- Assumptions for environment-dependent tests: `Assumptions.assumeTrue(libPath != null)`

## Fixtures and Factories

**Test Data (Go):**

Configuration fixtures:
```go
config := `
port: 8765
listen_address: "127.0.0.1"
persist_path: "./chroma_test_data"
allow_reset: true
`
```

Builder factory pattern:
```go
server, err := StartServer(StartServerConfig{ConfigString: config})
```

Embedded test factory:
```go
embedded, err := NewEmbedded(
    WithEmbeddedPersistPath(t.TempDir()),
    WithEmbeddedAllowReset(true),
)
```

**Location:**
- Fixtures embedded in test functions
- No separate fixture directory
- Timestamps used for unique resource names: `fmt.Sprintf("collection_%d", time.Now().UnixNano())`

**Test Data Patterns:**
- Unique names per test run via time-based suffixes
- Temporary directories via `t.TempDir()`
- YAML configuration strings inline
- Reusable patterns extracted to helper functions

**Java:**
```java
// From PanamaChromaRuntimeTest.java
private static String embeddedYaml(Path persistDir) {
    String escapedPath = persistDir.toAbsolutePath().toString().replace("\\", "\\\\");
    return "persist_path: \"" + escapedPath + "\"\n"
            + "sqlite_filename: \"chroma.sqlite3\"\n"
            + "allow_reset: true\n";
}
```

## Coverage

**Requirements:**
- Not explicitly enforced
- No codecov/coverage threshold checks observed in CI/build files

**View Coverage:**
```bash
# Go coverage (manual)
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Current Coverage Status:**
- Core functionality well tested (chroma.go, embedded.go have extensive tests)
- Integration tests exercise real runtime
- Edge cases covered in `embedded_integration_edge_test.go`
- FFI boundary testing in `library_test.go`

## Test Types

**Unit Tests (Go):**
- Scope: Single function/method in isolation
- Approach: Direct function calls with asserting results
- Example: `TestVersionWithErrorWhenLibraryNotLoaded` - tests error path when lib not loaded
- Pattern: Setup preconditions (e.g., clear libHandle), call function, assert output

**Integration Tests (Go):**
- Scope: Multi-function workflows with real embedded runtime
- Approach: Start actual Chroma embedded instance, perform operations, assert state
- Example: `TestEmbeddedModeBasicFlow` - full lifecycle (create tenant, database, collection)
- Pattern: Setup via `newEmbeddedForIntegrationTest()`, exercise API, cleanup with `t.Cleanup()`

**Edge Case/Convergence Tests (Go):**
- Scope: Async operations and eventual consistency
- Approach: Poll with timeout waiting for desired state
- Example: `waitForCollectionConvergence()` helper with `require.Eventually()`
- Pattern: Custom predicate functions, retryable error detection, timeout assertions

**Smoke Tests (Java):**
- Scope: Basic initialization and lifecycle
- Approach: Verify startup works and no unhandled exceptions
- Example: `versionAndEmbeddedLifecycleSmokeTest()` - just ensure no errors
- Pattern: Try-with-resources for AutoCloseable, verify version != null

**Validation Tests (Java):**
- Scope: Constructor argument validation and state machine enforcement
- Approach: Assert exceptions thrown for invalid inputs
- Example: `closeInvokesActionOnce()`, `initRejectsNullLibraryPath()`
- Pattern: `assertThrows(ExceptionType.class, () -> { code })`

## Async Testing

**Go Pattern:**

Polling/retry with `require.Eventually()`:
```go
require.Eventually(t, func() bool {
    resp, err := http.Get(url)
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    return resp.StatusCode == http.StatusOK
}, timeout, pollInterval, message)
```

Used for:
- Waiting for server startup
- Collection convergence after updates
- Index completion

**Rust Pattern:**
- Uses Tokio async runtime for tests
- No explicit async test pattern in current code (FFI is synchronous wrapper)

**Java Pattern:**
- No explicit async testing pattern
- @TempDir provides isolated filesystem
- Assumptions allow skipping when CHROMA_LIB_PATH not set

## Error Testing

**Go Pattern:**

Specific error type assertion:
```go
version, err := VersionWithError()
require.ErrorIs(t, err, ErrLibraryNotLoaded)
require.Empty(t, version)
```

Error message content assertion:
```go
_, err := StartServer(config)
require.Error(t, err)
require.Contains(t, strings.ToLower(err.Error()), "startup")
```

Error wrapping verification:
```go
expectedErr := "expected error message"
require.ErrorContains(t, err, expectedErr)
```

**Rust Pattern:**
- Panic boundary testing via `catch_unwind(AssertUnwindSafe(...))`
- Checked in FFI tests that panics are properly converted to error codes

**Java Pattern:**

Exception type assertion:
```java
assertThrows(IllegalArgumentException.class, () ->
    new EmbeddedSession(0L, ignored -> {})
);
```

Exception on state violation:
```java
session.close();
assertThrows(IllegalStateException.class, session::handle);
```

## Test Execution Strategy

**Local Development:**
```bash
make test          # Go tests only (debug build)
make test-release  # Go tests with release build
make test-all      # Go + Rust + Java (if Gradle available)
make lint          # All linters (Go + Rust)
make bench         # All benchmarks (Go + Rust)
```

**Windows Development:**
```powershell
pwsh -File .\scripts\dev-windows.ps1 -Task test
pwsh -File .\scripts\dev-windows.ps1 -Task lint
```

**Test Ordering:**
1. Unit tests (fast, isolated)
2. Integration tests (slower, real runtime)
3. Benchmarks (optional, for performance verification)

**Dependency Chain:**
- All Go tests require: `make build-debug` (builds Rust shim first)
- All Java tests require: `make build-debug` (tests need lib to load)
- Rust tests are independent

## Common Test Utilities

**Go:**

Wait for convergence helper:
```go
func waitForCollectionConvergence(
    t *testing.T,
    embedded *Embedded,
    request EmbeddedGetCollectionRequest,
    description string,
    predicate func(*EmbeddedCollection) (bool, error),
)
```

Embedded test factory:
```go
func newEmbeddedForIntegrationTest(t *testing.T) *Embedded
```

Retry error detection:
```go
func isRetriableGetCollectionError(err error) bool {
    return strings.Contains(strings.ToLower(err.Error()), "not found")
}
```

Read YAML fixture helper:
```go
func readShimCargoVersion(t *testing.T) string
```

**Java:**

YAML generation helper:
```java
private static String embeddedYaml(Path persistDir) {
    String escapedPath = persistDir.toAbsolutePath().toString().replace("\\", "\\\\");
    return "persist_path: \"" + escapedPath + "\"\n" + ...;
}
```

Environment assumption (skip if CHROMA_LIB_PATH missing):
```java
String libPath = System.getenv("CHROMA_LIB_PATH");
Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH required");
```

## Test Naming Convention

**Go:**
- `Test<Feature>` for basic unit tests
- `Test<Feature><EdgeCase>` for edge/error cases
- `TestEmbedded<Feature>` for embedded-specific tests
- `TestEmbedded<Feature>EdgeCases` for integration edge cases

Examples:
- `TestInitAndVersion` - basic init test
- `TestVersionWithErrorWhenLibraryNotLoaded` - error path
- `TestEmbeddedModeBasicFlow` - full embedded lifecycle
- `TestEmbeddedWhereValidationEdgeCases` - complex integration case

**Java:**
- `test<FeatureName>` for methods
- Descriptive names matching test purpose

Examples:
- `testInitAndVersion()` → `versionAndEmbeddedLifecycleSmokeTest()`
- `testConstructorRejectsZeroHandle()` → `constructorRejectsZeroHandle()`

## Test Parallelization

**Go:**
- Tests run in parallel by default (standard behavior)
- Can disable with `t.Parallel()` or environment variables if needed
- Isolation via `t.TempDir()` prevents conflicts

**Rust:**
- Cargo runs tests sequentially by default
- Can parallelize with `cargo test -- --test-threads=N`

**Java:**
- Gradle runs tests in parallel by default (configurable)
- JUnit 5 uses default parallelization settings

---

*Testing analysis: 2026-03-19*
