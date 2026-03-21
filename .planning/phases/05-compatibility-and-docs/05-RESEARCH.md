# Phase 5: Compatibility and Docs - Research

**Researched:** 2026-03-21
**Domain:** Go API compatibility verification, documentation updates, release management
**Confidence:** HIGH

## Summary

Phase 5 is a verification-and-documentation phase with no structural code changes. The primary technical challenge is `go-apidiff`, which reports 90 "incompatible changes" that are ALL false positives caused by type aliases pointing to `internal/runtime.*`. This was anticipated by D-04 and verified experimentally: consumer code compiles and behaves identically. The remaining work is straightforward documentation updates (README.md, CLAUDE.md, GO_API_SURFACE.md), a CHANGELOG.md entry, CI integration of apidiff on PRs to main, and a PR compatibility checklist.

The `apidiff` tool (from `golang.org/x/exp/cmd/apidiff`) is installed and working. The correct workflow uses the Go module cache to export the v0.3.4 API surface, then compares against the current working tree. A CI step can replicate this workflow using `go mod download` to resolve the baseline tag dynamically.

**Primary recommendation:** Run apidiff for documentation purposes, explicitly document the 90 false positives as expected type-alias artifacts, verify real compatibility through the existing compile gate (`compat_test.go`) plus `go test -race ./...`, and mark the apidiff check as passed with explanation per D-04.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Install `go-apidiff` via `go install golang.org/x/exp/cmd/apidiff@latest`, run one-shot against v0.3.4 tag for this release
- **D-02:** Also add a CI step in `.github/workflows/ci.yml` that runs apidiff on PRs to main, comparing against the latest release tag (dynamically resolved)
- **D-03:** CI apidiff step runs only on PRs to main -- not on every push
- **D-04:** If apidiff reports false positives from type aliases (e.g., "type changed from runtime.Server to Server"), document them as expected and mark the check as passed -- these aren't real breaking changes
- **D-05:** README.md adds a brief architecture note mentioning that implementation lives in `internal/` packages while root is a facade -- helps contributors who discover the repo
- **D-06:** GO_API_SURFACE.md: update file path references only (e.g., "defined in chroma.go" becomes "defined in internal/runtime/chroma.go, re-exported at root"). Content and examples stay identical.
- **D-07:** CLAUDE.md architecture diagram updated to show root facade -> internal/runtime + internal/library. Same concise style, updated paths.
- **D-08:** CLAUDE.md "Key Patterns" section gets a brief facade pattern note: type aliases, thin wrappers, zero logic rule
- **D-09:** Checklist lives in the PR description (markdown checklist) -- no separate file
- **D-10:** Checklist items: go-apidiff clean vs v0.3.4, import path resolves correctly, race detector clean (`go test -race ./...`), cross-compile passes (linux/darwin/windows)
- **D-11:** No manual verification steps -- automated checks are sufficient. examples/ already covered by `go build ./...`
- **D-12:** Both GitHub release body (attached to v0.4.0 tag) AND a CHANGELOG.md entry in the repo
- **D-13:** Compatibility statement: direct reassurance -- "No breaking changes. Your existing code using github.com/amikos-tech/chroma-go-local requires zero modifications."
- **D-14:** No roadmap hints or forward-looking notes in release notes -- keep focused on v0.4.0

### Claude's Discretion
- Exact wording of the README architecture note
- CHANGELOG.md format (Keep a Changelog vs custom)
- apidiff CI step implementation details (job name, caching)
- Ordering of checklist items in PR description

### Deferred Ideas (OUT OF SCOPE)
- pkg.go.dev rendering check for type aliases pointing to internal/ paths -- noted as concern in STATE.md, can be evaluated post-merge on a staging branch (FUTURE-01 in REQUIREMENTS.md)
- go-apidiff as permanent CI gate evaluation (FUTURE-02 in REQUIREMENTS.md) -- partially addressed by D-02, full evaluation deferred
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DOCS-01 | `go-apidiff` run against v0.3.4 tag confirms zero breaking changes | apidiff workflow verified locally; 90 false positives from type aliases documented; per D-04, these are expected and the check passes |
| DOCS-02 | README.md updated with new directory layout and build instructions | Current README Project Structure section identified (lines 664-688); needs architecture note after "Building" section per D-05 |
| DOCS-03 | CLAUDE.md updated to reflect new architecture | Architecture diagram at lines 48-54 and Key Patterns at lines 61-76 identified; needs facade pattern note per D-07/D-08 |
| DOCS-04 | GO_API_SURFACE.md references updated for new file locations | GO_API_SURFACE.md has no explicit file path references in body text; only implicit "defined in chroma.go" style references need updating per D-06 |
| COMPAT-01 | Explicit compatibility checklist completed before merge | Checklist template defined per D-09/D-10/D-11; goes in PR description |
| COMPAT-02 | No import-path break for current users verified | Consumer test verified locally -- code using `github.com/amikos-tech/chroma-go-local` compiles and resolves correctly against the facade |
| COMPAT-03 | Release notes include refactor summary and compatibility statement | Both CHANGELOG.md and GitHub release body needed per D-12; compatibility statement wording locked per D-13 |
</phase_requirements>

