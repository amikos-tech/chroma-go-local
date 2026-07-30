---
phase: quick/260730-ggr
plan: 01
subsystem: infra
tags: [github-actions, cosign, sigstore, oidc, supply-chain, release]

# Dependency graph
requires: []
provides:
  - "publish-release guard step 'Verify signing identity matches released ref' that hard-fails before cosign runs when github.workflow_ref's ref does not match the ref the artifacts were built from"
  - "Fail-loud remediation path: ::error annotation naming both refs plus a copy-pasteable `gh workflow run release.yml --ref <tag> -f release_tag=<tag>`"
  - "Closes the dispatch-from-main backfill escape hatch that produced release.yml@refs/heads/main signing identities"
affects: [release, supply-chain-verification, chroma-go-consumer-allowlist]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Signing-identity precondition guard: compare ${WORKFLOW_REF##*@} against the same actions/checkout ref ternary, so guard and checkout cannot drift"
    - "Untrusted workflow_dispatch inputs reach the shell only via env:, never inlined into run:"

key-files:
  created: []
  modified:
    - .github/workflows/release.yml

key-decisions:
  - "Guard placed immediately before 'Install cosign' rather than duplicated into the build jobs — one non-zero exit skips signing, R2 upload, CDN purge and release publish, so a single copy is sufficient (radically simple)"
  - "EXPECTED_REF reuses the checkout ternary from build-artifacts verbatim instead of a reimplementation, guaranteeing the guard compares against literally the ref that was built"
  - "No normalization/sanitization of a malformed release_tag — failing loud is the intended behavior (RESEARCH Gotcha 3)"
  - "Comment block retained to record why the guard exists (identity comes from workflow_ref, not the checkout) and that the premise holds only while signing stays inline in this file"

patterns-established:
  - "Precondition guards for supply-chain steps: verify the OIDC-derived identity matches the built ref before any artifact is signed or published"

requirements-completed: [ISSUE-100]

# Metrics
duration: 3min
completed: 2026-07-30
---

# Quick Task 260730-ggr: Fix issue #100 (release workflow signs with wrong ref identity) Summary

**`release.yml` now refuses to sign when the cosign keyless identity would not match the released ref — a 25-line `publish-release` guard comparing `${WORKFLOW_REF##*@}` against the build's checkout ternary, verified against a 4-case exit-code matrix.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-07-30T09:21:27Z
- **Completed:** 2026-07-30T09:24:10Z
- **Tasks:** 2 (1 code, 1 verification-only)
- **Files modified:** 1

## Accomplishments

- Added the step `Verify signing identity matches released ref` to `publish-release`, positioned between `Build checksum manifest` (5) and `Install cosign` (7). A non-zero exit there skips signing, R2 upload, CDN purge and the GitHub release publish — so no artifact can be signed or shipped with a mismatched identity.
- The guard derives the signing ref as `${WORKFLOW_REF##*@}` from `github.workflow_ref` (the value cosign actually turns into the certificate SAN) and compares it to `EXPECTED_REF`, which is byte-identical to the `actions/checkout` ternary in `build-artifacts`. Verified equal programmatically, so the two can never silently drift apart.
- On mismatch it emits `::error title=Signing identity mismatch::` naming both refs, references issue #100, and prints the tag-substituted `gh workflow run release.yml --ref v0.3.6 -f release_tag=v0.3.6` remediation, so an operator hitting this in CI is not left guessing.
- Proved the full exit-code matrix against the script extracted from the committed YAML (not a re-typed copy): dispatch-from-main = 1, tag-dispatch = 0, tag-push = 0, mismatched-input = 1.
- Confirmed zero `${{ }}` interpolation inside the `run:` body — the free-form `release_tag` input reaches bash only through `env:`, closing the script-injection sink (T-ggr-02).

## Task Commits

1. **Task 1: Insert the signing-identity guard into publish-release** — `52ea354` (fix)
2. **Task 2: Prove the guard's exit-code matrix locally** — no commit (verification-only task; the plan specifies read-only, "Write no new files", and no test script is committed)
3. **Follow-up: Close code-review bypass** — `4a5d7aa` (fix), see below

**Plan metadata:** handled by the orchestrator's docs commit.

## Follow-up fix: guard bypass closed (post-review)

The auto code-review (`260730-ggr-REVIEW.md`) found the `52ea354` guard bypassable: `SIGNING_REF="${WORKFLOW_REF##*@}"` strips through the *last* `@`, and git ref/branch names may legally contain `@`. A branch named e.g. `hotfix@refs/tags/v0.3.6` made the guard compare equal and exit 0 — reproducing issue #100 straight through the new guard.

Commit `4a5d7aa` replaces the suffix-parse comparison with a full cosign-SAN-identity comparison (`env.EXPECTED_WORKFLOW_REF` = `github.repository`/`.github/workflows/release.yml@`+ref, matched against `github.workflow_ref` verbatim — no substring parsing at all), and hoists the checkout ref ternary to a single workflow-level `env.RELEASE_REF` consumed by both `actions/checkout` steps and the guard (closing the "three independent copies" drift risk). This also closes the fork/renamed-workflow-path spoof (repo + workflow path are now checked, not just the ref).

Re-verified with a 12-case matrix (bypass case, `@`-containing legitimate tag, fork/path spoofs, the original 4-case matrix) — all pass. See `260730-ggr-VERIFICATION.md` for the independent re-check against `HEAD`.

Three review warnings (repo/path spoofing, duplicated `EXPECTED_REF`) were resolved by this fix; four remain open and out of scope for this task: WR-03 (guard step ordering vs. earlier filesystem-writing steps), WR-04 (`release_tag` not format-validated before reaching `::error::` annotations), WR-05 (unquoted remediation command in the printed error), WR-06 (backfill-from-branch dispatch path now permanently red; no docs update). Recommend a follow-up `[BUG]`/`[CLN]` issue for these.

## Files Created/Modified

- `.github/workflows/release.yml` — one new `shell: bash` step in `publish-release` (+25 lines, 0 deletions) with its justifying comment block. No other step, action SHA, or `if:` gate touched.

## Verification Results

| Check | Result |
|-------|--------|
| `actionlint .github/workflows/release.yml` | clean |
| Step order (`yq`, CLAUDE.md YAML mandate) | `Build checksum manifest`=5 < guard=6 < `Install cosign`=7 |
| Guard bash under `shellcheck -s bash` | clean |
| `${{` occurrences inside guard `run:` | 0 |
| `WORKFLOW_REF` sourced from `github.workflow_ref` | yes |
| `EXPECTED_REF` vs `build-artifacts` checkout `ref` | byte-identical |
| Case A — dispatch from `main`, `release_tag=v0.3.6` | **exit 1** + annotation + remediation |
| Case B — dispatch from tag `v0.3.6` | exit 0 |
| Case C — tag push `v0.3.6` | exit 0 |
| Case D — dispatched on `v0.3.6`, `release_tag=v0.3.5` | **exit 1** |
| `git diff --stat` vs base | exactly 1 file, additive only, no new files |

Threat register dispositions from the plan are satisfied: T-ggr-01 (mitigated by the guard itself), T-ggr-02 (`env:`-only input, all expansions quoted), T-ggr-03 (root cause fixed, downstream allowlist can stop growing), T-ggr-04 (accepted; recorded in the step's inline comment), T-ggr-SC (n/a — no package installs, all action SHAs unmodified).

## Decisions Made

None beyond the plan — the guard was inserted verbatim from RESEARCH.md "Proposed Guard Step (validated)" as the plan instructed ("an insertion, not a redesign"). The plan's Task 1 prose showed the remediation line with inner quotes (`--ref "${RELEASE_TAG}"`) while RESEARCH.md showed it unquoted; RESEARCH.md was followed since the plan directs verbatim copying twice, and the printed value is a bare tag name where quoting is cosmetic. Both forms satisfy the plan's verification greps.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

`actionlint .github/workflows/*.yml` (the plan's full-suite command) exits non-zero, but solely from a **pre-existing, unrelated** finding in `.github/workflows/ci.yml:313` (`SC2129` style: consider grouping redirects). Confirmed out of scope two ways: `ci.yml` is byte-identical to the plan base commit, and the same warning reproduces when linting the base version of that file in isolation. Per the executor scope boundary it was **not** fixed; logged to `deferred-items.md` with a suggested `[CLN]` follow-up. `release.yml` itself — the only file this plan changed — is `actionlint` clean.

## Deferred / Manual Follow-up

- **Post-merge, manual (inherently un-automatable pre-merge):** end-to-end identity confirmation needs a real tag plus live OIDC. On the next release, run `cosign verify-blob --bundle <artifact>.sigstore.json --certificate-identity "https://github.com/amikos-tech/chroma-go-local/.github/workflows/release.yml@refs/tags/<version>" --certificate-oidc-issuer "https://token.actions.githubusercontent.com" --use-signed-timestamps <artifact>` and confirm it passes with no allowlist widening.
- Once a guarded release ships cleanly, the downstream chroma-go `v0.3.4`/`v0.3.5` identity-allowlist exceptions can be retired (T-ggr-03).
- Releases must now be started on the tag ref (`gh workflow run release.yml --ref <tag> -f release_tag=<tag>`); dispatching from `main` is intentionally blocked. Note that backfilling an **old** tag by dispatching on that tag runs the pre-guard workflow file, where the identity is already correct anyway.

## User Setup Required

None - no external service configuration required. No new secrets, permissions, or inputs.

## Next Phase Readiness

- Change is additive, statically verified, and behaviorally proven; ready for PR. Per CLAUDE.md this must be delivered via PR — not pushed straight to `main`.
- No follow-on code work required; the remaining item is the post-release manual `cosign verify-blob` confirmation above.

## Self-Check: PASSED

| Claim | Result |
|-------|--------|
| `.github/workflows/release.yml` exists and contains the guard step at HEAD | FOUND (verified via `git show HEAD:...`) |
| Commit `52ea354` exists | FOUND (`52ea354845f99203ea8cb2b4e195544f63b9fbd0`) |
| `260730-ggr-SUMMARY.md` exists | FOUND |
| `deferred-items.md` exists | FOUND |
| Diff vs base is 1 file, +25/-0 | CONFIRMED (`1 file changed, 25 insertions(+)`) |

No missing items. (An initial self-check script reported a false negative on the commit lookup
because `set -o pipefail` combined with `grep -q`'s early exit turns `git log`'s SIGPIPE into a
non-zero pipeline status; re-verified without `pipefail` and via `git rev-parse`.)

---
*Phase: quick/260730-ggr*
*Completed: 2026-07-30*
