# Phase 5: Compatibility and Docs - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-21
**Phase:** 05-compatibility-and-docs
**Areas discussed:** API diff verification, Documentation refresh, Compatibility checklist, Release notes

---

## API diff verification

| Option | Description | Selected |
|--------|-------------|----------|
| One-shot go install | go install apidiff, run once, document command | |
| CI gate in workflow | Add apidiff step to ci.yml on every push | |
| Both — one-shot now + CI gate | Run one-shot for release, add CI step for ongoing protection | ✓ |

**User's choice:** Both — one-shot now + CI gate
**Notes:** None

### Follow-up: False positive handling

| Option | Description | Selected |
|--------|-------------|----------|
| Document and skip | If only alias-related diffs, document as expected, mark passed | ✓ |
| Suppress in script | Write wrapper script to filter known alias diffs | |
| Investigate alternatives | Research if apidiff handles aliases better, or find different tool | |

**User's choice:** Document and skip
**Notes:** None

### Follow-up: CI baseline

| Option | Description | Selected |
|--------|-------------|----------|
| Latest release tag | Dynamically resolve latest vX.Y.Z tag | ✓ |
| Fixed tag per branch | Hardcode v0.3.4, manually bump after each release | |

**User's choice:** Latest release tag
**Notes:** None

### Follow-up: CI trigger

| Option | Description | Selected |
|--------|-------------|----------|
| PRs to main only | Run apidiff only on PRs targeting main | ✓ |
| Every push | Run on all pushes including feature branches | |

**User's choice:** PRs to main only
**Notes:** None

---

## Documentation refresh

### README.md scope

| Option | Description | Selected |
|--------|-------------|----------|
| Invisible to users | No mention of internal/ packages | |
| Brief architecture note | Short section noting implementation in internal/, root is facade | ✓ |

**User's choice:** Brief architecture note
**Notes:** None

### GO_API_SURFACE.md

| Option | Description | Selected |
|--------|-------------|----------|
| Update file paths only | Keep content identical, update path references | ✓ |
| Rewrite for facade perspective | Describe API purely from root package perspective | |
| Remove GO_API_SURFACE.md | Redundant with pkg.go.dev, remove and link there instead | |

**User's choice:** Update file paths only
**Notes:** None

### CLAUDE.md architecture diagram

| Option | Description | Selected |
|--------|-------------|----------|
| Update existing diagram | Replace flat-root ASCII diagram with facade -> internal layout | ✓ |
| Expanded diagram + explanation | More detailed diagram plus paragraph on why split exists | |

**User's choice:** Update existing diagram
**Notes:** None

### CLAUDE.md Key Patterns

| Option | Description | Selected |
|--------|-------------|----------|
| Add facade note | Brief note about facade pattern: type aliases, thin wrappers, zero logic | ✓ |
| Skip — existing is enough | Architecture diagram already implies the facade | |

**User's choice:** Add facade note
**Notes:** None

---

## Compatibility checklist

### Checklist location

| Option | Description | Selected |
|--------|-------------|----------|
| In the PR description | Markdown checklist in v0.4.0 PR body | ✓ |
| Standalone file in .planning/ | COMPAT-CHECKLIST.md committed to repo | |
| Both | PR description + committed file | |

**User's choice:** In the PR description
**Notes:** None

### Checklist items (multi-select)

| Option | Description | Selected |
|--------|-------------|----------|
| go-apidiff clean | apidiff reports zero incompatible changes vs v0.3.4 | ✓ |
| Import path test | Verify import path resolves correctly | ✓ |
| Race detector clean | go test -race ./... passes | ✓ |
| Cross-compile passes | GOOS=linux/darwin/windows go build ./... | ✓ |

**User's choice:** All four items selected
**Notes:** None

### Manual verification

| Option | Description | Selected |
|--------|-------------|----------|
| examples/ compile check | Manually verify examples/go/basic/main.go compiles and runs | |
| No manual steps | Automated checks are sufficient | ✓ |
| pkg.go.dev preview | Check facade rendering on pkg.go.dev | |

**User's choice:** No manual steps
**Notes:** None

---

## Release notes

### Format

| Option | Description | Selected |
|--------|-------------|----------|
| GitHub release body | Standard GitHub release attached to v0.4.0 tag | |
| CHANGELOG.md entry | Add/append CHANGELOG.md in the repo | |
| Both | GitHub release body + CHANGELOG.md | ✓ |

**User's choice:** Both
**Notes:** None

### Compatibility statement

| Option | Description | Selected |
|--------|-------------|----------|
| Direct reassurance | "No breaking changes. Your existing code requires zero modifications." | ✓ |
| Detailed with rationale | Explain refactor, list what moved, confirm APIs identical | |
| Minimal | Just "Backward-compatible refactor" | |

**User's choice:** Direct reassurance
**Notes:** None

### Roadmap hints

| Option | Description | Selected |
|--------|-------------|----------|
| No roadmap hints | Keep focused on v0.4.0 | ✓ |
| Brief forward-looking note | One-liner about what the layout enables | |

**User's choice:** No roadmap hints
**Notes:** None

---

## Claude's Discretion

- Exact wording of the README architecture note
- CHANGELOG.md format (Keep a Changelog vs custom)
- apidiff CI step implementation details (job name, caching)
- Ordering of checklist items in PR description

## Deferred Ideas

- pkg.go.dev rendering check for type aliases (evaluate post-merge)
- go-apidiff as permanent CI gate full evaluation (partially addressed)
