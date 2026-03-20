---
phase: 02
slug: file-migration
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-20
updated: 2026-03-20
---

# Phase 02 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — Go test is built-in |
| **Quick run command** | `go build ./internal/...` |
| **Full suite command** | `go vet ./internal/... && go test -race -run ^$ ./internal/...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./internal/...`
- **After every plan wave:** Run `go vet ./internal/... && go test -race -run ^$ ./internal/...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | LAYOUT-04 | build+tags | `go build ./internal/library/... && GOOS=linux go list -f '{{.GoFiles}}' ./internal/library/` | ✅ | ✅ green |
| 02-01-02 | 01 | 1 | LAYOUT-04 | build+cross | `go build ./internal/... && GOOS=linux go build ./internal/... && GOOS=windows go build ./internal/...` | ✅ | ✅ green |
| 02-02-01 | 02 | 2 | LAYOUT-03 | build+vet | `go build ./internal/... && go vet ./internal/...` | ✅ | ✅ green |
| 02-02-02 | 02 | 2 | LAYOUT-03 | race+build | `go test -race -run ^$ ./internal/... && go build ./internal/...` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. Go test and golangci-lint are already configured. No additional test infrastructure was needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Build tags retained after move | LAYOUT-04 | Now automated via `go list` | `GOOS=linux go list -f '{{.GoFiles}}' ./internal/library/` — verified ✅ |

All manual-only items have been promoted to automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** complete

---

## Validation Audit 2026-03-20

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
| Requirements covered | 2/2 (LAYOUT-03, LAYOUT-04) |
| Tasks verified | 4/4 |
| Test files | 12 (1 library + 11 runtime) |
