---
phase: quick/260730-pii
verified: 2026-07-30T19:30:00Z
status: passed
score: 8/8 must-haves verified (against superseded scope — see Scope Change)
overrides_applied: 0
---

# Quick Task 260730-pii: Setup-protoc availability follow-up Verification Report

**Task Goal (as delivered):** Remove the external action-repository availability dependency for `protoc` setup. The originally planned approach — vendoring `chroma-core/setup-protoc` under `.github/actions/setup-protoc` — was implemented, then **superseded during code review** by commit `bc8200c`, which installs `protoc` directly from its GitHub release at all four call sites.

**Verified:** 2026-07-30T19:30:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Scope Change: PLAN.md and SUMMARY.md Are Superseded

`260730-pii-PLAN.md` and `260730-pii-SUMMARY.md` describe the **vendored local action**, which no longer exists in the tree. Code review of that approach surfaced two blocking defects and one factual error:

1. **Backfill regression (critical).** `release.yml` supports `workflow_dispatch` with an "Existing tag to release/backfill" input; `RELEASE_REF` feeds `actions/checkout`, so the workspace is the *old tag*. A `uses: ./.github/actions/...` reference resolves against `$GITHUB_WORKSPACE` and therefore cannot resolve for any tag predating the vendoring. Confirmed by direct evidence: `.github/actions/setup-protoc/action.yml` is absent in `v0.3.5`. PLAN.md truth #2 ("callers use the local action from the checked-out repository revision") encoded this defect as an intended property.
2. **License error (critical).** PLAN.md truth #1 states the vendored action carries an "MIT license". The vendored `LICENSE` was the 674-line **GNU GPL v3**, and `arduino/setup-protoc` is GPL-3.0 per the GitHub API. The upstream `package.json` declares `"license": "MIT"`, contradicting its own `LICENSE` file; the plan adopted the permissive reading. Compounding: `.github/` is included in Go module zips (verified against the published `v0.3.3` archive on `proxy.golang.org`), so the 1.27MB GPL bundle would have been redistributed with every `go get` of a module the README declares MIT.
3. **Unauditable artifact.** `dist/index.js` was 37,407 lines of generated bundle. Its provenance *was* verified clean (sha256 `e2bdf2e1b4ae…9db44f`, byte-identical to `chroma-core/setup-protoc@df9e7872`), but UPSTREAM.md's own instruction to "review the entire generated-bundle diff" is not a performable check.

This verification does **not** trust PLAN.md or SUMMARY.md. It verifies the **delivered** state at `HEAD`.

## Goal Achievement

### Observable Truths (delivered scope)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | No third-party or vendored action code remains in the tree | VERIFIED | `.github/actions/` removed; `grep -rn "setup-protoc" .github/` → no matches |
| 2 | All four call sites install `protoc` inline, resolvable at any checked-out ref | VERIFIED | `grep -c 'Set up protoc'` → ci.yml 3, release.yml 1; each is a `run:` block in the workflow file, which on `workflow_dispatch` is read from the default branch, not the checked-out tag |
| 3 | Installed version matches what `"31.x"` previously resolved to (no silent downgrade) | VERIFIED | `31.x` → newest 31.z = **31.1** (`gh api` release list); pinned `PROTOC_VERSION: "31.1"`; executed extraction reports `libprotoc 31.1` |
| 4 | Archive integrity is verified before use | VERIFIED | Per-platform sha256 pinned and checked via `sha256sum`/`shasum` fallback. **Negative test:** substituting a bad digest exits **1** with `computed checksum did NOT match`. The superseded vendored action performed *no* integrity checking (`grep -niE "sha256\|checksum\|integrity\|digest" src/installer.ts` → no matches) |
| 5 | All three matrix platforms resolve a correct asset | VERIFIED | `run:` body extracted via `yq` and executed with `RUNNER_OS` set to macOS/Linux/Windows: all three downloaded, passed checksum, and extracted. Windows asset yields `protoc.exe`; unix extraction preserves the exec bit (`r-xr-xr-x`) |
| 6 | No `GITHUB_TOKEN` is exposed to protoc setup | VERIFIED | `grep -n "repo-token" .github/workflows/*.yml` → no matches. Direct release download requires no token |
| 7 | Workflows remain valid; new shell is lint-clean | VERIFIED | `yq` parses both; `actionlint -shellcheck=` exit 0; `actionlint` with shellcheck reports **zero** findings in the new blocks (only the pre-existing SC2129 at `ci.yml:445`, unrelated and out of scope) |
| 8 | `chroma/` runtime-data directory is ignored | VERIFIED | `.gitignore` contains `chroma/` (commit `3d3a7e3`); this is the one original PLAN.md truth that survives unchanged |

