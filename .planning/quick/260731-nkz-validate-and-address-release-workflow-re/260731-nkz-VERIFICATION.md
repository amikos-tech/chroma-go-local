---
task: quick/260731-nkz
verified: 2026-07-31T14:35:30Z
status: human_needed
score: "8/8 must-haves verified"
commit: f9d4e030837d1588dc82744417a0377d4f14c45c
human_verification:
  - test: "Observe the next normal hosted release, or exercise the workflow in a safe non-production release context."
    expected: "The CDN purge step reaches the external service, emits either the single success line or exactly one bounded diagnostic plus one fixed warning, and the following release-publication step runs."
    why_human: "Repository checks and the behavior harness cannot prove the hosted runner, credentials, external API, and release action work together."
---

# Quick Task 260731-nkz Verification Report

**Task goal:** Validate and address release workflow review findings 1-10, implementing only confirmed useful corrections while preserving purge safety and lint behavior.

**Automated verdict:** All 8 plan must-haves and all 10 finding dispositions are supported by the current tree. One hosted external-integration check remains for human observation.

**Re-verification:** No — initial verification.

## Goal Achievement

### Observable Truths

| # | Truth | Result | Evidence |
|---|---|---|---|
| 1 | Every handled purge failure emits one warning, one status/body diagnostic, and exits successfully through the shared helper. | VERIFIED | `.github/workflows/release.yml:475-500` owns both outputs and `exit 0`. The independently rerun 29-case harness produced exactly one warning and one diagnostic for every one of its 28 handled-failure cases; its success case produced neither. |
| 2 | Unset or empty Cloudflare configuration is handled under `set -u`, without logging credential values. | VERIFIED | `.github/workflows/release.yml:503-513` uses `${CF_ZONE_ID:-}` and `${CLOUDFLARE_API_TOKEN:-}`. Unset and empty cases returned zero with `not-requested`/`<empty>` diagnostics. The credential sentinel never appeared, and the diagnostic helper does not reference either credential. |
| 3 | Remote diagnostics expose at most 512 source bytes, flatten CR/LF, and mark truncation. | VERIFIED | `.github/workflows/release.yml:481-494` applies `LC_ALL=C head -c 512`, CR/LF translation, guarded byte counting, numeric validation, and the fixed `...[truncated]` marker. Harness cases verified oversized, exact-512-byte, and multiline annotation-like bodies. |
| 4 | Sanitizer or byte-count failure cannot suppress the diagnostic or warning. | VERIFIED | `.github/workflows/release.yml:482-494` guards the full sanitizer pipeline and byte count independently. Forced `head`, `tr`, and `wc` failures all returned zero and retained one warning plus one diagnostic; sanitizer failures used `<unreadable>`. |
| 5 | Valid JSON scalars parse successfully, while only 2xx plus exact boolean `success:true` succeeds. | VERIFIED | `.github/workflows/release.yml:544-578` gates on 2xx, uses `jq empty` for parsing, and retains `jq -e '.success == true'`. Valid `null` and `false` reached generic API-failure handling, malformed input reached malformed-JSON handling, numeric `success:1` failed, and boolean `success:true` succeeded. |
| 6 | The 1012 special case and repeated post-helper exits are gone without losing retries or remediation. | VERIFIED | The extracted run body contains no 1012 query or message-qualified branch. It has exactly two explicit `exit 0` statements: helper termination and successful completion. Retry flags remain at `.github/workflows/release.yml:528-531`, and the 401, 403, retryable, catch-all, empty, malformed, and generic API messages remain distinct. The body shrank from 147 to 113 lines. |
| 7 | Contributor guidance uses the shared pin, conditional Go toolchain wording, and ShellCheck 0.9-or-newer guidance. | VERIFIED | `README.md:14-16,87,110,601-621`, `CLAUDE.md:18-20,113-120`, and `AGENTS.md:16-18,40-43` retain Go 1.21+, explain automatic switching and conditional local Go 1.24+, name `GOTOOLCHAIN=local`/older pins, and document ShellCheck 0.9 or newer. The current numeric actionlint pin is absent from all three guides. |
| 8 | Make and PowerShell preserve equivalent workflow-lint behavior without a parity framework. | VERIFIED | `Makefile:178-210` and `scripts/dev-windows.ps1:210-243` still read `.actionlint-version`, form the same module path, resolve ShellCheck, pass `-shellcheck` and SC2129, then run `yamllint -c .yamllint .`. The commit changes only one diagnostic line in each entry point. Both commands passed with identical resolved tool/module output, and no parity-only artifact was added. |

