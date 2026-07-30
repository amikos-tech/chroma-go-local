---
phase: quick/260730-p2k
plan: 01
subsystem: infra
tags: [github-actions, ci, protoc, supply-chain, action-pinning]

# Dependency graph
requires: []
provides:
  - All 4 protoc setup steps running on the chroma-core/setup-protoc fork (Node 24 runtime)
  - Removal of the last arduino/setup-protoc (Node 20) dependency from CI and release workflows
affects: [ci, release, build-tooling]

# Tech tracking
tech-stack:
  added: [chroma-core/setup-protoc@df9e7872 (v4c)]
  patterns:
    - "Third-party GitHub Actions pinned by full 40-char commit SHA with a human-readable version comment"

key-files:
  created: []
  modified:
    - .github/workflows/ci.yml
    - .github/workflows/release.yml

key-decisions:
  - "Pinned the fork by full 40-char commit SHA (df9e7872...) rather than the v4c tag, matching the repo's existing action-pinning convention and preventing silent re-pin"
  - "Kept the `# v4c` trailing comment so the SHA stays human-readable"
  - "Left the pre-existing shellcheck SC2129 style warning at ci.yml:313 untouched as out-of-scope"

patterns-established:
  - "Diff-scope assertion: action-swap changes verified to touch only the intended uses: lines (exactly 8 diff lines, all setup-protoc)"
  - "Structural yq validation of Actions steps, not just grep, to catch indentation damage that would orphan a with: block"

requirements-completed: [ISSUE-96]

# Metrics
duration: 4min
completed: 2026-07-30
---

# Quick Task 260730-p2k: Replace arduino/setup-protoc with chroma-core fork Summary

**Swapped all 4 protoc setup call sites to the SHA-pinned `chroma-core/setup-protoc` fork, moving protoc setup off the deprecated Node 20 Actions runtime onto Node 24 with releases-API retry/backoff.**

## Performance

- **Duration:** ~4 min
- **Tasks:** 2/2
- **Files modified:** 2

## Accomplishments

- Replaced `arduino/setup-protoc@c65c8195` with `chroma-core/setup-protoc@df9e7872eaabfd0ddfafd9e27fe77c6229bc7d22 # v4c` at all 4 call sites (ci.yml 54/124/198, release.yml 60)
- Preserved `version: "31.x"` and `repo-token: ${{ secrets.GITHUB_TOKEN }}` byte-for-byte at every site
- Verified the change is surgically scoped: exactly 4 lines changed, all setup-protoc, no collateral action-pin edits
- GitHub issue #96 fully addressed — no Node 20 runtime dependency remains from setup-protoc

## Task Commits

1. **Task 1: Swap the setup-protoc action pin at all 4 call sites** — `3cb268c` (chore)
2. **Task 2: Validate YAML, Actions syntax, and diff scope** — no commit (validation-only task; produced no file changes by design, per its own action spec "Validate the edits without changing any file")

## Files Created/Modified

- `.github/workflows/ci.yml` — 3 protoc setup steps repointed to the chroma-core fork (build matrix jobs)
- `.github/workflows/release.yml` — 1 protoc setup step repointed to the chroma-core fork (build job)

## Verification Results

| Check | Result |
|-------|--------|
| `grep` pin count — ci.yml | 3 |
| `grep` pin count — release.yml | 1 |
| Non-comment `arduino/setup-protoc` refs remaining | 0 |
| `yq` parses both workflows | PASS (`CI`, `Release Artifacts`) |
| `yq` structural step count (correct pin + `version: "31.x"` + `repo-token`) | 4 (3 ci + 1 release) |
| `actionlint` Actions schema/syntax | PASS (exit 0 with `-shellcheck=`) |
| Diff scope | 8 changed lines, 8 mention setup-protoc |
| `git diff --stat` | 2 files changed, 4 insertions(+), 4 deletions(-) |
| Line numbers unchanged (54/124/198/60) | Confirms no indentation/structural drift |

**Supply-chain check (T-p2k-01):** Re-verified the pin at execution time —
`gh api repos/chroma-core/setup-protoc/commits/df9e7872eaabfd0ddfafd9e27fe77c6229bc7d22`
resolves to commit "upgrade to httpm to kill a warning". The full-SHA pin means the
executed action code is immutable regardless of fork branch/tag movement.

## Decisions Made

None beyond the plan — followed the plan as specified. The plan's own decisions
(full-SHA pin, `# v4c` comment, replace-all edit in ci.yml) were carried out as written.

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

**1. actionlint exits non-zero on a pre-existing shellcheck style warning (out of scope, not fixed)**

- **Found during:** Task 2 verification
- **Symptom:** `actionlint` exited 1 with
  `ci.yml:313:9: shellcheck reported issue in this script: SC2129:style:15:3: Consider using { cmd1; cmd2; } >> file instead of individual redirects`
- **Analysis:** The finding is at ci.yml line 313, inside an unrelated `run:` block. My
  edits are at lines 54/124/198. Confirmed pre-existing by extracting the base file
  (`git show HEAD:.github/workflows/ci.yml`) and linting it — the identical warning
  reproduces at the same line before any edit.
- **Resolution:** Left untouched per the executor scope boundary (pre-existing warnings in
  unrelated code are not auto-fixed). Actions schema/syntax validity — the thing this plan
  actually needed to confirm — was verified clean via `actionlint -shellcheck=` (exit 0).
  This also matches the plan's own guidance that only hard schema/syntax errors block completion.
- **Deferred item:** SC2129 style cleanup at `.github/workflows/ci.yml:313` remains open for a
  future `[CLN]` change if desired. It is cosmetic and has no effect on CI behavior.

**2. Sandbox rejected the plan's compound verification one-liners**

- **Symptom:** The worktree-isolation sandbox refused the plan's `&&`-chained verify commands
  (too complex to prove they stay in-worktree) and also refused `yq eval ...` because the
  `eval` subcommand name pattern-matched shell `eval`.
- **Resolution:** Ran each assertion as a separate plain command with absolute paths, and used
  yq's implicit-eval form (`yq '<expr>' file`) instead of `yq eval`. All assertions from both
  tasks were executed and passed; only the invocation shape changed, not the checks themselves.

## Notes on Fork Adoption

The `chroma-core` fork is org-owned, 0 commits behind upstream, and its delta over
`arduino/setup-protoc` is a Node 20 → Node 24 runtime bump plus retry/backoff on the GitHub
releases API call. Token exposure is unchanged (`repo-token` was already passed to the
upstream action), so no new privilege is granted by this swap — consistent with the plan's
T-p2k-02 `accept` disposition.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

Ready. The change is CI-configuration-only; no Go, Rust, or Java source was touched, so no
build or test suite needed to run. Real-world validation happens on the next CI run, when the
fork actually executes across the ubuntu/macos/windows matrix (the fork retains upstream's
darwin/linux, x86_64/aarch_64 support).

## Self-Check: PASSED

- `.github/workflows/ci.yml` — FOUND, contains 3 chroma-core pins
- `.github/workflows/release.yml` — FOUND, contains 1 chroma-core pin
- Commit `3cb268c` — FOUND in git log
- Working tree clean after commit; no untracked or deleted files

---
*Quick task: 260730-p2k*
*Completed: 2026-07-30*
