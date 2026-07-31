---
phase: quick/260731-eho
plan: 01
status: complete
subsystem: ci
tags: [github-actions, cloudflare, curl, actionlint, shellcheck, yamllint]

requires:
  - phase: quick/260731-dgm
    provides: Initial non-fatal Cloudflare purge diagnostics and workflow linting
provides:
  - Retry-safe Cloudflare body and HTTP-status capture
  - Distinct transport/HTTP, API rejection, and response-validation diagnostics
  - Standalone pinned workflow-lint CI job with ShellCheck enabled
  - Correct historical AND-list and static-analysis explanations
affects: [release-workflow, ci-workflow, workflow-maintenance]

tech-stack:
  added: [repository-local yamllint configuration]
  patterns:
    - Capture curl response body and stderr in separate temporary files
    - Validate HTTP status and JSON syntax before requiring boolean success
    - Run workflow linting independently of build matrices

key-files:
  created:
    - .yamllint
  modified:
    - .github/workflows/release.yml
    - .github/workflows/ci.yml
    - .planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-PLAN.md

key-decisions:
  - "Treat status 000 and every non-2xx response as transport/HTTP failure; only 2xx JSON with boolean success:true is success."
  - "Keep response text in compact ordinary logs and keep Actions annotations fixed."
  - "Pin workflow linting to ubuntu-24.04, actionlint v1.7.11, and Ubuntu yamllint 1.33.0-1."

requirements-completed:
  - EHO-01
  - EHO-02
  - EHO-03
  - EHO-04
  - EHO-05
  - EHO-06
  - EHO-07
  - EHO-08
  - EHO-09

duration: 30 min
completed: 2026-07-31
---

# Quick Task 260731-eho: CDN Purge and Workflow Lint Findings Summary

Cloudflare retries now evaluate only the final body and status, while a standalone pinned CI job enforces actionlint, ShellCheck, and yamllint across every workflow.

## Accomplishments

- Replaced combined curl output with temporary body/stderr files and a guarded final HTTP-code assignment.
- Preserved retries for transient failures while proving a permanent HTTP 403 is attempted once.
- Added distinct fixed annotations for transport/HTTP failure and exact `success:false`, plus accurate empty, malformed, missing-field, wrong-type, wrong-top-level-type, and unavailable-jq diagnostics.
- Moved workflow linting out of the OS build matrix, pinned its Linux runner and yamllint package, and restored embedded ShellCheck analysis with only SC2129 ignored.
- Moved the yamllint policy to root `.yamllint` and corrected the historical plan wherever it overstated immediate errexit or static-linter guarantees.

## Commit

- `c3a0e3a` — squashed implementation commit for both plan tasks

The executor created atomic task commits `8f4c54c` and `979cc01` on its isolated worktree branch; repository policy required squashing them into the implementation commit above.

## Files Created/Modified

- `.github/workflows/release.yml` — captures the final retry body/status and classifies purge failures without blocking publication.
- `.github/workflows/ci.yml` — adds an independent `ubuntu-24.04` workflow-lint job with pinned tools and version diagnostics.
- `.yamllint` — stores the repository's shared workflow YAML policy.
- `.planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-PLAN.md` — corrects AND-list, errexit, actionlint, and ShellCheck claims.

## Verification

- `bash /tmp/260731-eho-cdn-purge-test.sh` passed all ten cases. The transient 503→200 case made two requests and ignored the first HTML body; HTTP 403 made one request; all modeled failures reached `PUBLISH_REACHABLE`; and the bearer-token sentinel never appeared.
- `actionlint -ignore 'SC2129' .github/workflows/*.yml` passed with embedded ShellCheck analysis enabled.
- `yamllint -c .yamllint .github/workflows/*.yml` passed.
- Equivalent yq v4.53.3 structural assertions returned `true` for both the release capture contract and standalone lint-job contract.
- `make lint` passed with zero golangci-lint issues and Rust clippy clean under `-D warnings`.
- `make test` built the debug shim and passed the full Go test suite. Expected runtime warnings and opt-in test skips remained non-failing.
- `git diff --check` passed, and the implementation commit contains exactly the four declared files.

## Deviations from Plan

### Auto-fixed verification syntax

Two plan-authored yq expressions were incompatible with installed yq v4.53.3 because of pipeline precedence and unsupported jq-style `index()`. Equivalent variable-based and `any_c(...)` queries passed without changing implementation coverage.

### Corrected jq status classification during merge review

The executor's first version treated every jq gate status other than 1 as malformed JSON. jq can also return 5 when valid JSON has an incompatible top-level type. Final parsing now checks JSON syntax separately, so arrays, strings, and numbers reach the intended `not-object` diagnostic. The disposable harness was extended with this case and all ten cases passed.

## Self-Check: PASSED

- The four implementation files, plan, and summary exist.
- Implementation commit `c3a0e3a` is present.
- No deferred implementation issues or user setup remain.

---
*Quick task: 260731-eho*
*Completed: 2026-07-31*
