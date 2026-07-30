---
phase: quick/260730-ggr
reviewed: 2026-07-30T09:30:50Z
depth: quick
files_reviewed: 1
files_reviewed_list:
  - .github/workflows/release.yml
findings:
  critical: 1
  warning: 6
  info: 1
  total: 8
status: issues_found
---

# Quick Task 260730-ggr: Code Review Report

**Reviewed:** 2026-07-30T09:30:50Z
**Depth:** quick
**Files Reviewed:** 1
**Status:** issues_found

## Summary

Reviewed the single changed file `.github/workflows/release.yml` (+25 lines, commit `52ea354`), focusing on the new `Verify signing identity matches released ref` guard in `publish-release`.

The guard is correctly placed (before `Install cosign`, so nothing is signed, uploaded to R2, or published on failure), uses `env:` rather than `${{ }}` interpolation inside `run:` (no direct shell-injection sink), and does hard-fail the exact scenario from issue #100 (dispatch from `main` with a tag input). `actionlint .github/workflows/release.yml` is clean. So the happy path and the reported-bug path both behave as intended.

However, the comparison logic is **bypassable**. `SIGNING_REF="${WORKFLOW_REF##*@}"` uses greedy suffix stripping on a value whose ref component may itself contain `@` — a legal git ref character. I confirmed empirically that git accepts a branch named `hotfix@refs/tags/v0.3.6` (`git check-ref-format --branch` passes) and that the guard exits **0** for `WORKFLOW_REF=.../release.yml@refs/heads/hotfix@refs/tags/v0.3.6` with `EXPECTED_REF=refs/tags/v0.3.6` — i.e. artifacts get signed with a `refs/heads/...` identity while the guard reports a match. That is the precise failure mode the step exists to prevent, so it is a BLOCKER for a control whose only job is to be unbypassable.

Two further structural weaknesses: the guard compares only the *ref* suffix, never the repository or workflow-file path that also form the cosign SAN; and `EXPECTED_REF` is a third independent copy of the checkout ternary. Note that the SUMMARY's claim that the guard and the checkout "cannot drift" / "can never silently drift apart" is not supported by the implementation — they are two unlinked string literals in different jobs, verified equal once by hand at commit time. Both are fixed by the same rewrite proposed in CR-01/WR-01/WR-02.

## Critical Issues

### CR-01: Greedy `##*@` parse lets a ref name containing `@` bypass the signing-identity guard

**File:** `.github/workflows/release.yml:271-272`
**Issue:** `SIGNING_REF="${WORKFLOW_REF##*@}"` strips through the **last** `@` in `github.workflow_ref`. Git ref names may legally contain `@` (only a bare `@` and the sequence `@{` are rejected — verified with `git check-ref-format --branch 'hotfix@refs/tags/v0.3.6'` → exit 0). Two consequences, both proven by running the committed script verbatim:

1. **Guard bypass (fail-open).** Dispatching from a branch named `hotfix@refs/tags/v0.3.6` with `release_tag=v0.3.6` yields
   `WORKFLOW_REF=amikos-tech/chroma-go-local/.github/workflows/release.yml@refs/heads/hotfix@refs/tags/v0.3.6`.
   `${WORKFLOW_REF##*@}` → `refs/tags/v0.3.6`, which equals `EXPECTED_REF`, so the guard prints
   `Signing identity ref refs/tags/v0.3.6 matches released ref refs/tags/v0.3.6.` and exits **0** — while cosign then signs with SAN `.../release.yml@refs/heads/hotfix@refs/tags/v0.3.6`. Artifacts are signed, pushed to R2 and attached to the GitHub release under a branch identity: issue #100 reproduced *through* the guard. The same trick works with a tag named `x@refs/tags/v0.3.6`. This is reachable by anyone who can create a branch and dispatch the workflow, including an operator deliberately working around the guard.
2. **False positive (fail-closed).** A legitimate tag containing `@` (e.g. `v1.0.0@beta`, matched by the `v*` trigger) truncates to `beta` and hard-fails a valid tag-push release with a misleading message: `signed as release.yml@beta but were built from refs/tags/v1.0.0@beta`.

**Fix:** Drop the substring parse entirely and compare the **full** identity string, which also closes WR-01. `github.repository` and the workflow path are fixed, known values, so an exact comparison has no parsing edge cases:

