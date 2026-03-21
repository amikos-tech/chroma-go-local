---
phase: 5
slug: compatibility-and-docs
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-21
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify (existing) |
| **Config file** | none (standard `go test`) |
| **Quick run command** | `go build ./...` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...`
- **After every plan wave:** Run `make test && go test -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green + apidiff one-shot + cross-compile + visual doc review
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 1 | DOCS-01 | script | `apidiff -incompatible /tmp/old.txt /tmp/new.txt` (output review) | N/A — manual one-shot | ⬜ pending |
| 05-01-02 | 01 | 1 | DOCS-01/D-02 | CI | CI workflow validates on PR | N/A — new CI job | ⬜ pending |
| 05-02-01 | 02 | 1 | DOCS-02 | manual | Visual inspection of README.md | Exists | ⬜ pending |
| 05-02-02 | 02 | 1 | DOCS-03 | manual | Visual inspection of CLAUDE.md | Exists | ⬜ pending |
| 05-02-03 | 02 | 1 | DOCS-04 | manual | Visual inspection of GO_API_SURFACE.md | Exists | ⬜ pending |
| 05-03-01 | 03 | 2 | COMPAT-01 | smoke | `go test -race ./... && GOOS=linux go build ./... && GOOS=windows go build ./...` | Existing tests | ⬜ pending |
| 05-03-02 | 03 | 2 | COMPAT-02 | unit | `go test -run TestCompat ./...` (compat_test.go) | ✅ Exists | ⬜ pending |
| 05-03-03 | 03 | 2 | COMPAT-03 | manual | CHANGELOG.md exists + content review | N/A — new file | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `CHANGELOG.md` — new file, created as part of COMPAT-03
- [ ] CI apidiff job in `.github/workflows/ci.yml` — new job, created as part of DOCS-01/D-02
- [ ] No new test files needed — existing `compat_test.go` and `make test` cover validation

*Existing infrastructure covers most phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| README.md paths accurate | DOCS-02 | Documentation content review | Verify no file paths reference old root-level implementation files |
| CLAUDE.md paths accurate | DOCS-03 | Documentation content review | Verify architecture diagram shows facade → internal/ split |
| GO_API_SURFACE.md paths updated | DOCS-04 | Documentation content review | Verify all file path references point to new locations |
| Release notes present | COMPAT-03 | Content quality check | Verify CHANGELOG.md has compatibility statement per D-13 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
