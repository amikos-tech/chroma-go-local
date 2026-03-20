---
phase: 3
slug: root-facade
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-20
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) |
| **Config file** | None (standard `go test`) |
| **Quick run command** | `go build ./...` |
| **Full suite command** | `go build ./... && go test ./internal/...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...`
- **After every plan wave:** Run `go build ./... && go test ./internal/...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | FACADE-01 | compile-gate | `go build ./...` | N/A | ⬜ pending |
| 03-01-02 | 01 | 1 | FACADE-02 | compile-gate | `go build ./examples/go/basic/` | ✅ | ⬜ pending |
| 03-01-03 | 01 | 1 | FACADE-03 | compile-gate | `go build ./...` | N/A | ⬜ pending |
| 03-01-04 | 01 | 1 | FACADE-04 | manual-only | `grep -rn 'if\|for\|switch\|select' *.go` | N/A | ⬜ pending |
| 03-01-05 | 01 | 1 | FACADE-05 | compile-gate | `go build ./examples/go/basic/` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. The `examples/go/basic/main.go` serves as the compilation gate. No additional test infrastructure needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Zero implementation logic in root files | FACADE-04 | Requires structural inspection, not functional test | Run `grep -rn 'if\|for\|switch\|select' *.go` at root — expect zero matches in facade files |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
