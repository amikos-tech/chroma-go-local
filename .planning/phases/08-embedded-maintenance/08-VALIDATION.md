---
phase: 8
slug: embedded-maintenance
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-26
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | JUnit Jupiter 5.11.4 |
| **Config file** | `java/build.gradle.kts` (root), `java/jna/build.gradle.kts`, `java/panama/build.gradle.kts` |
| **Quick run command** | `make test-java` |
| **Full suite command** | `make test-all` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `make build-java` (compile check)
- **After every plan wave:** Run `make test-java` (full integration)
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 08-01-01 | 01 | 1 | EMNT-01 | integration | `make test-java` | ❌ W0 | ⬜ pending |
| 08-01-02 | 01 | 1 | EMNT-02 | integration | `make test-java` | ❌ W0 | ⬜ pending |
| 08-01-03 | 01 | 1 | EMNT-03 | integration | `make test-java` | ❌ W0 | ⬜ pending |
| 08-01-04 | 01 | 1 | EMNT-04 | unit (core) | Already tested in Phase 6 | ✅ | ⬜ pending |
| 08-01-05 | 01 | 1 | EMNT-05 | integration | `make test-java` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `java/jna/src/test/java/io/chroma/jna/JnaEmbeddedMaintenanceTest.java` — integration tests for EMNT-01, EMNT-02, EMNT-03, EMNT-05
- [ ] `java/panama/src/test/java/io/chroma/panama/PanamaEmbeddedMaintenanceTest.java` — integration tests for EMNT-01, EMNT-02, EMNT-03, EMNT-05

*EMNT-04 is already covered by Phase 6 core module tests.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| *None* | — | — | — |

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
