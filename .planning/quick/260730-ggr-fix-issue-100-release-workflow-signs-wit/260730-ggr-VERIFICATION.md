---
phase: quick/260730-ggr
verified: 2026-07-30T14:00:00Z
status: passed
score: 9/9 must-haves verified
overrides_applied: 0
---

# Quick Task 260730-ggr: Fix issue #100 (release signing identity) Verification Report

**Task Goal:** Fix issue #100 — release workflow signs with `refs/heads/main` identity instead of `refs/tags/<version>` when dispatched from `main`. The final delivered guard must compare the FULL cosign SAN identity (repo + workflow path + ref, via `EXPECTED_WORKFLOW_REF` built from `github.repository` and `env.RELEASE_REF`) against `github.workflow_ref`, not just a parsed ref suffix — this hardening was added in follow-up commit `4a5d7aa` after the initial guard (`52ea354`) was found bypassable during code review.

**Verified:** 2026-07-30T14:00:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Important Finding: SUMMARY.md Is Stale

`260730-ggr-SUMMARY.md` documents only commit `52ea354` — the **original, bypassable** guard (`SIGNING_REF="${WORKFLOW_REF##*@}"` compared against `EXPECTED_REF`). It does not mention the follow-up commit `4a5d7aa` ("close signing-identity guard bypass and consolidate expected-ref") at all, even though `4a5d7aa` is already on `main` and is what actually ships. The SUMMARY's line "Verified equal programmatically, so the two can never silently drift apart" describes the *old*, now-superseded mechanism.

This verification did **not** trust SUMMARY.md. It re-derived the guard script directly from the committed `release.yml` at `HEAD` (which is commit `4a5d7aa`, one commit ahead of what SUMMARY.md describes) and re-ran the review's own bypass proof against it.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Guard hard-fails before signing when `workflow_ref`'s ref differs from the built ref | VERIFIED | Live re-run of guard script extracted via `yq` from `release.yml`@HEAD: dispatch-from-main case exits 1 with correct `::error` annotation |
| 2 | Guard passes unchanged for tag push and dispatch-on-tag | VERIFIED | Re-run: tag-dispatch and tag-push cases both exit 0 |
| 3 | Mismatch output names both refs + prints `gh workflow run release.yml --ref <tag> -f release_tag=<tag>` remediation | VERIFIED | Captured stdout: `::error title=Signing identity mismatch::Artifacts would be signed as ... but must be ...` followed by `gh workflow run release.yml --ref v0.3.6 -f release_tag=v0.3.6` |
| 4 | No artifact is signed/uploaded/published when guard fails | VERIFIED | `yq` step-order dump: guard is step 6, immediately before `Install cosign` (7), `Sign and verify artifacts` (8), `Upload artifacts to R2` (9), `Publish GitHub release` (11); non-zero exit under `set -euo pipefail` in a GH Actions job with no `continue-on-error` skips all subsequent steps |
| 5 | `release.yml` is `actionlint` clean; guard bash is `shellcheck` clean | VERIFIED | `actionlint .github/workflows/release.yml` exit 0; guard `run:` body piped through `shellcheck -s bash -` exit 0 |
| 6 | **Critical bypass (CR-01) closed**: a branch/ref name containing `@` can no longer spoof a match | VERIFIED | Reproduced REVIEW.md's exact proof-of-bypass input (`.../release.yml@refs/heads/hotfix@refs/tags/v0.3.6` vs `EXPECTED=.../release.yml@refs/tags/v0.3.6`) against the **current** guard script — now exits **1** (was 0 against the pre-fix `52ea354` guard per REVIEW.md) |
| 7 | Guard validates full identity (repo + workflow path + ref), not just ref suffix (WR-01 closed) | VERIFIED | Reproduced REVIEW.md's fork/path-spoof input (`attacker/fork/.github/workflows/other.yml@refs/tags/v0.3.6` vs the real repo/path/ref) — exits **1**. Legitimate tag containing `@` (`v1.0.0@beta`) with matching full identity now correctly exits **0** (WR-01's flagged false-positive case also resolved as a side effect) |
| 8 | `EXPECTED_WORKFLOW_REF`/`RELEASE_REF` is a single source of truth, not independently re-typed (WR-02 closed) | VERIFIED | Workflow-level `env.RELEASE_REF` ternary appears exactly **once** in the file (line 21); both `actions/checkout` steps (`build-artifacts`, `build-java-artifacts`) and the guard's `EXPECTED_WORKFLOW_REF` all reference `${{ env.RELEASE_REF }}` — no independent copies of the ternary remain (`grep -c "format('refs/tags/{0}'"` → 1 occurrence total) |
| 9 | No unresolved debt markers (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) in the changed file | VERIFIED | `grep -iE "TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER" .github/workflows/release.yml` → no matches |

