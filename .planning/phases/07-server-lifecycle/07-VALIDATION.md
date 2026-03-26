---
phase: 7
slug: server-lifecycle
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-26
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | JUnit Jupiter 5.11.4 |
| **Config file** | `java/build.gradle.kts` (parent), per-module `build.gradle.kts` |
| **Quick run command** | `cd java && CHROMA_LIB_PATH=../target/debug gradle --no-daemon :jna:test :panama:test` |
| **Full suite command** | `make test-java` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** `cd java && CHROMA_LIB_PATH=../target/debug gradle --no-daemon :core:test :jna:test :panama:test`
- **After every plan wave:** `make test-java`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 07-01-01 | 01 | 1 | SRVR-01 | integration | `gradle --no-daemon :jna:test --tests "*ServerLifecycle*"` | ❌ W0 | ⬜ pending |
| 07-01-02 | 01 | 1 | SRVR-02 | integration | `gradle --no-daemon :jna:test --tests "*ServerLifecycle*"` | ❌ W0 | ⬜ pending |
| 07-01-03 | 01 | 1 | SRVR-03 | integration | `gradle --no-daemon :jna:test --tests "*ServerLifecycle*"` | ❌ W0 | ⬜ pending |
| 07-02-01 | 02 | 1 | SRVR-01 | integration | `gradle --no-daemon :panama:test --tests "*ServerLifecycle*"` | ❌ W0 | ⬜ pending |
| 07-02-02 | 02 | 1 | SRVR-02 | integration | `gradle --no-daemon :panama:test --tests "*ServerLifecycle*"` | ❌ W0 | ⬜ pending |
| 07-02-03 | 02 | 1 | SRVR-03 | integration | `gradle --no-daemon :panama:test --tests "*ServerLifecycle*"` | ❌ W0 | ⬜ pending |
| 07-03-01 | 03 | 2 | SRVR-04 | integration | `make test-java` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `java/jna/src/test/java/.../jna/JnaServerLifecycleTest.java` — covers SRVR-01 through SRVR-04 for JNA backend
- [ ] `java/panama/src/test/java/.../panama/PanamaServerLifecycleTest.java` — covers SRVR-01 through SRVR-04 for Panama backend
- [ ] Existing smoke tests in `JnaChromaRuntimeTest` and `PanamaChromaRuntimeTest` must continue passing after retrofit

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Panama Windows DLL unload workaround | SRVR-02 | Windows-only behavior, CI runs macOS/Linux | Verify `close()` skips `arena.close()` on Windows via code review |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