**Score:** 8/8 truths verified.

## Findings 1-10

| Finding | Result | Current-tree evidence |
|---|---|---|
| 1 | FIXED | No `.code == 1012` query or special token-scope warning remains; 403 and generic API remediation remain. |
| 2 | FIXED | All five changed guidance/entry-point files describe Go 1.21+ with automatic toolchain switching and local Go 1.24+ only when switching is unavailable or disabled. |
| 3 | FIXED | The value read from `.actionlint-version` is absent from `README.md`, `CLAUDE.md`, `AGENTS.md`, `Makefile`, and `scripts/dev-windows.ps1`; prose refers to the pin file. |
| 4 | FIXED | `warn_purge_failure` accepts one argument and reads `PURGE_CODE`/`PURGE_BODY_FILE`; only helper and success exits remain. |
| 5 | VERIFIED — NO CHANGE | Make and PowerShell still implement the same shared-pin, module, ShellCheck/SC2129, yamllint, root-directory, and command-order contract. Both live commands passed with identical tool classes and module output. No wrapper, parity test, or framework was introduced. |
| 6 | FIXED | All three contributor guides say ShellCheck 0.9 or newer and clarify that newer versions are allowed and printed versions are diagnostic rather than equality gates. |
| 7 | FIXED | Default-value expansion safely routes truly unset zone/token variables through missing-configuration handling under `set -u`. |
| 8 | FIXED | Diagnostics cap remote input at 512 bytes and add a fixed ASCII marker only when guarded byte counting proves more bytes exist. |
| 9 | FIXED | Parse classification uses `jq empty`; semantic success remains exact boolean `.success == true`. |
| 10 | FIXED | Locale-stable sanitizer and byte-count operations are guarded; forced command failures did not suppress either required output. |

## Required Artifacts

| Artifact | Existence and substance | Wiring | Result |
|---|---|---|---|
| `.github/workflows/release.yml` | Substantive 113-line purge body; no stub/debt markers. | Active `publish-release` job step immediately precedes `Publish GitHub release`; `continue-on-error: true` is retained. | VERIFIED |
| `Makefile` | Substantive `lint-workflows` target with clear preflight and complete command sequence. | Invoked directly and by the repository lint aggregate; live execution passed. | VERIFIED |
| `scripts/dev-windows.ps1` | Substantive `Lint-Workflows` function with shared-pin resolution and checked invocations. | Routed by `-Task lint-workflows`; live execution passed. | VERIFIED |
| `README.md` | Updated end-user prerequisites and lint documentation; no copied numeric pin. | Correctly names both repository lint entry points and their shared sources. | VERIFIED |
| `CLAUDE.md` | Updated agent-facing prerequisites and lint contract; no copied numeric pin. | References the actual Make/PowerShell behavior and pin file. | VERIFIED |
| `AGENTS.md` | Updated repository instructions; no copied numeric pin. | Aligns validation guidance with the executable lint targets. | VERIFIED |

## Key Link Verification

| From | To | Result | Evidence |
|---|---|---|---|
| Purge failure branches | `warn_purge_failure` | WIRED | Eleven distinct handled-failure branches call the one-argument helper; eleven unique remediation strings were counted. |
| `warn_purge_failure` | GitHub Actions logs | WIRED | Fixed ordinary-log prefix, C-locale byte handling, CR/LF flattening, 512-byte cap, fixed marker, and fixed annotation framing are present and behavior-tested. |
| Make and PowerShell | `.actionlint-version`, ShellCheck/SC2129, `.yamllint` | WIRED | Both live entry points resolved and printed the same module and tool versions, then exited zero. |
| Contributor guides | `.actionlint-version` | WIRED | All three guides name the pin file, and dynamic checks found no copied current numeric pin. |

## Safety and Control-Flow Trace