## Standard Stack

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| `golang.org/x/exp/cmd/apidiff` | v0.0.0-20260312153236 (latest) | API compatibility diff between Go package versions | Official Go team tool for semantic version compatibility checking |
| `go mod download` | (Go 1.26.1 built-in) | Fetch old module version to cache for apidiff export | Standard Go module mechanism; avoids git worktree complexity |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| `go test -race ./...` | (Go 1.26.1 built-in) | Race detector verification for compatibility checklist | D-10 checklist item |
| `go build ./...` cross-compile | (Go 1.26.1 built-in) | Cross-platform compilation check | D-10 checklist item (linux/darwin/windows) |
| `gh release create` | GitHub CLI | Create v0.4.0 release with custom body | D-12 release notes (done at merge time, not during implementation) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `golang.org/x/exp/cmd/apidiff` (direct) | `joelanford/go-apidiff` GitHub Action | The Action compares between commits (PR base vs HEAD); we need tag-based comparison. Direct `apidiff` is more flexible for our use case. |
| Module cache approach | Git worktree approach | Worktree requires `go.mod` at worktree root and has module resolution issues. Module cache is cleaner. |

## Architecture Patterns

### apidiff Workflow (One-Shot for v0.3.4)

```bash
# Step 1: Install apidiff
go install golang.org/x/exp/cmd/apidiff@latest

# Step 2: Download v0.3.4 to module cache, get its local path
OLD_DIR=$(go mod download -json github.com/amikos-tech/chroma-go-local@v0.3.4 | jq -r .Dir)

# Step 3: Export old API surface
cd "$OLD_DIR"
apidiff -w /tmp/old.txt github.com/amikos-tech/chroma-go-local

# Step 4: Export new API surface (from working tree)
cd /path/to/repo
apidiff -w /tmp/new.txt github.com/amikos-tech/chroma-go-local

# Step 5: Compare
apidiff -incompatible /tmp/old.txt /tmp/new.txt
```

### CI apidiff Job Structure (for PRs to main)

The CI step should be a separate job in `ci.yml` that:
1. Runs only on `pull_request` events targeting `main` (D-03)
2. Dynamically resolves the latest release tag via `git tag -l 'v*' --sort=-v:refname | head -1`
3. Downloads that tagged version to the module cache
4. Exports old and new API surfaces
5. Runs `apidiff -incompatible` and reports results
6. Does NOT fail the build on type-alias false positives (informational only)

Recommended job name: `api-compat`

```yaml
  api-compat:
    name: API Compatibility Check
    if: github.event_name == 'pull_request' && github.base_ref == 'main'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
        with:
          fetch-depth: 0
      - uses: actions/setup-go@4b73464bb391d4059bd26b0524d20df3927bd417 # v6.3.0
        with:
          go-version-file: go.mod
          cache: true
      - name: Run API compatibility check
        run: |
          go install golang.org/x/exp/cmd/apidiff@latest
          LATEST_TAG=$(git tag -l 'v*' --sort=-v:refname | head -1)
          echo "Comparing against: $LATEST_TAG"
          OLD_DIR=$(go mod download -json "github.com/amikos-tech/chroma-go-local@${LATEST_TAG}" | jq -r .Dir)
          cd "$OLD_DIR" && apidiff -w /tmp/old.txt github.com/amikos-tech/chroma-go-local
          cd "$GITHUB_WORKSPACE" && apidiff -w /tmp/new.txt github.com/amikos-tech/chroma-go-local
          echo "## API Diff (vs $LATEST_TAG)" >> "$GITHUB_STEP_SUMMARY"
          if apidiff -incompatible /tmp/old.txt /tmp/new.txt > /tmp/diff.txt 2>&1; then
            echo "No incompatible changes detected." >> "$GITHUB_STEP_SUMMARY"
          else
            echo '```' >> "$GITHUB_STEP_SUMMARY"
            cat /tmp/diff.txt >> "$GITHUB_STEP_SUMMARY"
            echo '```' >> "$GITHUB_STEP_SUMMARY"
            echo "::warning::API changes detected vs $LATEST_TAG - review step summary"
          fi
