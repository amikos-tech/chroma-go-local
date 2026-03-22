# Coding Conventions

**Analysis Date:** 2026-03-19

## Naming Patterns

**Files:**
- Go: lowercase with underscores for multi-word names (`chroma_test.go`, `embedded_test.go`)
- Rust: lowercase with underscores (`lib.rs`)
- Java: PascalCase for classes (`EmbeddedSession.java`, `PanamaChromaRuntime.java`)

**Functions:**
- Go: PascalCase for exported, camelCase for unexported
  - Example: `Init()`, `StartServer()`, `NewEmbedded()` (exported)
  - Example: `registerFunctions()`, `callFFIHandle()`, `goStringFromPtr()` (unexported)
- Rust: snake_case for all functions
  - Example: `parse_database_name()`, `resolve_database_name()`, `set_last_error()`
- Java: camelCase for methods, PascalCase for classes
  - Example: `startEmbedded()`, `version()`, `close()` in classes like `PanamaChromaRuntime`

**Variables:**
- Go: camelCase for local variables and struct fields
  - Example: `libHandle`, `libErr`, `persistPath`, `serverConfig`
- Rust: snake_case for local variables
  - Example: `panic_message`, `default_tenant`, `database_name`
- Java: camelCase for fields and local variables
  - Example: `handle`, `closeAction`, `libPath`

**Types:**
- Go: PascalCase for struct names, interfaces, and type definitions
  - Example: `Server`, `Embedded`, `StartServerConfig`, `ServerOption`
  - Receiver names: short (1-2 chars) like `s *Server`, `e *Embedded`
- Rust: PascalCase for structs, enums, type aliases
  - Example error constants: UPPERCASE with underscores (`SUCCESS`, `ERROR_NULL_INPUT`)
- Java: PascalCase for classes, interfaces, enums
  - Example: `EmbeddedSession`, `ChromaRuntime`, `ChromaException`

**Constants:**
- Go: UPPERCASE in error code blocks, but also use regular camelCase for configuration constants
  - Example: `DefaultTenantID`, `DefaultDatabase`, `DefaultEmbeddedDir`, `maxCStringLen`
- Rust: UPPERCASE with underscores for all constants
  - Example: `DEFAULT_TENANT`, `DEFAULT_DATABASE`, `DEFAULT_QUERY_RESULTS`, `DELETE_RECORDS_LIMIT_REQUIRES_FILTER_ERR`

## Code Style

**Formatting:**
- Go: Uses standard `gofmt` for formatting
  - Command: `gofmt -w .`
  - Also uses `goimports` for import organization
  - Command: `goimports -w .`
- Rust: Uses `cargo fmt`
  - Command: `cd shim && cargo fmt`
- Java: Configured through Gradle build system
  - Standard Java formatting conventions (4-space indent)

**Linting:**
- Go: `golangci-lint` with configuration in `.golangci.yml`
  - Key linters enabled: dupword, ginkgolinter, gocritic, mirror
  - Custom checks: ST1000, ST1001, ST1003 for staticcheck
  - Enables dot-import for fmt package only
  - Exclusions for generated code, examples, third_party directories
- Rust: `cargo clippy` with warnings as errors
  - Command: `cd shim && cargo clippy --locked -- -D warnings`
  - Treats all warnings as errors for strict code quality
- Java: Gradle-based checks
  - Command: `gradle --no-daemon :core:check :jna:check :panama:check`

## Import Organization

**Go Order:**
1. Standard library imports (`fmt`, `strings`, `sync`, etc.)
2. Third-party imports (`github.com/...`)
3. Internal imports (none currently used)

Example from `chroma.go`:
```go
import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/pkg/errors"
)
```

**Rust Order:**
- `use std::*` imports for standard library
- External crate imports (figment, serde, tokio, chroma-*)
- Grouped logically by functionality

Example from `lib.rs`:
```rust
use std::any::Any;
use std::collections::HashMap;
use std::ffi::{c_char, c_void, CStr, CString};
// ... more std imports
use chroma_config::registry::Registry;
use chroma_frontend::config::FrontendServerConfig;
// ... other chroma imports
```

**Java Order:**
- `java.*` imports first
- `javax.*` imports
- `org.*` imports
- `static` imports for assertions and utilities
- Organized within blocks by package

## Error Handling

**Go Patterns:**
- Wrapped errors using `github.com/pkg/errors` for context
- Explicit error return as second value in signature
- Early return on error pattern
- Error codes mapped to specific error types (See `errors.go`)
- Example from `chroma.go`:
  ```go
  func callFFIHandle(call func() uintptr) (uintptr, error) {
      ffiMu.Lock()
      defer ffiMu.Unlock()
      handle := call()
      if handle == 0 {
          return 0, nullPointerError(getLastErrorUnlocked())
      }
      return handle, nil
  }
  ```

**Rust Patterns:**
- FFI boundary panic catching with `panic::catch_unwind()`
- Error storage in static `LAST_ERROR` mutex
- Error message sanitization (null byte replacement)
- Macro guards (`ffi_guard_ptr_mut!`, `ffi_guard_ptr_const!`, `ffi_guard_code!`) for panic boundaries
- Example from `lib.rs`:
  ```rust
  fn set_last_error(msg: &str) {
      let sanitized = msg.replace('\0', "\\0");
      match LAST_ERROR.lock() {
          Ok(mut slot) => {
              *slot = Some(sanitized);
          }
          Err(poisoned) => {
              let mut slot = poisoned.into_inner();
              *slot = Some(sanitized);
          }
      }
  }
  ```

