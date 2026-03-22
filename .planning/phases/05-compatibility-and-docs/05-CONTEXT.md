# Phase 5: Compatibility and Docs - Context

**Gathered:** 2026-03-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Machine-verified API compatibility against the v0.3.4 baseline, documentation refresh for the new `internal/runtime/` and `internal/library/` layout, and release readiness for v0.4.0. No new features, no API changes, no structural code changes.

</domain>

<decisions>
## Implementation Decisions

### API diff verification
- **D-01:** Install `go-apidiff` via `go install golang.org/x/exp/cmd/apidiff@latest`, run one-shot against v0.3.4 tag for this release
- **D-02:** Also add a CI step in `.github/workflows/ci.yml` that runs apidiff on PRs to main, comparing against the latest release tag (dynamically resolved)
- **D-03:** CI apidiff step runs only on PRs to main — not on every push
- **D-04:** If apidiff reports false positives from type aliases (e.g., "type changed from runtime.Server to Server"), document them as expected and mark the check as passed — these aren't real breaking changes

### Documentation refresh
- **D-05:** README.md adds a brief architecture note mentioning that implementation lives in `internal/` packages while root is a facade — helps contributors who discover the repo
- **D-06:** GO_API_SURFACE.md: update file path references only (e.g., "defined in chroma.go" becomes "defined in internal/runtime/chroma.go, re-exported at root"). Content and examples stay identical.
- **D-07:** CLAUDE.md architecture diagram updated to show root facade -> internal/runtime + internal/library. Same concise style, updated paths.
- **D-08:** CLAUDE.md "Key Patterns" section gets a brief facade pattern note: type aliases, thin wrappers, zero logic rule

### Compatibility checklist
- **D-09:** Checklist lives in the PR description (markdown checklist) — no separate file
- **D-10:** Checklist items: go-apidiff clean vs v0.3.4, import path resolves correctly, race detector clean (`go test -race ./...`), cross-compile passes (linux/darwin/windows)
- **D-11:** No manual verification steps — automated checks are sufficient. examples/ already covered by `go build ./...`

### Release notes
- **D-12:** Both GitHub release body (attached to v0.4.0 tag) AND a CHANGELOG.md entry in the repo
- **D-13:** Compatibility statement: direct reassurance — "No breaking changes. Your existing code using github.com/amikos-tech/chroma-go-local requires zero modifications."
- **D-14:** No roadmap hints or forward-looking notes in release notes — keep focused on v0.4.0

### Claude's Discretion
- Exact wording of the README architecture note
- CHANGELOG.md format (Keep a Changelog vs custom)
- apidiff CI step implementation details (job name, caching)
- Ordering of checklist items in PR description

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project structure
- `.planning/ROADMAP.md` — Phase 5 goal, success criteria (DOCS-01 through DOCS-04, COMPAT-01 through COMPAT-03)
- `.planning/REQUIREMENTS.md` — DOCS-01 (go-apidiff), DOCS-02 (README), DOCS-03 (CLAUDE.md), DOCS-04 (GO_API_SURFACE.md), COMPAT-01 (checklist), COMPAT-02 (import path), COMPAT-03 (release notes)
- `.planning/PROJECT.md` — Core value (backward-compatible import path), constraints

### Prior phase context
- `.planning/phases/03-root-facade/03-CONTEXT.md` — Facade file organization, wrapper function strategy (D-04: thin wrappers), type alias approach
- `.planning/phases/04-build-and-test/04-CONTEXT.md` — compat_test.go design (110 symbols + 9 behavioral tests), lint fixes, CI validation

### Docs to update (source of truth for current state)
- `README.md` — Current user-facing documentation, needs architecture note added
- `CLAUDE.md` — Current contributor/AI documentation, needs architecture diagram and facade pattern note
- `GO_API_SURFACE.md` — Current API reference, needs file path references updated
- `.github/workflows/ci.yml` — CI workflow where apidiff step will be added

### Codebase (facade files that define current API surface)
- `chroma.go`, `config.go`, `embedded.go`, `errors.go`, `backup.go`, `rebuild.go`, `compaction.go`, `wal_prune.go`, `doc.go` — Root facade files
- `internal/runtime/` — Implementation package (8 files)
- `internal/library/` — FFI loading package (4 files)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `compat_test.go` at root: 110-symbol compile gate + 9 behavioral smoke tests — already validates API surface (Phase 4)
- `examples/go/basic/main.go` — existing consumer example that exercises the facade
- `.github/workflows/ci.yml` — existing 3-OS matrix CI workflow to extend with apidiff step

### Established Patterns
- CI workflow uses `actions/setup-go`, `golangci-lint-action`, `go test -v ./...`
- Root package named `chroma`, imports `internal/runtime`
- Type aliases (`type X = runtime.X`) and thin wrapper functions for all re-exports

### Integration Points
- CI workflow needs new apidiff job/step for PRs to main
- README.md architecture section fits after "Building" and before "Java Scaffold"
- CLAUDE.md architecture diagram at line ~17, Key Patterns section at line ~33

</code_context>

<specifics>
## Specific Ideas

- Defense-in-depth: compile gate (Phase 4) + behavioral smoke (Phase 4) + go-apidiff (Phase 5) together provide comprehensive regression protection
- CI apidiff uses latest release tag dynamically — no manual tag bumping after each release

</specifics>

<deferred>
## Deferred Ideas

- pkg.go.dev rendering check for type aliases pointing to internal/ paths — noted as concern in STATE.md, can be evaluated post-merge on a staging branch (FUTURE-01 in REQUIREMENTS.md)
- go-apidiff as permanent CI gate evaluation (FUTURE-02 in REQUIREMENTS.md) — partially addressed by D-02, full evaluation deferred

</deferred>

---

*Phase: 05-compatibility-and-docs*
*Context gathered: 2026-03-21*