```

Note: This job uses `::warning` instead of failing, because type-alias false positives are expected after the internal/ refactor. The step summary provides full visibility.

### Documentation Update Targets

| File | Section | Change |
|------|---------|--------|
| `README.md` | "Project Structure" (line 664) | Replace old flat layout with facade + internal layout |
| `README.md` | After "Building" / before "Java Scaffold" | Add architecture note about facade pattern (D-05) |
| `CLAUDE.md` | Architecture diagram (line 48) | Update to show `root facade -> internal/runtime + internal/library` |
| `CLAUDE.md` | Key Patterns (line 61) | Add facade pattern note: type aliases, thin wrappers, zero logic (D-08) |
| `GO_API_SURFACE.md` | Throughout | Update file path references only -- no content changes (D-06) |

### CHANGELOG.md Format

Use "Keep a Changelog" format (https://keepachangelog.com/) since no existing CHANGELOG.md exists:

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.4.0] - 2026-XX-XX

### Changed
- Internal structure reorganized: Go implementation moved from repo root
  into `internal/runtime/` and `internal/library/` subtrees. The root
  package is now a thin facade that re-exports all public symbols via
  type aliases and function wrappers.

### Compatibility
- No breaking changes. Your existing code using
  `github.com/amikos-tech/chroma-go-local` requires zero modifications.
- Import path unchanged: `github.com/amikos-tech/chroma-go-local`
- All 110+ exported symbols preserved with identical signatures
- Verified via `go-apidiff` against v0.3.4 baseline

[0.4.0]: https://github.com/amikos-tech/chroma-go-local/compare/v0.3.4...v0.4.0
```

### PR Compatibility Checklist Template

Per D-09, this lives in the PR description:

```markdown
## Compatibility Checklist

- [ ] `go-apidiff` run against v0.3.4: all reported changes are type-alias false positives (documented in commit)
- [ ] Import path `github.com/amikos-tech/chroma-go-local` resolves correctly
- [ ] Race detector clean: `go test -race ./...` passes
- [ ] Cross-compile passes: `GOOS=linux`, `GOOS=darwin`, `GOOS=windows`
- [ ] `compat_test.go` compile gate: 110 symbols + 9 behavioral tests pass
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| API compatibility checking | Manual symbol-by-symbol comparison | `golang.org/x/exp/cmd/apidiff` | Handles type correspondence, method sets, interface changes -- manual is error-prone |
| Release tag resolution in CI | Hard-coded tag names | `git tag -l 'v*' --sort=-v:refname \| head -1` | Automatically picks up new releases without manual bumping |
| CHANGELOG format | Custom format | Keep a Changelog convention | Widely recognized, parseable, consistent structure |

## Common Pitfalls

### Pitfall 1: apidiff False Positives with Type Aliases to internal/
**What goes wrong:** `apidiff` reports every type alias (`type X = internal/runtime.X`) as an incompatible change because the underlying type's package path changed from root to `internal/runtime`.
**Why it happens:** apidiff tracks the fully-qualified type path. An alias to `internal/runtime.Server` is technically a different type identity than the original `Server` defined at root, even though Go's type system treats them as identical at compile time.
**How to avoid:** Per D-04, document these as expected false positives. The real compatibility verification comes from `compat_test.go` (110-symbol compile gate + 9 behavioral tests) and successful consumer compilation.
**Warning signs:** 90 lines of "incompatible changes" that ALL follow the pattern `changed from X to internal/runtime.X`. If any line shows a genuinely different signature or removed symbol, that IS a real breaking change.
**Verified finding:** Exactly 90 false positives observed locally: 56 type-alias changes + 34 function-signature changes (where parameter/return types are now aliases). Zero genuine incompatible changes.

### Pitfall 2: apidiff Module Cache vs Working Tree Confusion
**What goes wrong:** Running `apidiff -w old.txt github.com/amikos-tech/chroma-go-local@v0.3.4` fails because `@version` syntax is only supported by `go get`/`go install`, not by arbitrary tools.
**Why it happens:** apidiff's `-w` flag expects to be run from the directory containing the module, not with `@version` syntax.
**How to avoid:** Use `go mod download -json ... | jq -r .Dir` to get the cached directory path, then `cd` into it before running `apidiff -w`.
**Warning signs:** Error message "can only use path@version syntax with 'go get' and 'go install' in module-aware mode"

### Pitfall 3: README Project Structure Section Has Stale Paths
**What goes wrong:** The README lists `chroma.go`, `config.go`, `library.go`, `errors.go` as root implementation files, but they are now facade files with different content.
**Why it happens:** The Project Structure section was written for the pre-refactor layout.
**How to avoid:** Update the Project Structure to show the facade/internal split, making it clear which files are facade and which are implementation.

### Pitfall 4: CHANGELOG.md Date Placeholder
**What goes wrong:** Creating CHANGELOG.md with a placeholder date that gets forgotten before release.
**Why it happens:** The merge/release date is not known at implementation time.
**How to avoid:** Use `UNRELEASED` as the date, update to actual date at release time.

### Pitfall 5: CI apidiff Step Should Not Be a Hard Gate
**What goes wrong:** Making the apidiff CI step a required check causes all PRs to fail due to the type-alias false positives.
**Why it happens:** The type-alias pattern will always trigger false positives.
**How to avoid:** Make the step informational (writes to step summary, uses `::warning`) rather than failing. Future evaluation of making it a hard gate is deferred (FUTURE-02).

## Code Examples

### apidiff Export and Compare Script

```bash
#!/usr/bin/env bash
# scripts/apidiff-check.sh - Run API compatibility check against a baseline tag
set -euo pipefail