**Java Patterns:**
- Throw exceptions for precondition validation (IllegalArgumentException, IllegalStateException)
- Automatic resource management with try-with-resources and AutoCloseable
- Static assertion imports from JUnit 5 (org.junit.jupiter.api.Assertions)
- Example from `EmbeddedSession.java`:
  ```java
  public EmbeddedSession(long handle, LongConsumer closeAction) {
      if (handle == 0L) {
          throw new IllegalArgumentException("embedded handle must be non-zero");
      }
      if (closeAction == null) {
          throw new IllegalArgumentException("closeAction must be set");
      }
      // ...
  }
  ```

## Logging

**Framework:** Standard library logging
- Go: Uses standard `fmt` package and test `t.Log()` for test output
- Rust: No logging library in shim; errors stored in `LAST_ERROR` static
- Java: No logging in core/test code; relies on assertions and exception messages

**Patterns:**
- Go: `t.Logf()` for informational test output, `t.Fatalf()` for test failures
  - Example: `t.Logf("Chroma shim version: %s", version)`
- Rust: Error context captured in `LAST_ERROR` accessible via `chroma_get_last_error()`
- Java: System.getenv() for environment inspection in tests

## Comments

**When to Comment:**
- Document public API functions with comment blocks above function definition
- Comment complex FFI operations and unsafe code blocks
- Explain non-obvious business logic or workarounds

**Doc Comments:**
- Go: Comment blocks directly above declarations
  - Example from `chroma.go`:
    ```go
    // Init initializes the Chroma library. Must be called before any other functions.
    // If libPath is empty, it will look for CHROMA_LIB_PATH environment variable.
    func Init(libPath string) error {
    ```
- Rust: Doc comments using `///` for public items
  - Not heavily used; mostly inline comments for FFI boundaries
- Java: JavaDoc comments for public classes and methods
  - Example: `/** ... */` pattern (sparse usage in test code)

**Comment Style:**
- Begin with capital letter
- End with period
- One sentence per comment (or few related sentences)
- No trivial/obvious comments

## Function Design

**Size:**
- Go: Functions stay under 50-100 lines typically
- Rust: FFI functions may be longer (30-150 lines) due to error handling ceremony
- Java: Constructors and public methods kept concise (under 30 lines)

**Parameters:**
- Go: Receiver first (for methods), then parameters left to right
  - Use `*ConfigType` for builder patterns with options
  - Use `Request` types for complex multi-parameter operations
  - Example: `func (e *Embedded) CreateCollection(req EmbeddedCreateCollectionRequest) (*EmbeddedCollection, error)`
- Rust: Parameters ordered logically, self first for methods
  - C FFI functions use raw pointers and size_t for arrays
- Java: Use builder patterns or constructor injection
  - Minimal parameters (prefer dependency injection)

**Return Values:**
- Go: Explicit error as second return value
  - Panic only in initialization or fatal conditions
  - Example: `func Foo() (Result, error)`
- Rust: FFI returns error codes (i32) with output via pointer parameters
  - Inner Rust code uses Result<T, E>
- Java: Throws exceptions for errors, returns values normally
  - Use Optional for nullable returns (not prevalent in current code)

## Module Design

**Exports:**
- Go: Public functions start with uppercase letter, unexported with lowercase
  - Public types: `Server`, `Embedded`, `StartServerConfig`
  - Public functions: `Init()`, `StartServer()`, `NewEmbedded()`
- Rust: `pub` keyword for FFI-exported items, private by default
  - All FFI functions start with `chroma_` prefix
- Java: Public classes/methods for runtime initialization
  - Package-private classes for internal state management

**Barrel Files:**
- Not used in this codebase
- Go package root exports all public types directly from their files

## FFI-Specific Patterns

**Go FFI Bridge (purego):**
- Global variable declarations for all FFI functions at package level
- Registration pattern in `registerFunctions()` with struct of name/target pairs
- Defer unlock pattern for FFI calls: `defer ffiMu.Unlock()` after lock
- C string conversions: `cStringFromGo()` for Go→C, `goStringFromPtr()` for C→Go
- Handle validation: Check for zero/nil returns and call `getLastErrorUnlocked()`

**Rust FFI Boundaries:**
- All public FFI functions use `#[no_mangle]` extern "C"
- Panic catching with `with_ffi_panic_boundary()` helper
- Macro guards: `ffi_guard_ptr_mut!`, `ffi_guard_ptr_const!`, `ffi_guard_code!`
- CString/CStr conversions for string boundaries
- Static LAST_ERROR for error message passing back to Go

**Java FFI Bridges:**
- Two implementations: JNA (fallback) and Panama (primary for Java 22+)
- PanamaChromaRuntime and JnaChromaRuntime both implement same interface
- Constructor validates preconditions (non-null, non-zero handles)
- Resource management via AutoCloseable and try-with-resources
- EmbeddedSession wraps handle with close-once semantics using AtomicBoolean

## Configuration Patterns

**Builder Pattern with Options:**
- Go: `ServerOption` and `EmbeddedOption` function types
  - `func DefaultServerConfig() *ServerConfig` returns default
  - `WithPort(port int) ServerOption` returns option function
  - Example usage: `NewServer(WithPort(8000), WithPersistPath("./data"))`
- Configuration struct has unexported `rawYAML` field for override
- YAML generation via `toYAML()` method on config structs

**Safety Patterns:**
- Struct fields unexported (lowercase) with constructor functions
- Mutex protection for concurrent access (see `Server.stateMu`, `Embedded.stateMu`)
- Atomic operations for handle swaps: `atomic.SwapUintptr()`
- Finalizers for cleanup: `runtime.SetFinalizer(server, func(s *Server) { _ = s.Close() })`

---

*Convention analysis: 2026-03-19*