| Concern | Trace | Result |
|---|---|---|
| Credential safety | Secrets flow only into the curl authorization/request configuration; helper output uses status, bounded response data, and fixed remediation. Sentinel checks passed. | VERIFIED |
| Annotation safety | Remote data is prefixed as ordinary output and CR/LF is flattened; only fixed remediation enters `::warning`. The forged-annotation case created no additional workflow command. | VERIFIED |
| Failure reachability | Every modeled handled failure exits zero inside the helper; unexpected step failures are additionally covered by `continue-on-error: true`; publication is the immediately following step. | VERIFIED |
| Success behavior | A 2xx object with exact boolean `success:true` emits the success line and exits zero; non-object/scalar and non-boolean cases do not. | VERIFIED |

## Behavioral Checks

| Check | Result |
|---|---|
| `bash /tmp/260731-nkz-cache-purge-test.sh` | PASS — 29/29 cases; all handled failures had exit 0, one warning, and one diagnostic. |
| Structural `yq`/shell assertions | PASS — safety tokens, retry flags, `continue-on-error`, JSON commands, exit count, and 1012 removal verified. |
| Missing-Go simulations for Make and PowerShell | PASS — both failed clearly when Go was removed from `PATH` and displayed the conditional Go 1.21+/1.24+ guidance. |
| `make lint-workflows` | PASS — ShellCheck, actionlint, and repository-wide yamllint completed. |
| `pwsh -NoProfile -File scripts/dev-windows.ps1 -Task lint-workflows` | PASS — same tool classes/module and policy completed. |
| `make lint` | PASS — Go lint reported 0 issues, Rust clippy passed with warnings denied, and workflow/YAML lint passed. |
| `make test` | PASS — debug shim built and all Go package suites passed; documented opt-in/runtime-state skips remained non-failing. |
| `git diff --check` | PASS. |

## Squash Commit Scope

Commit `f9d4e030837d1588dc82744417a0377d4f14c45c` is the current `HEAD`, has one parent, and changes exactly these six declared files:

- `.github/workflows/release.yml`
- `AGENTS.md`
- `CLAUDE.md`
- `Makefile`
- `README.md`
- `scripts/dev-windows.ps1`

No implementation file outside that set, pin file, YAML policy, dependency file, runtime/FFI source, CI job definition, parity framework, or planning artifact is in the squash commit. Commit diff whitespace and repository policy scans passed.

## Requirements Coverage

| Requirement | Result | Evidence |
|---|---|---|
| NKZ-01 | SATISFIED | Finding 1 fixed. |
| NKZ-02 | SATISFIED | Finding 2 fixed. |
| NKZ-03 | SATISFIED | Finding 3 fixed. |
| NKZ-04 | SATISFIED | Finding 4 fixed. |
| NKZ-05 | SATISFIED | No-change decision corroborated by source comparison and both live entry points. |
| NKZ-06 | SATISFIED | Finding 6 fixed. |
| NKZ-07 | SATISFIED | Finding 7 fixed and behavior-tested. |
| NKZ-08 | SATISFIED | Finding 8 fixed and boundary-tested. |
| NKZ-09 | SATISFIED | Finding 9 fixed and scalar/malformed/success cases tested. |
| NKZ-10 | SATISFIED | Finding 10 fixed and forced sanitizer/count failures tested. |

## Anti-Pattern and Disconfirmation Pass

- No `TBD`, `FIXME`, or `XXX` debt markers were found in the six changed files.
- No placeholder, empty implementation, copied actionlint numeral, exact ShellCheck equality gate, or phase-related stub was found.
- The behavior harness is deliberately disposable, as required by the plan; it is verification evidence rather than a committed regression suite.
- The harness proves that the script exits in a way that makes publication reachable, but it does not execute the hosted release action. That limitation is routed to human verification below rather than treated as live-integration proof.

## Human Verification Required

### Hosted release integration

**Test:** Observe the next normal hosted release, or exercise the workflow in a safe non-production release context without changing production credentials solely for this check.

**Expected:** The purge request reaches the external API. A successful response produces the single success line; any naturally occurring handled failure produces exactly one bounded diagnostic and one fixed warning. In either case, the following release-publication step runs.

**Why human:** Local repository checks can verify shell behavior, workflow structure, and linting, but not the hosted runner, configured credentials, external API response, and release action as one end-to-end system.

## Gaps Summary

No code, artifact, wiring, requirement, or commit-scope gaps were found. Automated goal verification is complete; only hosted external-integration observation remains.

---

_Verified: 2026-07-31T14:35:30Z_
_Verifier: gsd-verifier_