BASELINE_TAG="${1:-$(git tag -l 'v*' --sort=-v:refname | head -1)}"
MODULE="github.com/amikos-tech/chroma-go-local"

echo "Comparing current API against ${BASELINE_TAG}..."

# Export old API
OLD_DIR=$(go mod download -json "${MODULE}@${BASELINE_TAG}" | jq -r .Dir)
(cd "$OLD_DIR" && apidiff -w /tmp/apidiff_old.txt "$MODULE")

# Export new API
apidiff -w /tmp/apidiff_new.txt "$MODULE"

# Compare
echo "=== Incompatible changes ==="
apidiff -incompatible /tmp/apidiff_old.txt /tmp/apidiff_new.txt || true

# Cleanup
rm -f /tmp/apidiff_old.txt /tmp/apidiff_new.txt
```

### Updated CLAUDE.md Architecture Diagram

```
Go Package (root)                   Internal Implementation
├── chroma.go     ─── facade ───►   internal/runtime/
├── config.go         (type         ├── chroma.go      (server lifecycle)
├── embedded.go        aliases      ├── config.go      (builder options)
├── errors.go          + thin       ├── embedded.go    (embedded mode)
├── backup.go          wrappers)    ├── errors.go      (error types)
├── rebuild.go                      ├── backup.go      (backup API)
├── compaction.go                   ├── rebuild.go     (rebuild API)
├── wal_prune.go                    ├── compaction.go  (compaction API)
└── doc.go                          └── wal_prune.go   (WAL prune API)
                                    internal/library/
Rust Shim (shim/)                   ├── library.go     (FFI loading)
└── src/lib.rs ◄────────────────    ├── library_unix.go
    (chroma_* symbols)              └── library_windows.go

