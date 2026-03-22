# Phase 2: File Migration - Research

**Researched:** 2026-03-20
**Domain:** Go package restructuring, file migration, build tags, race detector
**Confidence:** HIGH

## Summary

Phase 2 moves all Go implementation files from the repo root into `internal/runtime/` and `internal/library/` packages. The skeleton packages from Phase 1 already exist with correct `package` declarations. The migration involves changing `package chroma` to `package runtime` or `package library`, exporting `loadLibrary` as `LoadLibrary`, updating import paths, and fixing relative file paths in tests that assumed the working directory was the repo root.

The key technical challenge is maintaining compilation at each commit step (the two-commit strategy defined in CONTEXT.md) while ensuring all 135 tests (127 Test* + 4 Benchmark* + auxiliary) continue to pass. The `readShimCargoVersion` helper in `chroma_test.go` reads `shim/Cargo.toml` via a relative path, which breaks when tests move to `internal/runtime/` because `go test` sets the working directory to the package source directory.

**Primary recommendation:** Follow the two-commit migration strategy exactly as specified in CONTEXT.md, with careful attention to relative path fixups in test files and the `loadLibrary` -> `LoadLibrary` export.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- `internal/library` exports exactly one symbol: `LoadLibrary(path string) (uintptr, error)`
- All path resolution helpers, types (`libraryCandidate`, `libraryLoadPlan`, `candidateSet`), and utility functions remain unexported
- `internal/runtime` calls `library.LoadLibrary()` from `Init()`, stores the returned handle, then registers FFI functions
- Dependency direction: runtime -> library (library is a leaf package with zero upstream imports)
- File-to-package mapping: `internal/library/` gets library.go, library_unix.go, library_windows.go, library_test.go; `internal/runtime/` gets everything else
- Maintenance ops (backup, rebuild, compaction, wal_prune) stay in runtime -- methods on Server/Embedded receiver types must be in the same package
- Tests become `package library` and `package runtime` respectively (white-box, same-package access preserved)
- No tests remain at root after this phase
- `internal_test.go` (Phase 1 anchor test) removed
- Two commits, each independently compilable:
  1. Move library package files to internal/library/
  2. Move runtime package files to internal/runtime/, add `import library` in runtime, remove internal_test.go
- Keep existing Init() + sync.Once + package-level globals pattern exactly as-is, just relocated
- `Init()` calls `library.LoadLibrary(libPath)` instead of `loadLibrary()`
- All 40+ FFI function pointers, libHandle, libOnce, libErr, ffiMu remain as package-level vars in internal/runtime

### Claude's Discretion
- Exact ordering of files within each commit (as long as two-commit structure is preserved)
- Whether to rename `loadLibrary` -> `LoadLibrary` in library_unix.go/library_windows.go or create a new exported wrapper
- Minor import path adjustments needed after the package rename

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LAYOUT-03 | All FFI globals and `sync.Once` initialization moved atomically to implementation package (no split state) | All 40+ FFI function pointers, libHandle, libOnce, libErr, ffiMu are declared in chroma.go lines 15-67; they all move together into internal/runtime in commit 2. The Init() function and registerFunctions() move alongside them. No globals remain at root. |
| LAYOUT-04 | Platform-specific files retain correct build tags after move | library_unix.go has `//go:build !windows` (line 1), library_windows.go has `//go:build windows` (line 1). These tags are file-level and transfer intact when files move. Verification via `go list -tags '' -f '{{.GoFiles}}'` and cross-compile checks. |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go toolchain | 1.26.1 (local), go.mod specifies 1.21 | Build, test, race detector | Project's Go version |
| purego | v0.9.1 | Pure Go FFI (no cgo) | Already in go.mod, used by library_unix.go |
| golang.org/x/sys | v0.6.0 | Windows syscall (LoadLibrary) | Already in go.mod, used by library_windows.go |
| github.com/pkg/errors | v0.9.1 | Error wrapping | Already in go.mod, used throughout |

### Supporting (test only)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| testify | v1.11.1 | Test assertions | Already in go.mod, used in all test files |
| gopter | v0.2.11 | Property-based testing | Already in go.mod, used in embedded_property_test.go |