**Score:** 9/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.github/workflows/release.yml` | Contains `Verify signing identity matches released ref` step between `Build checksum manifest` and `Install cosign`, using full-identity comparison | VERIFIED | Step present at position 6 of 11 in `publish-release`; `env.EXPECTED_WORKFLOW_REF: ${{ github.repository }}/.github/workflows/release.yml@${{ env.RELEASE_REF }}` compared verbatim (`!=`) against `env.WORKFLOW_REF: ${{ github.workflow_ref }}` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| Guard step `env.WORKFLOW_REF` | `github.workflow_ref` context | `env:` block | WIRED | `grep` confirms `WORKFLOW_REF: ${{ github.workflow_ref }}`, no inlining into `run:` |
| Guard step `env.EXPECTED_WORKFLOW_REF` | workflow-level `env.RELEASE_REF` + `github.repository` | `${{ env.RELEASE_REF }}` reuse | WIRED | Single definition at workflow-level `env:` (line 17-21), consumed by both checkout `ref:` fields and the guard — confirmed byte-identical reuse, zero drift risk |
| Guard step position | `Install cosign` / `Sign and verify artifacts` / `Upload artifacts to R2` / `Publish GitHub release` | non-zero exit under `set -euo pipefail` skips subsequent job steps | WIRED | Step order confirmed via `yq`: guard (6) precedes all four signing/publish steps (7, 8, 9, 11) |

### Behavioral Spot-Checks (Guard Exit-Code Matrix, Re-Run Independently)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Dispatch from `main` (issue #100 original bug) | guard script w/ `WORKFLOW_REF=.../release.yml@refs/heads/main`, `EXPECTED_WORKFLOW_REF=.../release.yml@refs/tags/v0.3.6` | exit 1 | PASS |
| Dispatch from tag `v0.3.6` | matching full identities | exit 0 | PASS |
| Tag push `v0.3.6` | matching full identities | exit 0 | PASS |
| Dispatched on `v0.3.6`, `release_tag=v0.3.5` | mismatched | exit 1 | PASS |
| **CR-01 bypass reproduction**: branch `hotfix@refs/tags/v0.3.6` | `WORKFLOW_REF=.../release.yml@refs/heads/hotfix@refs/tags/v0.3.6` vs `EXPECTED=.../release.yml@refs/tags/v0.3.6` | exit **1** (was 0 pre-fix per REVIEW.md) | PASS |
| **WR-01 reproduction**: forked repo, renamed workflow file | `WORKFLOW_REF=attacker/fork/.github/workflows/other.yml@refs/tags/v0.3.6` vs real identity | exit 1 | PASS |
| Legitimate tag containing `@` (`v1.0.0@beta`) | matching full identities | exit 0 (no more false-positive truncation) | PASS |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER in `release.yml` | — | none |
| `260730-ggr-SUMMARY.md` | whole file | Documents pre-fix commit `52ea354` only; omits follow-up fix `4a5d7aa` that is what actually ships on `main` | INFO | Does not affect code correctness (verified independently against `HEAD`), but the task's own documentation trail is out of date and would mislead a future reader relying on SUMMARY.md alone |

**Residual, pre-existing REVIEW.md warnings not addressed by this task (out of original task scope — not re-checked as blockers):** WR-03 (guard position relative to `Normalize artifact names`/checksum steps), WR-04 (unsanitized `::error` annotation injection via `release_tag`), WR-05 (unquoted `gh workflow run` remediation string), WR-06 (stale dispatch-input description / `if:` gate not narrowed). These were flagged as warnings (not the CRITICAL finding) in REVIEW.md and are unrelated to the specific verification ask (bypass closure + full-identity comparison); they remain open in the codebase as-is. Not treated as blockers here since the task's explicit goal was closing the CR-01 bypass, which is confirmed done.

### Human Verification Required

None. All checks in this task are statically/behaviorally verifiable via `actionlint`, `shellcheck`, `yq`, and direct extraction+execution of the guard's `run:` body. The one item that genuinely cannot be verified pre-merge — live OIDC/cosign identity confirmation on an actual tagged release — was already correctly flagged in PLAN.md/SUMMARY.md as deferred, post-merge, manual, and does not block this verification since the guard's *logic* has been proven directly.

### Gaps Summary

No gaps. All 9 must-haves verified against the actual committed file at `HEAD` (commit `4a5d7aa`), including live re-execution of the exact bypass string identified as CRITICAL in code review — confirmed closed, not merely claimed closed.

---

_Verified: 2026-07-30T14:00:00Z_
_Verifier: Claude (gsd-verifier)_