Java scaffold (java/)
├── core   (shared API models)
├── jna    (Java 17 fallback)
└── panama (Java 22 primary)
```

### Updated README Project Structure

```
.
├── chroma.go        # Root facade: type aliases + thin wrappers
├── config.go        # Facade: ServerConfig, ServerOption
├── embedded.go      # Facade: Embedded, EmbeddedOption, request/response types
├── errors.go        # Facade: error types and sentinel errors
├── backup.go        # Facade: BackupOption, BackupManifest
├── rebuild.go       # Facade: RebuildCollectionOption, RebuildCollectionResult
├── compaction.go    # Facade: CompactCollectionRequest (methods via type alias)
├── wal_prune.go     # Facade: WALPruneOption, WALPruneResult
├── doc.go           # Package documentation
├── compat_test.go   # Compile-time API surface gate (110 symbols + 9 behavioral)
├── internal/
│   ├── runtime/     # Server, embedded, config, backup, rebuild, compaction, WAL prune
│   └── library/     # FFI loading via purego (platform-specific)
├── java/            # Java scaffold modules (core, jna, panama)
├── shim/            # Rust FFI shim
├── examples/        # Go and Java usage examples
├── scripts/         # Dev helpers (Windows, backfill)
└── Makefile         # Build orchestration
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `go-apidiff` (joelanford) commit-based | `golang.org/x/exp/cmd/apidiff` with export files | Ongoing | Export file approach gives more control over comparison baselines |
| apidiff had type alias bugs (issue #70695) | Fixed: supports `types.Alias` via `types.Unalias` | Jan 2025 | Type aliases handled correctly for Go 1.23+; false positives in our case are by design (internal path exposure), not a bug |

## Open Questions

1. **apidiff False Positive Count Stability**
   - What we know: Currently 90 false positives. This number should be stable unless new public symbols are added.
   - What's unclear: If future Go versions improve apidiff's handling of type aliases to internal packages, the count may change.
   - Recommendation: Document the current count (90) in the apidiff check output. If the number changes unexpectedly, investigate manually.

2. **CI apidiff Caching**
   - What we know: `go mod download` will cache the baseline version. The `actions/setup-go` cache should cover this.
   - What's unclear: Whether the Go module cache for the downloaded tag version is included in the setup-go cache key.
   - Recommendation: Rely on setup-go's default caching; the download is fast (<2s) even without cache.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify (existing) |
| Config file | none (standard `go test`) |
| Quick run command | `go build ./...` (compile gate) |
| Full suite command | `make test` (builds shim + runs Go tests) |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DOCS-01 | apidiff reports no genuine breaking changes | script | `apidiff -incompatible /tmp/old.txt /tmp/new.txt` (output review) | N/A -- manual one-shot |
| DOCS-02 | README.md paths are accurate | manual | Visual inspection of updated sections | N/A |
| DOCS-03 | CLAUDE.md paths are accurate | manual | Visual inspection of updated sections | N/A |
| DOCS-04 | GO_API_SURFACE.md paths updated | manual | Visual inspection of updated references | N/A |
| COMPAT-01 | Compatibility checklist items pass | smoke | `go test -race ./... && GOOS=linux go build ./... && GOOS=windows go build ./...` | Existing tests |
| COMPAT-02 | Import path works for consumers | unit | `compat_test.go` compile gate (110 symbols) | Exists |
| COMPAT-03 | Release notes present | manual | CHANGELOG.md exists + content review | N/A -- Wave 0 creation |

### Sampling Rate
- **Per task commit:** `go build ./...` (fast compile check)
- **Per wave merge:** `make test` + `go test -race ./...`
- **Phase gate:** Full suite green + apidiff one-shot + cross-compile + visual doc review

### Wave 0 Gaps
- [ ] `CHANGELOG.md` -- new file, created as part of COMPAT-03
- [ ] CI apidiff job in `.github/workflows/ci.yml` -- new job, created as part of DOCS-01/D-02
- [ ] No new test files needed -- existing `compat_test.go` and `make test` cover validation

## Sources

### Primary (HIGH confidence)
- `golang.org/x/exp/cmd/apidiff` v0.0.0-20260312153236 -- installed and tested locally; export/compare workflow verified
- `apidiff --help` output -- flags `-w`, `-m`, `-incompatible`, `-allow-internal` documented
- Local experimental run: 90 false positives confirmed (56 type alias + 34 function signature), zero genuine breaking changes
- Consumer compilation test: `/tmp/apidiff-consumer/main.go` with `go build` -- compiles successfully against the facade
- GitHub issue golang/go#70695 (types.Alias support) -- resolved Jan 2025, COMPLETED status

### Secondary (MEDIUM confidence)
- [apidiff command documentation](https://pkg.go.dev/golang.org/x/exp/cmd/apidiff) -- published Mar 12, 2026
- [apidiff README](https://go.googlesource.com/exp/+/master/apidiff/README.md) -- type correspondence rules documented
- [joelanford/go-apidiff](https://github.com/joelanford/go-apidiff) -- GitHub Action alternative evaluated, not selected
- [Keep a Changelog](https://keepachangelog.com/) -- format specification for CHANGELOG.md

### Tertiary (LOW confidence)
- None -- all findings verified against primary sources or local experiments

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- `apidiff` installed, tested, workflow verified end-to-end
- Architecture: HIGH -- all documentation files read, change targets identified with line numbers
- Pitfalls: HIGH -- false positive count verified experimentally (90 items), module cache workflow tested, CI pattern documented

**Research date:** 2026-03-21
**Valid until:** 2026-04-21 (stable domain; apidiff and Go tooling unlikely to change)