```yaml
      - name: Verify signing identity matches released ref
        shell: bash
        env:
          WORKFLOW_REF: ${{ github.workflow_ref }}
          EXPECTED_WORKFLOW_REF: ${{ github.repository }}/.github/workflows/release.yml@${{ github.event_name == 'workflow_dispatch' && format('refs/tags/{0}', github.event.inputs.release_tag) || github.ref }}
          RELEASE_TAG: ${{ github.event_name == 'workflow_dispatch' && github.event.inputs.release_tag || github.ref_name }}
        run: |
          set -euo pipefail
          if [ "${WORKFLOW_REF}" != "${EXPECTED_WORKFLOW_REF}" ]; then
            printf '::error title=Signing identity mismatch::Artifacts would be signed as %s but must be %s\n' \
              "${WORKFLOW_REF}" "${EXPECTED_WORKFLOW_REF}"
            printf 'Refusing to sign: the cosign certificate identity must match the released ref (see issue #100).\n'
            printf 'Re-run the release from the tag itself:\n'
            printf '  gh workflow run release.yml --ref %q -f release_tag=%q\n' "${RELEASE_TAG}" "${RELEASE_TAG}"
            exit 1
          fi
          printf 'Signing identity %s matches the released ref.\n' "${WORKFLOW_REF}"
```

If the suffix-only comparison is kept for some reason, at minimum use non-greedy `${WORKFLOW_REF#*@}` (first `@`) — verified to return the full `refs/heads/hotfix@refs/tags/v0.3.6` and therefore to fail closed on the bypass input.

## Warnings

### WR-01: Guard validates only the ref, not the repository or workflow-file path that also form the cosign identity

**File:** `.github/workflows/release.yml:271-279`
**Issue:** The cosign `--certificate-identity` built at lines 294 / 384 is `https://github.com/${WORKFLOW_REF}` — repo **plus** path **plus** ref. The guard checks only the ref component, so it passes for identities downstream consumers will still reject. Verified: `WORKFLOW_REF=attacker/fork/.github/workflows/other.yml@refs/tags/v0.3.6` with `EXPECTED_REF=refs/tags/v0.3.6` exits **0**. Renaming/moving `release.yml`, or a fork running the release job, silently produces a brand-new identity — exactly the "downstream must widen the allowlist" outcome the issue's acceptance criteria set out to end. The error message compounds this by hardcoding the string `release.yml@` (see IN-01) even though the real path is never inspected.
**Fix:** Use the full-identity comparison in CR-01, which pins repo, path, and ref in one exact string match.

### WR-02: `EXPECTED_REF` is a third unenforced copy of the checkout ternary — drift silently invalidates the guard

**File:** `.github/workflows/release.yml:267` (copies at `:33`, `:128`; bare-tag variants at `:151`, `:219`, `:268`, `:338`, `:429`)
**Issue:** The guard's correctness rests entirely on `EXPECTED_REF` at line 267 being identical to the `actions/checkout` `ref:` at lines 33 and 128. Nothing enforces that — they are independent literals in different jobs. If the build-job checkout expression is ever changed (e.g. to support a new dispatch input) and line 267 is not, the guard compares against a ref that was not built and passes anyway: a silent fail-open in a supply-chain control. The expression now appears 8 times in the file in two variants.
**Fix:** Hoist to workflow-level `env` (the `env` context is available in `jobs.<id>.steps[*].with`) so one definition feeds every consumer:

```yaml
env:
  RELEASE_REF: ${{ github.event_name == 'workflow_dispatch' && format('refs/tags/{0}', github.event.inputs.release_tag) || github.ref }}
  RELEASE_TAG: ${{ github.event_name == 'workflow_dispatch' && github.event.inputs.release_tag || github.ref_name }}
```

then `ref: ${{ env.RELEASE_REF }}` in both checkouts and `EXPECTED_REF: ${{ env.RELEASE_REF }}` in the guard.

### WR-03: Guard runs after the untrusted `release_tag` has already driven filesystem writes and checksums