No new dependencies are needed for this phase. The migration is purely structural.

## Architecture Patterns

### Target Project Structure (after Phase 2)
```
.
+-- internal/
|   +-- library/
|   |   +-- library.go          # Path resolution logic, LoadLibrary export
|   |   +-- library_unix.go     # //go:build !windows - purego.Dlopen
|   |   +-- library_windows.go  # //go:build windows - windows.LoadLibrary
|   |   +-- library_test.go     # package library (white-box)
|   +-- runtime/
|       +-- chroma.go           # FFI globals, Init(), Server type, callFFI helpers
|       +-- config.go           # ServerConfig, builder pattern, NewServer
|       +-- embedded.go         # Embedded type, all embedded methods
|       +-- errors.go           # Error codes, sentinel errors, errorFromCode
|       +-- backup.go           # Backup methods on Server/Embedded
|       +-- rebuild.go          # Rebuild methods on Embedded
|       +-- compaction.go       # Compact methods on Embedded
|       +-- wal_prune.go        # WAL prune methods on Embedded
|       +-- chroma_test.go                              # package runtime
|       +-- embedded_test.go                            # package runtime
|       +-- embedded_benchmark_test.go                  # package runtime
|       +-- embedded_integration_edge_test.go           # package runtime
|       +-- embedded_metadata_validation_test.go        # package runtime
|       +-- embedded_property_test.go                   # package runtime
|       +-- embedded_create_collection_persistence_test.go  # package runtime
|       +-- backup_test.go                              # package runtime
|       +-- rebuild_test.go                             # package runtime
|       +-- compaction_test.go                          # package runtime
|       +-- wal_prune_test.go                           # package runtime
+-- go.mod              # Unchanged
+-- go.sum              # Unchanged
+-- internal_test.go    # REMOVED in commit 2
```

### Pattern 1: Package Declaration Change
**What:** Every moved file changes its `package` declaration
**When to use:** Every file being moved
**Example:**
```go
// BEFORE (at root):
package chroma

// AFTER (in internal/library/):
package library

// AFTER (in internal/runtime/):
package runtime
```

### Pattern 2: Export Promotion for Cross-Package Access
**What:** `loadLibrary` must become `LoadLibrary` since `runtime` needs to call it from `library`
**When to use:** The `loadLibrary` function in library_unix.go and library_windows.go
**Example:**
```go
// internal/library/library_unix.go
//go:build !windows

package library

import (
    "os"
    "runtime"

    "github.com/ebitengine/purego"
)

// LoadLibrary loads the Chroma shared library from the given path.
func LoadLibrary(path string) (uintptr, error) {
    plan, err := resolveLibraryLoadPlan(path, runtime.GOOS, os.Getenv, os.Stat)
    // ... rest unchanged except `package library` ...
}
```

### Pattern 3: Cross-Package Import in Init()
**What:** `Init()` in runtime calls `library.LoadLibrary()` instead of same-package `loadLibrary()`
**When to use:** The `Init` function in chroma.go (moved to internal/runtime/chroma.go)
**Example:**
```go
// internal/runtime/chroma.go
package runtime

import (
    "github.com/amikos-tech/chroma-go-local/internal/library"
    // ... other imports ...
)

func Init(libPath string) error {
    libOnce.Do(func() {
        libHandle, libErr = library.LoadLibrary(libPath)
        if libErr != nil {
            return
        }
        libErr = registerFunctions()
    })
    return libErr
}
```

### Anti-Patterns to Avoid
- **Split-state across packages:** Do NOT put some FFI globals in `library` and some in `runtime`. All globals stay together in `runtime` (LAYOUT-03).
- **Exporting internal helpers:** Do NOT export `resolveLibraryLoadPlan`, `candidateSet`, etc. Only `LoadLibrary` is exported from `library`.
- **Package-level init() functions:** Do NOT add `init()` functions. The explicit `Init()` call pattern is preserved as-is.
- **Changing test package to `_test` suffix:** Tests stay as white-box (`package runtime`, `package library`), NOT `package runtime_test`. This preserves access to unexported symbols like `libHandle`, `ffiMu`, `resolveLibraryLoadPlan`, etc.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Build tag verification | Manual file inspection | `go list -f '{{.GoFiles}}' ./internal/library/` | go list respects GOOS/GOARCH and build tags; manual checks can miss edge cases |
| Race condition detection | Manual mutex review | `go test -race ./internal/...` | Race detector catches all data races at runtime; manual review cannot |
| Test count verification | Manual counting | `go test -v ./internal/... \| grep -c '^--- '` | Automated count prevents test orphan drift |
| Cross-compile verification | Manual GOOS switching | `GOOS=linux go build ./internal/...` etc. | go build with explicit GOOS catches platform-specific import issues |

