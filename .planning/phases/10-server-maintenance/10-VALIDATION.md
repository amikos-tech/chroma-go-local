---
phase: 10
slug: server-maintenance
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-27
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + Gradle JUnit 5 |
| **Config file** | Makefile (Go), java/build.gradle.kts (Java) |
| **Quick run command** | `make test` |
| **Full suite command** | `make test-all` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `make test`
- **After every plan wave:** Run `make test-all`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 10-01-01 | 01 | 1 | SMNT-01 | integration | `make test` | ❌ W0 | ⬜ pending |
| 10-01-02 | 01 | 1 | SMNT-02 | integration | `make test` | ❌ W0 | ⬜ pending |
| 10-01-03 | 01 | 1 | SMNT-03 | integration | `make test` | ❌ W0 | ⬜ pending |
| 10-01-04 | 01 | 1 | SMNT-04 | integration | `make test-all` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Server maintenance test file — stubs for SMNT-01, SMNT-02, SMNT-03
- [ ] Java server maintenance test — stubs for SMNT-04

*Existing Go test infrastructure and Java smoke test infrastructure cover framework needs.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
