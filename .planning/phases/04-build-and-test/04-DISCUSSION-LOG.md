# Phase 4: Build and Test - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-21
**Phase:** 04-build-and-test
**Areas discussed:** compat_test.go design, Root-level test strategy, Makefile & CI adjustments

---

## compat_test.go design

### Q1: What approach for compat_test.go?

| Option | Description | Selected |
|--------|-------------|----------|
| Compile-only references | Blank-assign every exported symbol to _ vars. Zero runtime cost, pure surface gate. | |
| Type assertion checks | Compile-only + interface satisfaction assertions (e.g., var _ io.Closer = (*Server)(nil)). | |
| Behavioral smoke tests | Actual Test* functions that call facade functions and verify results. | |
| Combined: #1 + #3 | Compile-only surface gate + behavioral smoke tests. Skip type assertions. | ✓ |

**User's choice:** Combined compile-only + behavioral smoke tests (skip type assertions)
**Notes:** User's goals: (1) don't break existing consumers, (2) free up repo space for Java impl, (3) future-proof with regression catching. Type assertions skipped because type aliases auto-forward methods — interface compliance can't drift independently. Three-layer defense: compile gate + behavioral smoke + go-apidiff in Phase 5.

### Q2: How should compat_test.go be organized?

| Option | Description | Selected |
|--------|-------------|----------|
| Single file | One compat_test.go with compile-only vars at top, TestXxx functions below. | ✓ |
| Two files | compat_test.go for compile gate, facade_test.go for behavioral tests. | |

**User's choice:** Single file
**Notes:** Matches roadmap's single-file expectation. Simple to scan.

### Q3: Which facade functions should get behavioral smoke tests?

| Option | Description | Selected |
|--------|-------------|----------|
| Key entry points only (~5 tests) | Init, Version, NewServer, StartEmbedded, DefaultConfigs. | |
| All wrapper functions (~37 tests) | Every wrapper behaviorally verified. Comprehensive but verbose. | |
| Entry points + one per feature area (~9-10 tests) | Key entry points plus one test per feature domain (backup, rebuild, compaction, WAL prune). | ✓ |

**User's choice:** Entry points + one per feature area
**Notes:** User liked maintenance aspect and confidence level. Three-layer defense means not every wrapper needs behavioral testing — compile gate covers symbol existence, internal tests cover implementation, go-apidiff catches surface drift.

### Q4: Test package declaration?

| Option | Description | Selected |
|--------|-------------|----------|
| package chroma_test (external) | Imports root package like a real consumer. Proves public import path works. | ✓ |
| package chroma (internal) | Can access unexported symbols. Doesn't prove consumer experience. | |

**User's choice:** package chroma_test
**Notes:** Matches roadmap specification. External perspective is the whole point of the compat gate.

---

## Root-level test strategy

### Q1: Should root-level tests be limited to compat_test.go?

| Option | Description | Selected |
|--------|-------------|----------|
| compat_test.go only | Root has exactly one test file. All deep testing in internal/. | ✓ |
| compat_test.go + integration tests | Second root-level file with end-to-end scenarios. | |
| Mirror key internal tests at root | Copy/adapt internal tests to run through facade. | |

**User's choice:** compat_test.go only
**Notes:** Clean separation: root = API surface gate, internal = implementation tests. No duplication.

---

## Makefile & CI adjustments

### Q1: Should make test add granular targets?

| Option | Description | Selected |
|--------|-------------|----------|
| Keep ./... only | No Makefile changes. go test -v ./... covers everything. | ✓ |
| Add granular targets | Add test-compat (root only) and test-internal alongside test-go. | |
| Add race detector target | Add test-race running go test -race ./... | |

**User's choice:** Keep ./... only
**Notes:** No target sprawl. ./... already traverses root compat_test.go + all internal tests.

### Q2: Should CI workflow be modified?

| Option | Description | Selected |
|--------|-------------|----------|
| No structural changes | CI already uses go test -v ./... and golangci-lint ./... | ✓ |
| Add cross-compile step | Explicit GOOS builds in CI. | |
| Add race detector step | go test -race ./... as separate CI step. | |

**User's choice:** No structural changes
**Notes:** Only change needed is the gci prefix fix in .golangci.yml, which CI picks up automatically.

### Q3: Lint config scope?

| Option | Description | Selected |
|--------|-------------|----------|
| Fix gci prefix only | Change stale prefix from chaoslabs-bg/tclr-v2 to amikos-tech/chroma-go-local. | ✓ |
| Fix prefix + audit config | Fix prefix and review all lint settings for relevance. | |

**User's choice:** Fix gci prefix only
**Notes:** Remaining config is valid for the reorganized layout. Minimal change, minimal risk.

---

## Claude's Discretion

- Ordering of compile-only symbol references within compat_test.go
- Test helper setup/teardown patterns for behavioral smoke tests
- Whether behavioral tests use subtests or top-level functions
- Cross-compile verification approach

## Deferred Ideas

None — discussion stayed within phase scope