## Common Pitfalls

### Pitfall 1: Relative File Paths in Tests Break After Move
**What goes wrong:** `chroma_test.go` contains `os.ReadFile("shim/Cargo.toml")` which uses a path relative to the repo root. When moved to `internal/runtime/`, `go test` sets the working directory to `internal/runtime/`, so the relative path becomes invalid.
**Why it happens:** `go test` always runs with the package's source directory as the working directory, not the repo root.
**How to avoid:** Change `shim/Cargo.toml` to `../../shim/Cargo.toml` (two levels up from `internal/runtime/` to repo root). Alternatively, use `runtime.Caller(0)` to find the test file's directory and resolve paths relative to the module root.
**Warning signs:** `TestInitAndVersion` fails with "open shim/Cargo.toml: no such file or directory"
**Affected tests:** `readShimCargoVersion` in chroma_test.go (line 88)
**Recommendation:** Use the simple relative path fix `../../shim/Cargo.toml` -- it is straightforward and matches the known directory depth. A helper that walks up to find go.mod would be over-engineering for one usage.

### Pitfall 2: Persist Path Relativity in Tests
**What goes wrong:** Tests that specify `"./chroma_test_data"` as persist path will create data directories inside `internal/runtime/` instead of at the repo root.
**Why it happens:** Same working directory issue as Pitfall 1.
**How to avoid:** Tests that use `t.TempDir()` are safe (returns absolute path). Tests with hardcoded `"./chroma_test_data"` paths still work functionally but create data in a different location. Since `make clean` removes `./chroma_test_data` from root, these may leave artifacts in `internal/runtime/`.
**Warning signs:** Leftover `chroma_test_data` directories inside `internal/runtime/`.
**Recommendation:** Replace hardcoded `"./chroma_test_data*"` paths with `t.TempDir()` or accept the location change and update `make clean` if needed. The tests themselves will still pass either way since they create the directories.

### Pitfall 3: stdlib Name Collision with `package runtime`
**What goes wrong:** Files in `internal/runtime/` that import `"runtime"` (stdlib) while being `package runtime` will shadow the import.
**Why it happens:** The package name `runtime` collides with Go's standard `runtime` package. Multiple files import stdlib `runtime` for `runtime.SetFinalizer`, `runtime.KeepAlive`, `runtime.GOOS`.
**How to avoid:** Use import aliasing: `goruntime "runtime"` or `stdruntime "runtime"`. Then replace all `runtime.X` calls with the aliased name.
**Warning signs:** Compilation errors like "runtime.SetFinalizer not defined" or "runtime.KeepAlive not defined".
**Affected files:** chroma.go (uses `runtime.SetFinalizer`, `runtime.KeepAlive`), embedded.go (uses `runtime.SetFinalizer`, `runtime.KeepAlive`), backup.go (uses `runtime.KeepAlive`), rebuild.go (uses `runtime.KeepAlive`), compaction.go (uses `runtime.KeepAlive`), wal_prune.go (uses `runtime.KeepAlive`), library_unix.go (uses `runtime.GOOS`), library_windows.go (uses `runtime.GOOS`).
**Recommendation:** Use `goruntime "runtime"` as the alias in all internal/runtime files. For internal/library files, the package name is `library` so there is no collision -- they can import `"runtime"` normally.