**File:** `.github/workflows/release.yml:263` (relative to `:216-254`)
**Issue:** The guard is step 6 of `publish-release`, after `Normalize artifact names` has already used the free-form input in `mv "${f}" "${PROJECT}-${TAG}-${os}-${arch}.tar.gz"` (line 238) and after `Build checksum manifest`. A `release_tag` containing `/` or `..` therefore reaches a rename target before any validation; nothing is published (the guard aborts, and the build jobs' `checkout` of a nonexistent ref usually fails first), but the ordering means the only validation in the workflow sits downstream of the sinks it should protect. It also means the failure only surfaces after the 3-OS matrix and the Java build have completed.
**Fix:** Make the guard the **first** step of `publish-release` (immediately after `Checkout repository scripts`, before any artifact download). Better still, extract it into a tiny `preflight` job that `build-artifacts` and `build-java-artifacts` both `needs:`, so a mis-dispatched release fails in seconds instead of red-lining a full build.

### WR-04: `release_tag` is never format-validated, and reaches `::error` annotations unsanitized (workflow-command injection into the run log)

**File:** `.github/workflows/release.yml:268, 273-276`
**Issue:** `release_tag` is a free-form `type: string` input with no pattern check anywhere in the workflow. It is echoed into a workflow command: `echo "::error title=...::... but were built from ${EXPECTED_REF}"`. Verified: `EXPECTED_REF=$'refs/tags/v1\n::error::pwned'` causes the guard to emit a second, forged `::error::pwned` annotation. The same channel reaches `::stop-commands::` / `::add-mask::`, which can suppress or mask subsequent annotations — i.e. hide *other* steps' failures from whoever reviews the run. Privilege required (workflow dispatch) keeps this out of BLOCKER territory, but a log-integrity hole inside the step whose entire purpose is trustworthy release provenance is not acceptable.
**Fix:** Validate the input before use, and emit with `printf`/`GITHUB_STEP_SUMMARY` rather than interpolating untrusted text into `::` commands:

```bash
case "${RELEASE_TAG}" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) echo "::error::release_tag must look like vMAJOR.MINOR.PATCH"; exit 1 ;;
esac
```

### WR-05: Copy-pasteable remediation command interpolates untrusted input unquoted

**File:** `.github/workflows/release.yml:276`
**Issue:** `echo "  gh workflow run release.yml --ref ${RELEASE_TAG} -f release_tag=${RELEASE_TAG}"` prints a command the error text explicitly instructs an operator to run, built from an unvalidated dispatch input, with no quoting. `release_tag='v1; rm -rf ~'` renders as a runnable compound command on the operator's machine. The SUMMARY records that the plan specified `--ref "${RELEASE_TAG}"` and that the unquoted form was chosen because quoting is "cosmetic" — it is not; this is the one place the value is deliberately rendered for execution.
**Fix:** `printf '  gh workflow run release.yml --ref %q -f release_tag=%q\n' "${RELEASE_TAG}" "${RELEASE_TAG}"` (as in CR-01), and pair it with the WR-04 input validation.

### WR-06: The `workflow_dispatch` backfill path is now always-red, but its description, `if:` gate, and docs were not updated

**File:** `.github/workflows/release.yml:7-12, 193, 275-276`
**Issue:** Post-guard, dispatch is viable **only** from the tag ref — yet the input still advertises itself as `"Existing tag to release/backfill (for example v0.3.0)"`, and `if: startsWith(github.ref, 'refs/tags/') || github.event_name == 'workflow_dispatch'` (line 193) still admits dispatch from any branch. The result for the documented use case (backfill an old tag from `main`) is a guaranteed failing run after a full 3-OS matrix build, rather than a clear refusal. The printed remediation (`--ref <tag>`) also runs the workflow file **as it exists at that tag** — for pre-guard tags that is a different, unguarded workflow, so the advice does not deliver the guard's protection. Issue #100 called this out explicitly ("when the workflow at the tag is broken, releasing from `main` is the only way out without re-tagging — see #99"); the escape hatch is now closed with no replacement and no documentation of the new required procedure (no `docs/`, `README.md`, or `CHANGELOG.md` change accompanies the commit).
**Fix:** (a) Update the input `description:` to state that the workflow must be dispatched with `--ref <the same tag>`; (b) narrow the gate to `if: startsWith(github.ref, 'refs/tags/')` so a branch dispatch is skipped/refused before any build burns CI; (c) add a short "Releasing" note to `README.md` or `CHANGELOG.md` recording the required invocation and what to do when the workflow at a tag is broken (cut a new patch tag from `main`).

## Info

### IN-01: Error message hardcodes `release.yml@`, which can drift from the real workflow path

**File:** `.github/workflows/release.yml:273`
**Issue:** `Artifacts would be signed as release.yml@${SIGNING_REF}` prints a literal filename that the guard never actually reads out of `WORKFLOW_REF`. If the workflow is renamed or moved, the message reports a path that is not the one in the certificate — misleading during exactly the incident this step exists to flag.
**Fix:** Print the real value: `"${WORKFLOW_REF}"` (the CR-01 rewrite already does this).

---

_Reviewed: 2026-07-30T09:30:50Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: quick_
