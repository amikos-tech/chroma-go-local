# Phase 1: Layout Design - Research

**Researched:** 2026-03-20
**Domain:** Go internal packages, skeleton package creation, module structure
**Confidence:** HIGH

## Summary

Phase 1 is a purely additive structural operation: create two empty skeleton packages under `internal/runtime/` and `internal/library/` at the module root, then add a test file proving the root package can import them. No existing files move; no behavior changes.

The Go `internal/` directory mechanism is a stable, well-documented feature with simple rules: a package under `internal/` can only be imported by code in the directory tree rooted at `internal/`'s parent. Since `go.mod` lives at the repo root and `internal/` will be a direct child, every package in this module (including the root `chroma` package) can import from `internal/runtime` and `internal/library`. External consumers cannot.

**Primary recommendation:** Create two `.go` files (`internal/runtime/runtime.go` and `internal/library/library.go`) containing only a doc comment and `package` declaration, plus one `internal_test.go` at the repo root that blank-imports both packages from an external test perspective (`package chroma_test`). Verify with `go build ./...` and `go vet ./...`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- `internal/runtime/` gets ALL implementation files: chroma.go, config.go, embedded.go, errors.go, backup.go, rebuild.go, compaction.go, wal_prune.go
- `internal/library/` gets FFI loading files: library.go, library_unix.go, library_windows.go
- Maintenance operations (backup, rebuild, compaction, wal_prune) stay in runtime because they define methods on Server/Embedded receiver types -- Go requires methods in the same package as the receiver
- Dependency direction: runtime imports library (to load the shared lib); library is a leaf package with no upstream dependencies
- Package names: `runtime` and `library`
- Skeleton depth: bare package declarations only -- no stub types or placeholder functions
- Each skeleton file includes a brief 1-2 line doc comment explaining the package's intended role
- Doc comment lives in the main .go file (runtime.go, library.go), not a separate doc.go
- A standalone `internal_test.go` at repo root validates the internal/ anchor using `package chroma_test` with blank-imports of both internal packages
- `internal_test.go` is temporary -- removed after Phase 3 when the real facade exists

### Claude's Discretion
- Exact doc comment wording for skeleton packages
- Whether to add a .gitkeep or similar in empty directories (likely unnecessary since .go files exist)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LAYOUT-01 | All Go implementation files moved from repo root into `internal/` subtree at module root | This phase creates the target directories as skeletons; actual file movement happens in Phase 2. The skeleton structure directly enables LAYOUT-01 by establishing the destination directories. |
| LAYOUT-02 | Implementation organized into `internal/runtime/` (server, embedded, config, errors) and `internal/library/` (FFI loading, platform shims) | The skeleton packages define the exact two-package structure. Doc comments document intended contents. The `internal_test.go` validates that the root package can import both. |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go toolchain | 1.21+ (project uses 1.26.1 locally) | Build and vet | Project's go.mod specifies `go 1.21`; all internal/ features available since Go 1.4 |

### Supporting
No additional libraries needed for this phase. Skeleton packages contain no imports.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Bare `package` declarations | Stub types/interfaces | Over-engineering for Phase 1; stubs create merge conflicts when real files arrive in Phase 2 |
| `internal_test.go` anchor test | Manual `go build` verification | Test file provides repeatable automated proof; removed in Phase 3 anyway |

## Architecture Patterns

### Recommended Project Structure (after Phase 1)
```
github.com/amikos-tech/chroma-go-local/
├── go.mod                          # module root (unchanged)
├── chroma.go                       # existing root package (unchanged)
├── config.go                       # existing (unchanged)
├── embedded.go                     # existing (unchanged)
├── errors.go                       # existing (unchanged)
├── library.go                      # existing (unchanged)
├── library_unix.go                 # existing (unchanged)
├── library_windows.go              # existing (unchanged)
├── backup.go                       # existing (unchanged)
├── rebuild.go                      # existing (unchanged)
├── compaction.go                   # existing (unchanged)
├── wal_prune.go                    # existing (unchanged)
├── *_test.go                       # existing tests (unchanged)
├── internal_test.go                # NEW: anchor validation test
├── internal/
│   ├── runtime/
│   │   └── runtime.go              # NEW: skeleton (package runtime)
│   └── library/
│       └── library.go              # NEW: skeleton (package library)
├── shim/                           # Rust shim (unchanged)
└── java/                           # Java bindings (unchanged)
```

### Pattern 1: Go Internal Package Visibility
**What:** The `internal/` directory is a Go convention enforced by the toolchain. Any package under `internal/` can only be imported by packages in the directory tree rooted at `internal/`'s parent.
**When to use:** When you want to hide implementation details from external consumers while allowing internal code sharing.
**Rule for this project:**
- Module path: `github.com/amikos-tech/chroma-go-local`
- `internal/` is at module root
- Therefore: all packages in this module can import `internal/runtime` and `internal/library`
- External modules CANNOT import these packages (enforced by `go build`)