### Pitfall 4: Forgetting to Replace Phase 1 Skeleton Files
**What goes wrong:** The Phase 1 skeleton files (`internal/runtime/runtime.go`, `internal/library/library.go`) contain only package declarations and doc comments. If not removed or replaced, they contribute a competing file with no useful content.
**Why it happens:** The skeleton files served as compilation anchors in Phase 1.
**How to avoid:** In commit 1, the library.go being moved should replace `internal/library/library.go`. In commit 2, `internal/runtime/runtime.go` should be removed or its doc comment merged into one of the moved files (e.g., chroma.go).
**Warning signs:** `go vet` warnings about duplicate doc comments, or leftover stub files.
**Recommendation:** Remove the skeleton files as part of the move. Transfer the package doc comment to the primary file in each package (library.go for library, chroma.go for runtime).

### Pitfall 5: Build Tag Format
**What goes wrong:** Using old `// +build` syntax instead of `//go:build` (or vice versa).
**Why it happens:** Go 1.17+ prefers `//go:build` but older code may use `// +build`.
**How to avoid:** The existing files already use the `//go:build` syntax (verified: line 1 of library_unix.go and library_windows.go). Preserve the exact same line during the move. Do not add or change the format.
**Warning signs:** `go vet` warnings about build tag format mismatch.
**Recommendation:** Copy the build tag line verbatim. No changes needed.

### Pitfall 6: Compilation Order in Two-Commit Strategy
**What goes wrong:** After commit 1 (library files moved), the root package cannot compile because `loadLibrary()` was called from `Init()` in chroma.go, but `loadLibrary()` no longer exists at root.
**Why it happens:** Moving library files removes `loadLibrary()` from `package chroma`, but `chroma.go` still calls it.
**How to avoid:** In commit 1, after moving library files to `internal/library/`, add a temporary bridge in the root package that calls `library.LoadLibrary()`. Specifically: update `chroma.go`'s `Init()` to import and call `library.LoadLibrary()` instead of `loadLibrary()`. This is needed because `loadLibrary` (from library_unix.go / library_windows.go) will no longer be at root.
**Warning signs:** `go build ./...` fails after commit 1 with "undefined: loadLibrary".
**Recommendation:** Commit 1 MUST update chroma.go's `Init()` function to call `library.LoadLibrary(libPath)` and add the import for `internal/library`. This ensures the root package compiles after the library files move but before the runtime files move.

### Pitfall 7: Test Files That Reference FFI Symbols Directly
**What goes wrong:** `embedded_metadata_validation_test.go` directly assigns to FFI function pointers (`chromaEmbeddedUpdateCollection`, `chromaEmbeddedCreateCollection`) to mock FFI calls. This is white-box testing that must remain in `package runtime`.
**Why it happens:** These tests monkey-patch package-level function pointer variables.
**How to avoid:** Tests must use `package runtime` (not `package runtime_test`). This is already the plan per CONTEXT.md.
**Warning signs:** Compilation errors about unexported symbols.

## Code Examples

### Example 1: library_unix.go After Migration
```go
//go:build !windows

package library

import (
    "os"
    "runtime"

    "github.com/ebitengine/purego"
)

// LoadLibrary loads the Chroma shared library using dlopen.
func LoadLibrary(path string) (uintptr, error) {
    plan, err := resolveLibraryLoadPlan(path, runtime.GOOS, os.Getenv, os.Stat)
    if err != nil {
        return 0, err
    }

    loadAttempts := make([]string, 0, len(plan.candidates))
    for _, candidate := range plan.candidates {
        libHandle, loadErr := purego.Dlopen(candidate.path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
        if loadErr == nil {
            if libHandle != 0 {
                return libHandle, nil
            }
            loadAttempts = append(loadAttempts, formatLoadAttempt(candidate, nil))
            return 0, formatLibraryLoadError(plan, loadAttempts)
        }
        loadAttempts = append(loadAttempts, formatLoadAttempt(candidate, loadErr))
    }

    return 0, formatLibraryLoadError(plan, loadAttempts)
}
```

### Example 2: chroma.go Init() After Migration (in internal/runtime/)
```go
package runtime

import (
    "fmt"
    goruntime "runtime"
    "strings"
    "sync"
    "sync/atomic"
    "unsafe"

    "github.com/amikos-tech/chroma-go-local/internal/library"
    "github.com/ebitengine/purego"
    "github.com/pkg/errors"
)

// Init initializes the Chroma library.
func Init(libPath string) error {
    libOnce.Do(func() {
        libHandle, libErr = library.LoadLibrary(libPath)
        if libErr != nil {
            return
        }
        libErr = registerFunctions()
    })
    return libErr
}
```

