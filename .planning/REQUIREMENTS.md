# Requirements: chroma-go-local v0.4.0

**Defined:** 2026-03-20
**Core Value:** Public Go import path and API surface must remain 100% backward-compatible

## v1 Requirements

### Layout Migration

- [x] **LAYOUT-01**: All Go implementation files moved from repo root into `internal/` subtree at module root
- [x] **LAYOUT-02**: Implementation organized into `internal/runtime/` (server, embedded, config, errors) and `internal/library/` (FFI loading, platform shims)
- [x] **LAYOUT-03**: All FFI globals and `sync.Once` initialization moved atomically to implementation package (no split state)
- [x] **LAYOUT-04**: Platform-specific files (`library_unix.go`, `library_windows.go`) retain correct build tags after move

### Import Facade

- [x] **FACADE-01**: Root package exposes all current public types via type aliases (`type X = impl.X`)
- [x] **FACADE-02**: Root package re-exports all public functions via variable assignments or wrapper calls
- [x] **FACADE-03**: Root package re-exports all constants, variables, and error types
- [x] **FACADE-04**: Root package contains zero implementation logic (pure forwarding only)
- [x] **FACADE-05**: Import path `github.com/amikos-tech/chroma-go-local` remains valid and unchanged

### Test Reorganization

- [x] **TEST-01**: Implementation-focused tests moved alongside new internal packages
- [x] **TEST-02**: Public API compatibility tests remain at root level
- [x] **TEST-03**: `compat_test.go` added at root as compile-time API surface gate
- [x] **TEST-04**: `make test` passes with reorganized test layout

### Build & CI

- [x] **BUILD-01**: Makefile targets updated for new package paths (`make test`, `make lint`, `make test-all`)
- [x] **BUILD-02**: CI workflows (`.github/workflows/ci.yml`) updated for new structure
- [x] **BUILD-03**: Stale `gci` prefix in `.golangci.yml` corrected to `github.com/amikos-tech/chroma-go-local/`
- [x] **BUILD-04**: Cross-compile verification passes for `GOOS=windows`, `GOOS=linux`, `GOOS=darwin`

### Docs & Verification

- [ ] **DOCS-01**: `go-apidiff` run against v0.3.4 tag confirms zero breaking changes
- [x] **DOCS-02**: README.md updated with new directory layout and build instructions
- [x] **DOCS-03**: CLAUDE.md updated to reflect new architecture
- [x] **DOCS-04**: GO_API_SURFACE.md references updated for new file locations

### Compatibility Gate

- [ ] **COMPAT-01**: Explicit compatibility checklist completed before merge
- [ ] **COMPAT-02**: No import-path break for current users verified
- [x] **COMPAT-03**: Release notes include refactor summary and compatibility statement

## v2 Requirements

### Future Improvements

- **FUTURE-01**: Verify `pkg.go.dev` rendering of type aliases to internal paths
- **FUTURE-02**: Evaluate `go-apidiff` as permanent CI gate for future releases

## Out of Scope

| Feature | Reason |
|---------|--------|
| New API features or methods | This is purely structural; new features belong to future milestones |
| Java binding layout changes | Java layout stays in `java/`; not part of this refactor |
| Rust shim changes | Shim stays in `shim/`; no code changes needed |
| Go module path change | Must remain `github.com/amikos-tech/chroma-go-local` |
| Second `go.mod` under `go/` | Creates separate module requiring `replace` directives; breaks published module |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| LAYOUT-01 | Phase 1 | Complete |
| LAYOUT-02 | Phase 1 | Complete |
| LAYOUT-03 | Phase 2 | Complete |
| LAYOUT-04 | Phase 2 | Complete |
| FACADE-01 | Phase 3 | Complete |
| FACADE-02 | Phase 3 | Complete |
| FACADE-03 | Phase 3 | Complete |
| FACADE-04 | Phase 3 | Complete |
| FACADE-05 | Phase 3 | Complete |
| TEST-01 | Phase 4 | Complete |
| TEST-02 | Phase 4 | Complete |
| TEST-03 | Phase 4 | Complete |
| TEST-04 | Phase 4 | Complete |
| BUILD-01 | Phase 4 | Complete |
| BUILD-02 | Phase 4 | Complete |
| BUILD-03 | Phase 4 | Complete |
| BUILD-04 | Phase 4 | Complete |
| DOCS-01 | Phase 5 | Pending |
| DOCS-02 | Phase 5 | Complete |
| DOCS-03 | Phase 5 | Complete |
| DOCS-04 | Phase 5 | Complete |
| COMPAT-01 | Phase 5 | Pending |
| COMPAT-02 | Phase 5 | Pending |
| COMPAT-03 | Phase 5 | Complete |

**Coverage:**
- v1 requirements: 24 total
- Mapped to phases: 24
- Unmapped: 0

---
*Requirements defined: 2026-03-20*
*Last updated: 2026-03-20 after roadmap creation*
