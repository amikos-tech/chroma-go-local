---
phase: quick/260731-nkz
plan: 01
status: complete
subsystem: ci
tags: [github-actions, cloudflare, actionlint, shellcheck, yamllint, powershell]

requires:
  - phase: quick/260731-j64
    provides: Best-effort Cloudflare purge diagnostics and shared cross-platform workflow lint entry points
provides:
  - Failure-safe Cloudflare purge diagnostics with a 512-byte remote-data bound and explicit truncation marker
  - Parse-only JSON validation with exact boolean success classification
  - Conditional Go toolchain guidance and a ShellCheck 0.9-or-newer compatibility floor
  - Executable evidence that Make and PowerShell already share the same workflow-lint contract
affects: [release-workflow, contributor-tooling, windows-development, workflow-lint]

tech-stack:
  added: []
  patterns:
    - Route every handled purge failure through a one-argument helper that emits diagnostics and exits successfully
    - Keep untrusted response data in a bounded ordinary log line and fixed remediation in the annotation
    - Derive actionlint versions from the shared pin file while allowing automatic Go toolchain selection

key-files:
  created: []
  modified:
    - .github/workflows/release.yml
    - Makefile
    - scripts/dev-windows.ps1
    - README.md
    - CLAUDE.md
    - AGENTS.md

key-decisions:
  - "Treat every syntactically valid JSON value as parsed input, while requiring a 2xx object with boolean success:true for purge success."
  - "Remove the redundant Cloudflare code-1012 branch and retain existing 403 plus generic API remediation."
  - "Keep Make and PowerShell independent but equivalent; their shared pin and lint configuration already prevent meaningful drift without a new parity framework."
  - "Use Go 1.21+ with automatic toolchain switching as the normal lint path, with a local Go 1.24+ toolchain only when switching is unavailable or disabled."

patterns-established:
  - "Failure-safe diagnostics: guard sanitization and byte counting independently so neither can suppress the warning."
  - "Tooling documentation: name the shared pin file instead of copying its current numeric value."

requirements-completed:
  - NKZ-01
  - NKZ-02
  - NKZ-03
  - NKZ-04
  - NKZ-05
  - NKZ-06
  - NKZ-07
  - NKZ-08
  - NKZ-09
  - NKZ-10

duration: 13 min
completed: 2026-07-31
---

# Quick Task 260731-nkz: Release Workflow Review Follow-up Summary

Cloudflare cache purging now remains diagnostically useful across every handled failure without blocking release publication, while contributor lint guidance matches automatic Go toolchain behavior.

## Performance

- **Duration:** 13 min
- **Started:** 2026-07-31T14:14:18Z
- **Completed:** 2026-07-31T14:27:18Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Centralized purge failure status, response-file access, warning emission, and successful termination in one helper, removing repeated arguments and exit statements.
- Added locale-stable, failure-safe response sanitization, a 512-byte remote-data cap, and a fixed truncation marker without placing untrusted bytes in workflow annotations.
- Correctly classified valid JSON values such as `null` and `false` as parsed API failures while retaining exact boolean `success:true` as the only success condition.
- Aligned contributor guidance on automatic Go toolchain switching, the shared actionlint pin, and ShellCheck 0.9 or newer without changing the established Make/PowerShell lint behavior.

## Task Commits

Each task was committed atomically:

1. **Task 1: Collapse and harden the best-effort CDN purge path** — `36c7a3d` (`fix`)
2. **Task 2: Correct lint toolchain guidance while preserving proven cross-platform parity** — `6d4bd50` (`chore`)

The disposable behavior harness and this summary were intentionally not included in either task commit. The orchestrator owns the final quick-task metadata commit.

## Files Created/Modified

- `.github/workflows/release.yml` — Centralized best-effort purge termination, bounded diagnostics, parse-only JSON validation, and simplified remediation branches.
- `Makefile` — Accurate missing-Go guidance without changing the workflow-lint command sequence.
- `scripts/dev-windows.ps1` — Matching missing-Go guidance without changing the PowerShell lint implementation.
- `README.md` — User-facing Go toolchain, shared-pin, and ShellCheck compatibility guidance.
- `CLAUDE.md` — Agent-facing lint prerequisites aligned with executable behavior.
- `AGENTS.md` — Repository instructions aligned with the same lint contract and baseline.