**Verified empirically:** Created a test module with identical structure; `go build ./...` succeeds for root-to-internal imports and fails for cross-module internal imports.

### Pattern 2: Skeleton Package Declaration
**What:** A minimal Go source file containing only a package-level doc comment and `package` declaration. No types, no functions, no imports.
**When to use:** To establish directory structure and prove compilability before migrating real code.
**Example:**
```go
// Package runtime contains the Chroma server and embedded runtime implementation.
package runtime
```

### Pattern 3: External Test Package for Anchor Validation
**What:** A test file at the module root using `package chroma_test` (external test perspective) that blank-imports the internal packages. This proves the `internal/` visibility rules permit root-package access.
**When to use:** When you need compile-time proof that `internal/` is correctly positioned relative to the module root.
**Example:**
```go
package chroma_test

import (
	_ "github.com/amikos-tech/chroma-go-local/internal/library"
	_ "github.com/amikos-tech/chroma-go-local/internal/runtime"
)
```

### Anti-Patterns to Avoid
- **Placing stub types in skeleton files:** Creates merge conflicts when real files arrive in Phase 2. Bare declarations are sufficient.
- **Using `doc.go` instead of the main package file:** CONTEXT.md explicitly says doc comment lives in `runtime.go`/`library.go`, not a separate `doc.go`.
- **Adding `init()` functions to skeleton packages:** Skeleton files must be inert. Any initialization will be added in Phase 2.
- **Placing `internal/` anywhere other than module root:** Would break the visibility rule -- subpackages at root could not import internal packages.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Internal visibility enforcement | Custom import restrictions or build tags | Go's built-in `internal/` mechanism | Compiler-enforced, zero configuration, standard Go convention |
| Anchor validation | Manual `go build` checks | `internal_test.go` with blank imports | Automated, repeatable, self-documenting |

**Key insight:** Go's `internal/` mechanism is entirely convention-based and compiler-enforced. There is nothing to configure, no build tags to set, and no `.go` file metadata to add. Simply placing files under `internal/` activates the visibility restriction.

## Common Pitfalls

### Pitfall 1: Wrong `internal/` Placement
**What goes wrong:** Placing `internal/` inside a subdirectory (e.g., `go/internal/`) instead of at the module root causes "use of internal package not allowed" errors when the root package tries to import.
**Why it happens:** Misunderstanding the parent-tree rule. `internal/` restricts to the parent directory tree -- if `internal/` is under `go/`, only code under `go/` can import from it.
**How to avoid:** `internal/` must be a direct child of the directory containing `go.mod`. Verified: `go.mod` is at repo root, so `internal/` goes at repo root.
**Warning signs:** `go build` error: `use of internal package ... not allowed`

### Pitfall 2: Package Name Conflicts
**What goes wrong:** Naming the skeleton package `runtime` might shadow Go's standard library `runtime` package in future imports.
**Why it happens:** Go resolves unqualified package names by the last path element. If a file in `internal/runtime/` later imports `runtime` (stdlib), it would be a self-import.
**How to avoid:** This is a Phase 2 concern, not Phase 1. In the skeleton, there are no imports. When real code arrives, it will use the full import path `"runtime"` for stdlib vs the package's own declarations. Go handles this correctly -- a package can import stdlib `runtime` even if the package itself is named `runtime`. The compiler distinguishes by import path.
**Warning signs:** Circular import errors (will not happen in Phase 1 with bare skeletons).

### Pitfall 3: Mixing Package Declarations in Same Directory
**What goes wrong:** Having both `package chroma` and `package chroma_test` files in the root directory could theoretically confuse the build.
**Why it happens:** Go allows exactly two package names per directory: `package X` and `package X_test`. The `_test` suffix is special.
**How to avoid:** The existing test files all use `package chroma`. The new `internal_test.go` uses `package chroma_test`. This is perfectly valid -- Go explicitly supports both in the same directory. Verified empirically.
**Warning signs:** Build error about multiple packages in directory (would only happen with a third package name).

### Pitfall 4: `go test ./...` Discovering Empty Internal Packages
**What goes wrong:** `make test` runs `go test -v ./...` which now discovers `internal/runtime` and `internal/library`. They show `[no test files]`.
**Why it happens:** `./...` pattern matches all packages including new ones.
**How to avoid:** Nothing to avoid -- `[no test files]` is informational, not an error. `go test` still exits 0. Verified empirically.
**Warning signs:** None -- this is expected behavior.

