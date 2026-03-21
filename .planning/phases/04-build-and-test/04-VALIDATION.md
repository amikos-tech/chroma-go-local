---
phase: 4
slug: build-and-test
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-21
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + stretchr/testify v1.11.1 |
| **Config file** | None (standard go test) |
| **Quick run command** | `make test` |
| **Full suite command** | `make test-all` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./... && golangci-lint run ./...`
- **After every plan wave:** Run `make test && make lint`
- **Before `/gsd:verify-work`:** Full suite must be green (`make test-all` + cross-compile)
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | BUILD-03 | lint | `golangci-lint run ./...` | ✅ | ⬜ pending |
| 04-01-02 | 01 | 1 | TEST-01 | verification | `go test ./internal/...` | ✅ | ⬜ pending |
| 04-01-03 | 01 | 1 | BUILD-01 | integration | `make test && make lint` | ✅ | ⬜ pending |
| 04-02-01 | 02 | 1 | TEST-02, TEST-03 | compile + smoke | `go test -v -run TestCompat .` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 1 | TEST-04 | integration | `make test` | ✅ | ⬜ pending |
| 04-03-01 | 03 | 2 | BUILD-04 | build | `GOOS=linux go build ./... && GOOS=darwin go build ./... && GOOS=windows go build ./...` | N/A | ⬜ pending |
| 04-03-02 | 03 | 2 | BUILD-02 | verification | Visual inspection of ci.yml | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `compat_test.go` — API surface gate + behavioral smoke tests (covers TEST-02, TEST-03)
- No framework install needed — Go testing and testify already available

*Existing infrastructure covers most phase requirements. Only compat_test.go is net-new.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| CI workflow structure unchanged | BUILD-02 | Structural diff, not runtime | `git diff .github/workflows/ci.yml` should show no changes |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