## Decisions Made

- The response body remains ordinary log data behind a fixed prefix; only fixed remediation text enters the workflow warning annotation.
- A failed sanitizer produces `<unreadable>`, while a failed or nonnumeric byte count simply omits the truncation marker; both paths still emit exactly one diagnostic and one warning.
- Finding 5 required evidence, not new code: both lint entry points continue reading `.actionlint-version`, passing the resolved ShellCheck path with the SC2129 exception, and running repository-wide yamllint.

## TDD Evidence

- **RED:** Before implementation, `bash /tmp/260731-nkz-cache-purge-test.sh` exited 1 with 66 failed assertions. Failures covered unset variables under `set -u`, valid `null`/`false` classification, absent truncation marking and byte counting, sanitizer termination, repeated failure exits, and retained code-1012 handling.
- **GREEN:** The same disposable harness passed all 29 modeled outcomes. Every handled failure returned zero with exactly one `CDN purge diagnostic:` line and one real warning; the success case emitted neither. Retry flags, credential secrecy, annotation-injection resistance, 512-byte boundaries, fallback behavior, and publication reachability all passed.
- The plan explicitly required the harness to remain disposable, so it was not committed and no harness residue exists in the repository.

## Verification

- `bash -n /tmp/260731-nkz-cache-purge-test.sh` — passed.
- `bash /tmp/260731-nkz-cache-purge-test.sh` — passed all 29 behavior cases in GREEN and in final combined verification.
- Task 1 composite check (harness, `make lint-workflows`, plan-authored `yq -e` structural query, and workflow `git diff --check`) — passed.
- Task 2 dynamic prose/parity script — passed for all five files; the numeric actionlint pin appears only in the authoritative pin file, and all required toolchain/ShellCheck patterns are present.
- `make lint-workflows` — passed with ShellCheck 0.11.0, yamllint 1.38.0, and the actionlint module derived from `.actionlint-version`.
- `pwsh -NoProfile -File scripts/dev-windows.ps1 -Task lint-workflows` — passed with the same tool classes and shared actionlint module.
- `make lint` — passed: golangci-lint reported 0 issues, Rust clippy completed with warnings denied, and workflow/YAML lint passed.
- `make test` — passed after building the debug Rust shim; all Go package suites completed successfully. Existing runtime diagnostics and explicitly opt-in/runtime-state skips remained non-failing.
- `git diff --check 54270a00db93734371be998b91843cce6a66f094 HEAD` — passed.
- Base-to-head scope check — exactly the six declared implementation/documentation files changed.
- Diff and commit-message policy scan — passed.
- **Unavailable checks:** None.

## Deviations from Plan

None - plan executed exactly as written.

## Authentication Gates

None.

## Known Stubs

None. A targeted added-line scan found no TODO, FIXME, placeholder, coming-soon, or unavailable-data stubs. Empty purge globals are intentional preflight state, not rendered placeholders.

## Issues Encountered

- The first isolated Rust build was lengthy but completed normally.
- The initial summary self-check wrapper used a zsh read-only variable name. It made no repository changes; the check was rerun with a task-specific variable and passed.

## User Setup Required

None. Existing Cloudflare configuration remains unchanged; contributor tool prerequisites are documented for local lint execution.

## Next Phase Readiness

- Both atomic task commits are ready for the repository-required squash integration.
- No runtime API, FFI contract, CI job, pin file, YAML policy, or dependency changed.
- No implementation or verification blockers remain.

## Self-Check: PASSED

- All six modified files and this summary exist.
- Both atomic task commits are present.
- Summary frontmatter reports `status: complete`.
- Base-to-head scope and whitespace checks pass, and the disposable harness is absent from the repository.

---
*Quick task: 260731-nkz*
*Completed: 2026-07-31*