### Pitfall 5: Linter Warnings on Empty Packages
**What goes wrong:** Some linters might warn about packages with no exported symbols or no code.
**Why it happens:** Static analysis tools sometimes flag empty or minimal packages.
**How to avoid:** The doc comment satisfies `ST1000` (package comment check in staticcheck). The current `.golangci.yml` has `comments` in exclusion presets, so even missing comments would be suppressed. Verified: `golangci-lint run ./...` passes on skeleton packages.
**Warning signs:** Linter output referencing `internal/runtime/runtime.go` or `internal/library/library.go`.

## Code Examples

Verified patterns from empirical testing:

### Skeleton Package: internal/runtime/runtime.go
```go
// Package runtime contains the Chroma server and embedded runtime implementation.
package runtime
```

### Skeleton Package: internal/library/library.go
```go
// Package library provides platform-specific FFI loading for the Chroma shared library.
package library
```

### Anchor Validation Test: internal_test.go
```go
package chroma_test

import (
	_ "github.com/amikos-tech/chroma-go-local/internal/library"
	_ "github.com/amikos-tech/chroma-go-local/internal/runtime"
)
```

### Verification Commands
```bash
# Must all pass after Phase 1:
go build ./...          # Compiles all packages including skeletons
go vet ./...            # Static analysis passes
go test ./...           # Tests pass (internal packages show [no test files])
golangci-lint run ./... # Linter passes
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `internal/` only enforced in GOPATH | `internal/` enforced in modules too | Go 1.13 (Aug 2019) | Module-mode `internal/` is fully standard; this project uses modules |

**Deprecated/outdated:**
- Nothing relevant. The `internal/` mechanism has been stable since Go 1.4 (Feb 2015) and unchanged in module mode.

## Open Questions

1. **Doc comment wording**
   - What we know: CONTEXT.md says 1-2 line doc comments explaining intended role
   - What's unclear: Exact phrasing (marked as Claude's discretion)
   - Recommendation: Use descriptive comments that match the file-to-package mapping from CONTEXT.md decisions. Suggested above in Code Examples section.

2. **Stale gci prefix in .golangci.yml**
   - What we know: `.golangci.yml` has `prefix(github.com/chaoslabs-bg/tclr-v2/)` instead of `prefix(github.com/amikos-tech/chroma-go-local/)`
   - What's unclear: Whether this causes issues for Phase 1 skeleton packages (it does not -- skeletons have no imports)
   - Recommendation: Do NOT fix in Phase 1. This is BUILD-03, assigned to Phase 4. The stale prefix is harmless for Phase 1.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib), go 1.21+ |
| Config file | None required -- Go testing is convention-based |
| Quick run command | `go build ./...` (compile check is the primary validation for Phase 1) |
| Full suite command | `go build ./... && go vet ./... && go test ./...` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LAYOUT-01 | `internal/runtime/` and `internal/library/` directories exist with valid package declarations | smoke | `go build ./internal/...` | No -- Wave 0 |
| LAYOUT-02 | Root package can import internal packages without visibility errors | smoke | `go build ./... && go vet ./...` | No -- Wave 0 (internal_test.go created as part of phase) |

### Sampling Rate
- **Per task commit:** `go build ./...`
- **Per wave merge:** `go build ./... && go vet ./... && golangci-lint run ./...`
- **Phase gate:** `go build ./... && go vet ./... && go test ./... && golangci-lint run ./...`

### Wave 0 Gaps
- [x] Go test framework -- already available (stdlib)
- [ ] `internal/runtime/runtime.go` -- skeleton package (created as part of implementation)
- [ ] `internal/library/library.go` -- skeleton package (created as part of implementation)
- [ ] `internal_test.go` -- anchor validation test (created as part of implementation)

Note: For this phase, the "test infrastructure" IS the deliverable. The skeleton files and the anchor test file are both the implementation and the validation. No separate test setup is needed beyond what the phase itself creates.

## Sources

### Primary (HIGH confidence)
- [Go cmd/go documentation - Internal Directories](https://pkg.go.dev/cmd/go#hdr-Internal_Directories) -- authoritative rule definition
- [Go issue #38579](https://github.com/golang/go/issues/38579) -- module-mode internal visibility clarification
- Empirical verification -- created test module on local machine, confirmed all behaviors described

### Secondary (MEDIUM confidence)
- [Go Internal Packages article](https://www.aicodesnippet.com/go/packages-and-modules/go-internal-packages-controlling-visibility.html) -- community explanation of rules

### Tertiary (LOW confidence)
- None -- all findings verified empirically or via official docs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- Go stdlib only, no external dependencies; verified with local Go 1.26.1
- Architecture: HIGH -- `internal/` is a well-documented, stable Go convention; verified empirically with identical module structure
- Pitfalls: HIGH -- all potential issues tested and confirmed/disproved empirically

**Research date:** 2026-03-20
**Valid until:** Indefinite -- Go `internal/` mechanism is stable and unlikely to change
