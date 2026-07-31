---
phase: quick/260731-fqz
plan: 01
status: complete
subsystem: ci
tags: [github-actions, actionlint, shellcheck, yamllint, cloudflare, curl, jq]

requires:
  - phase: quick/260731-eho
    provides: Standalone workflow linting and retry-aware Cloudflare purge diagnostics
provides:
  - Package-install-free workflow linting with explicit tool paths and visible versions
  - One shared Make and CI lint contract for actionlint, ShellCheck, and repository-wide yamllint
  - Structurally non-blocking Cloudflare purge with fixed failure annotations
  - Corrected historical record of the ineffective yamllint apt pin
affects: [ci-workflow, release-workflow, workflow-maintenance]

tech-stack:
  added: []
  patterns:
    - Resolve runner-provided tools explicitly and assert expected CI versions
    - Use one Make target as the local and CI workflow-lint entrypoint
    - Keep expected cache-purge failures diagnostic while continue-on-error protects publication from unexpected failures

key-files:
  created:
    - .planning/quick/260731-fqz-address-ci-lint-reproducibility-and-rele/260731-fqz-SUMMARY.md
  modified:
    - .github/workflows/ci.yml
    - .github/workflows/release.yml
    - .yamllint
    - Makefile
    - .planning/STATE.md
    - .planning/quick/260731-eho-address-cdn-purge-and-ci-workflow-lint-f/260731-eho-SUMMARY.md

key-decisions:
  - "Use actionlint v1.7.11 through go run and pass the resolved ShellCheck path explicitly; CI asserts runner-provided ShellCheck 0.9.0 and yamllint 1.38.0."
  - "Treat only 2xx plus jq-confirmed success:true as a successful purge, with fixed warnings for every expected failure category."
  - "Use step-level continue-on-error in addition to handled exit-zero branches so unexpected purge failures cannot block GitHub release publication."

requirements-completed:
  - FQZ-01
  - FQZ-02
  - FQZ-03
  - FQZ-04
  - FQZ-05
  - FQZ-06
  - FQZ-07
  - FQZ-08
  - FQZ-09
  - FQZ-10

duration: 12 min
completed: 2026-07-31
---

# Quick Task 260731-fqz: CI Lint Reproducibility and Best-Effort Cache Purge Summary

CI and local linting now share an explicit actionlint/ShellCheck/yamllint contract, while Cloudflare cache purging is a small, classified, structurally non-blocking release step.

## Accomplishments

- Removed the ineffective apt installation path and made CI assert the actual runner-provided ShellCheck 0.9.0 and yamllint 1.38.0 executables.
- Added `lint-workflows` to `make lint`, pinning actionlint v1.7.11, retaining the SC2129 exception, passing ShellCheck explicitly, and linting YAML repository-wide while excluding generated `shim/target/`.
- Reduced the Cloudflare purge from multiple classifiers and temporary files to one response file, one curl result, one HTTP classification, and one `jq -e '.success == true'` success gate.
- Added fixed warnings for setup, transport, authentication/configuration, exhausted retryable HTTP, other HTTP, and API-validation failures without exposing the token or remote body in annotations.
- Corrected STATE and the EHO summary to record that preinstalled yamllint 1.38.0 remained on PATH despite the earlier intended 1.33.0 apt pin.

## Commit

- `4174a8b` — squashed implementation commit for workflow linting and cache-purge hardening

The executor created atomic task commits `cad13a3` and `67c83a5` on its isolated worktree branch; repository policy required squash-integrating them into the implementation commit above.

## Verification

- `bash /tmp/260731-fqz-cache-purge-test.sh` passed 14 cases covering missing credentials, temp-file creation, transport, 401, 403, 408, 429, 503, other HTTP, success, success:false, malformed JSON, empty JSON, and missing success. Every handled failure returned zero, publication remained reachable, and fixed annotations contained neither the bearer-token sentinel nor remote response text.
- `make lint-workflows` passed with actionlint v1.7.11, local ShellCheck 0.11.0, and yamllint 1.38.0.
- `EXPECTED_SHELLCHECK_VERSION=0.9.0 make lint-workflows` failed as expected on this machine with a clear 0.9.0-versus-0.11.0 mismatch, proving CI's version assertion is enforced.
- `make lint` passed: golangci-lint reported zero issues, Rust clippy completed with `-D warnings`, actionlint passed, and repository-wide yamllint passed.
- `make test` passed after building the debug Rust shim; the complete Go test suite passed. Existing runtime diagnostic warnings and opt-in test skips remained non-failing.
- Structural checks passed: CI contains no apt path or yamllint 1.33.0 pin; the purge step has `continue-on-error: true`; it contains one jq success gate; and removed classifier variables are absent.
- `git diff --check` passed.
- Commit inspection confirmed `4174a8b` contains exactly `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.yamllint`, and `Makefile`, with no `.planning/**` files.

## Environment Limitation

The local machine provides ShellCheck 0.11.0 rather than the ubuntu-24.04 runner's expected 0.9.0. Local linting passed with its visible version, and the explicit mismatch test proved that CI will reject drift instead of silently substituting another executable.

## Deviations from Plan

### Squash integration

The plan kept the two implementation tasks atomic in the isolated worktree. Repository policy required squash integration, so the active branch records them as one implementation commit.

## Self-Check: PASSED

- The implementation files, plan, and summary exist.
- The squashed implementation commit exists and contains only the declared implementation files.
- No deferred implementation issues or user setup remain.

---
*Quick task: 260731-fqz*
*Completed: 2026-07-31*
