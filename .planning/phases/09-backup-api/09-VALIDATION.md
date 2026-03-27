---
phase: 9
slug: backup-api
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-27
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | JUnit 5 (JNA + Panama modules) |
| **Config file** | `java/jna/build.gradle.kts`, `java/panama/build.gradle.kts` |
| **Quick run command** | `make test-java` |
| **Full suite command** | `make test-all` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `make test-java`
- **After every plan wave:** Run `make test-all`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 09-01-01 | 01 | 1 | BKUP-01 | unit | `make test-java` | ❌ W0 | ⬜ pending |
| 09-01-02 | 01 | 1 | BKUP-02 | unit | `make test-java` | ❌ W0 | ⬜ pending |
| 09-01-03 | 01 | 1 | BKUP-03 | unit | `make test-java` | ❌ W0 | ⬜ pending |
| 09-01-04 | 01 | 1 | BKUP-04 | integration | `make test-java` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Backup test stubs in JNA and Panama test modules
- [ ] Test fixtures for temp directory seeding with sentinel files

*Existing test infrastructure (JUnit 5, @TempDir, EmbeddedConfigBuilder, ServerConfigBuilder) covers framework needs.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