**Score:** 8/8 truths verified

### Original PLAN.md Must-Haves — Disposition

| # | Original truth | Disposition |
|---|----------------|-------------|
| 1 | Action vendored with source, bundle, lockfile, "MIT license" | SUPERSEDED — approach reverted; license claim was factually wrong (GPL-3.0) |
| 2 | Four callers use the local action from the checked-out revision | SUPERSEDED — this property *was* the backfill defect |
| 3 | Weekly read-only upstream-drift check | SUPERSEDED — `check-setup-protoc-upstream.yml` deleted; nothing to monitor once no third-party code is vendored |
| 4 | `chroma/` ignored | VERIFIED — retained |
| 5 | Prior plan records Windows/fork/bundle risks | RETAINED — `260730-p2k-PLAN.md` amendments are unaffected by the revert |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.github/workflows/ci.yml` | 3 inline protoc installs, checksum-verified | VERIFIED | Blocks at lines 53, 167, 285; `curl --fail-with-body --retry 3 --retry-all-errors --max-time 180` |
| `.github/workflows/release.yml` | 1 inline protoc install + verify step | VERIFIED | Block at line 59, followed by `Verify protoc` |
| `LICENSE` | Root MIT license matching README's claim | VERIFIED | Added in `2223396`; now the only license text in the tree outside `.git`/`.planning` |
| `.github/actions/setup-protoc/` | Absent | VERIFIED | Removed in `bc8200c` |
| `.github/workflows/check-setup-protoc-upstream.yml` | Absent | VERIFIED | Removed in `bc8200c` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| Each `Set up protoc` step | protobuf GitHub release asset | pinned URL + sha256 | WIRED | All three asset URLs return HTTP 200; digests match measured values |
| `release.yml` protoc step | backfill dispatch path | inline `run:` read from default-branch workflow file | WIRED | No `$GITHUB_WORKSPACE` dependency remains, so pre-existing tags are unaffected |
| `README.md` License section | root `LICENSE` | both declare MIT | WIRED | No competing license text anywhere in the tree |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `.github/workflows/ci.yml` | 445 | shellcheck SC2129 (style) in an unrelated pre-existing `run:` block | INFO | Pre-existing before this task; confirmed against the base file. Not in scope |
| `260730-pii-PLAN.md`, `-SUMMARY.md` | whole files | Describe the reverted vendoring approach as delivered | INFO | Does not affect code correctness (verified independently against `HEAD`); retained as the historical trail, with this section as the correction |

### Known Trade-off Accepted (not a gap)

Pinning `31.1` replaces the floating `"31.x"`, so patch updates are now manual. Dependabot cannot track a version literal inside a `run:` block, and the drift-check workflow was deliberately removed. This was raised in review and accepted: it buys reproducible builds and removes all third-party code from the protoc path. Revisit with a Renovate regex manager if protoc updates start lagging.

### Human Verification Required

The Ubuntu/macOS/Windows CI matrix must exercise the new step on a real runner. Local validation executed all three platform branches (download, checksum, extract) and ran the resulting binary on macOS, but could not execute `protoc.exe` on Windows or confirm the `cygpath`/`unzip`-vs-`7z` fallbacks against a real Git Bash environment. These are the only unproven paths; the next push exercises them.

### Gaps Summary

No gaps in the delivered scope. All 8 truths verified against the committed state at `HEAD`, including a negative test proving the checksum gate rejects a tampered archive rather than merely claiming to check it.

---

_Verified: 2026-07-30T19:30:00Z_
_Verifier: Claude (gsd-ship preflight)_