### Example 3: Relative Path Fix in chroma_test.go
```go
package runtime

// readShimCargoVersion reads the version from ../../shim/Cargo.toml
// (two directories up from internal/runtime/ to repo root).
func readShimCargoVersion(t *testing.T) string {
    t.Helper()
    data, err := os.ReadFile("../../shim/Cargo.toml")
    require.NoError(t, err)
    // ... rest unchanged ...
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `// +build` tags | `//go:build` tags | Go 1.17 (2021) | Both files already use `//go:build`; no change needed |
| `package chroma` at root | `package runtime` / `package library` in internal/ | This phase | Package declarations change, import aliasing needed for stdlib `runtime` |

**No deprecated patterns are involved.** This is purely a structural refactoring within the existing Go module system.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + testify v1.11.1 + gopter v0.2.11 |
| Config file | None (standard go test) |
| Quick run command | `go test -race -count=1 ./internal/...` |
| Full suite command | `CHROMA_LIB_PATH=$(path) go test -race -v ./internal/...` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LAYOUT-03 | FFI globals exclusively in internal/runtime, zero var state at root | smoke | `go list -f '{{.GoFiles}}' .` confirms no impl files; `grep -r 'var (' internal/runtime/chroma.go` confirms globals present | Wave 0 |
| LAYOUT-04 | Platform build tags retained after move | smoke | `GOOS=linux go list -f '{{.GoFiles}}' ./internal/library/` includes library_unix.go; `GOOS=windows go list -f '{{.GoFiles}}' ./internal/library/` includes library_windows.go | Wave 0 |
| LAYOUT-03 | Race-free initialization | integration | `go test -race -count=1 ./internal/...` | Existing tests move |
| LAYOUT-04 | Cross-compile passes | smoke | `GOOS=linux go build ./internal/...` and `GOOS=windows go build ./internal/...` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go build ./... && go test -race -count=1 ./internal/...`
- **Per wave merge:** `CHROMA_LIB_PATH=$(path) go test -race -v ./internal/... && go test -race -v ./...`
- **Phase gate:** Full suite green with race detector before `/gsd:verify-work`

### Wave 0 Gaps
- None -- existing test infrastructure (go test + testify + gopter) covers all phase requirements
- Tests move alongside implementation; no new test framework or configuration needed
- CHROMA_LIB_PATH must be set for tests that load the Rust shim (handled by Makefile)

## Open Questions

1. **Persist path test data location**
   - What we know: Tests using `"./chroma_test_data"` will create data inside `internal/runtime/` after the move
   - What's unclear: Whether this causes issues with `make clean` or CI cleanup
   - Recommendation: Accept the location change; it does not affect test correctness. Update `make clean` in Phase 4 if needed.

2. **Total test count verification**
   - What we know: Currently 135 tests listed via `go test -list '.*' ./...` at root (127 Test* functions + benchmarks)
   - What's unclear: Exact count after move (should be identical minus the zero tests from internal_test.go which has no Test functions)
   - Recommendation: Run `go test -list '.*' ./internal/...` after migration and compare counts

## Sources

### Primary (HIGH confidence)
- Direct codebase inspection of all 24 .go files at repo root
- `go list` output confirming current package structure
- `go.mod` confirming module path `github.com/amikos-tech/chroma-go-local`
- Go specification: `go test` working directory behavior (pkg.go.dev/cmd/go)

### Secondary (MEDIUM confidence)
- [Go test working directory behavior](https://github.com/golang-standards/project-layout/issues/97) - confirms working directory is package source dir
- [Testing Go from project root](https://brandur.org/fragments/testing-go-project-root) - relative path resolution patterns

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all libraries already in go.mod, no new dependencies
- Architecture: HIGH - direct inspection of all source files, dependency graph fully mapped
- Pitfalls: HIGH - verified all 7 pitfalls through direct code analysis and Go toolchain behavior
- File mapping: HIGH - every file inspected for package declaration, imports, and cross-file dependencies

**Research date:** 2026-03-20
**Valid until:** 2026-04-20 (stable -- Go migration patterns do not change frequently)
