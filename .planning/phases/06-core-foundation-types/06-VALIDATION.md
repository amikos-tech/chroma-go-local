---
phase: 6
slug: core-foundation-types
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-22
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | JUnit 5 (already configured in core/build.gradle.kts) |
| **Config file** | `java/core/build.gradle.kts` |
| **Quick run command** | `cd java && gradle --no-daemon :core:test` |
| **Full suite command** | `cd java && gradle --no-daemon :core:test :jna:test :panama:test` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd java && gradle --no-daemon :core:test`
- **After every plan wave:** Run `cd java && gradle --no-daemon :core:test :jna:test :panama:test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 06-01-01 | 01 | 1 | FOUND-02 | unit | `gradle :core:test --tests '*ServerConfigBuilder*'` | ❌ W0 | ⬜ pending |
| 06-01-02 | 01 | 1 | FOUND-03 | unit | `gradle :core:test --tests '*EmbeddedConfigBuilder*'` | ❌ W0 | ⬜ pending |
| 06-01-03 | 01 | 1 | FOUND-04 | unit | `gradle :core:test --tests '*Result*'` | ❌ W0 | ⬜ pending |
| 06-02-01 | 02 | 2 | FOUND-05 | unit | `gradle :core:test --tests '*AbstractChromaRuntime*'` | ❌ W0 | ⬜ pending |
| 06-02-02 | 02 | 2 | FOUND-06 | integration | `gradle :jna:test :panama:test` | ✅ exists | ⬜ pending |
| 06-02-03 | 02 | 2 | FOUND-01 | build | `gradle :core:build` | ✅ exists | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `java/core/src/test/java/tech/amikos/chroma/local/core/ServerConfigBuilderTest.java` — golden YAML tests for FOUND-02
- [ ] `java/core/src/test/java/tech/amikos/chroma/local/core/EmbeddedConfigBuilderTest.java` — golden YAML tests for FOUND-03
- [ ] `java/core/src/test/java/tech/amikos/chroma/local/core/ResultDeserializationTest.java` — JSON round-trip tests for FOUND-04
- [ ] Gson and SnakeYAML dependencies added to `java/core/build.gradle.kts`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Zero JNA/Panama imports in core | FOUND-01 | Static analysis | `grep -r 'com.sun.jna\|java.lang.foreign' java/core/src/main/` must return empty |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
