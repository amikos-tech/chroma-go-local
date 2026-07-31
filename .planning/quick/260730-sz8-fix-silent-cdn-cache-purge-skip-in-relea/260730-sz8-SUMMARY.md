---
phase: quick/260730-sz8
plan: 01
subsystem: ci
tags: [github-actions, release, cloudflare, cdn, observability]
requires: []
provides:
  - "loud CDN purge-skip annotation in release.yml publish-release job"
affects:
  - ".github/workflows/release.yml"
tech-stack:
  added: []
  patterns:
    - "GitHub Actions ::warning:: annotation instead of a step-level if: that silently skips"
key-files:
  created: []
  modified:
    - ".github/workflows/release.yml"
decisions:
  - "Guard both credentials in the run body rather than in the step-level if:, so the step is never reported as skipped"
  - "Non-fatal exit 0 on missing credentials -- a missing repo variable must not fail a release"
  - "Full if/then/fi blocks instead of [ -z X ] && Y one-liners, which abort under set -e"
metrics:
  duration: 6min
  tasks: 1
  files: 1
  completed: 2026-07-30
---

# Quick Task 260730-sz8: Fix Silent CDN Cache Purge Skip in release.yml Summary

Made the release workflow's Cloudflare CDN purge emit a `::warning::` annotation naming the missing credential instead of silently reporting the step as "skipped" (GitHub issue #97).

## What Changed

One step in the `publish-release` job of `.github/workflows/release.yml`:

1. **Removed `if: vars.CF_ZONE_ID != ''`** — the step is now unconditional and can never be rendered as "skipped" in the Actions UI.
2. **Replaced the token-only plain-echo guard** with a guard that checks *both* `CF_ZONE_ID` and `CLOUDFLARE_API_TOKEN`, accumulates the missing names into a `MISSING` variable, and emits a single-line `::warning title=CDN cache not purged::` annotation that names the missing credential(s), states the cache was NOT purged, and lists the two URLs (`latest.json`, `releases.json`) that may serve stale metadata until the CDN TTL expires. Exits 0 — non-fatal.

The `curl` purge invocation, the step name, `shell:`, the whole `env:` block, and the step's position between "Upload artifacts to R2" and "Publish GitHub release" are all unchanged. `git diff --stat`: 1 file, 12 insertions, 2 deletions.

## Verification

**Behavior matrix** — extracted the step body with `yq`, stubbed `curl`, ran all four credential combinations. All four rows match the plan's spec exactly:

| CF_ZONE_ID | CLOUDFLARE_API_TOKEN | Result | Exit |
|---|---|---|---|
| unset | unset | one annotation naming **both** credentials | 0 |
| set | unset | annotation naming `CLOUDFLARE_API_TOKEN` only | 0 |
| unset | set | annotation naming `CF_ZONE_ID` only | 0 |
| set | set | no annotation, `curl ... purge_cache` runs as before | 0 |

Row 3 was run with the token set to the sentinel `SEKRET-TOKEN-VALUE`; the sentinel does not appear in the annotation output, confirming T-sz8-01 is mitigated.

**Static gates**
- `actionlint .github/workflows/release.yml` — clean (includes its shellcheck pass over the run body, which validates the `set -e` safety of the guard).
- `yq -e` five-predicate gate — passes (exit 0) on the fixed file, fails on the pre-fix file (`git show HEAD:...`), confirming a real red-to-green transition. See deviation below regarding the gate's exact form.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The plan's `yq -e` gate expression is unpassable as written**

- **Found during:** Task 1 verification
- **Issue:** The plan's verify command chains the five predicates with `and`:
  `(has("if") | not) and (.run | test("::warning")) and (.run | test("CF_ZONE_ID")) and ...`
  Under yq v4.53.3 this returns `false` even when every individual predicate evaluates to `true`. Evaluated separately the step yields `has_if: false, w: true, z: true, t: true, p: true` — all correct — yet the `and` chain collapses to `false`. The cause is yq's binary-operator context handling: when the right-hand operand of `and` is a traversal expression (`.run | test(...)`), the operands do not align and the result is `false`. Reversing the operand order (`(.run | test(...)) and (has("if") | not)`) returns `true`, confirming the asymmetry.
- **Proof this is a gate defect, not an artifact defect:** the plan's exact expression was run against a synthetic minimal YAML file constructed to trivially satisfy all five conditions (no `if:` key, a `run:` containing all four literals). It still returned `false` / exit 1. The gate therefore can never pass for any input under this yq version, so the plan's claim that "this gate was confirmed to exit 1 against the pre-fix file" reflects an always-false expression rather than a discriminating check.
- **Fix:** used the semantically equivalent list form, which evaluates the identical five predicates:
  ```
  yq -e '.jobs.publish-release.steps[] | select(.name == "Purge release metadata from CDN cache") | [(has("if") | not), (.run | test("::warning")), (.run | test("CF_ZONE_ID")), (.run | test("CLOUDFLARE_API_TOKEN")), (.run | test("purge_cache"))] | all' .github/workflows/release.yml
  ```
  This exits 0 on the fixed file and non-zero on the pre-fix file — a genuine red-to-green gate.
- **Files modified:** none (verification tooling only; no artifact change resulted)
- **Commit:** n/a

No other deviations. The workflow edit itself was executed exactly as written, including the plan's hard requirements: full `if/then/fi` blocks (no `&&` one-liners that would abort under `set -e`), a single physical annotation line, and no secret interpolation.

## Threat Model Compliance

| Threat ID | Disposition | Status |
|---|---|---|
| T-sz8-01 (info disclosure via annotation) | mitigate | Verified — annotation interpolates only `${MISSING}` (literal variable *names*), `${RELEASES_DOMAIN}`, `${PROJECT}`. Sentinel-token dry run confirms no value leaks. |
| T-sz8-02 (DoS of release pipeline) | mitigate | Verified — all four combinations exit 0; full `if/then/fi` blocks used, no `set -e`-unsafe one-liner. |
| T-sz8-03 (removal of step-level `if:`) | accept | Unchanged as planned — with no credentials present the step performs no network call. |
| T-sz8-SC (package installs) | accept | No package installs in this change. |

## Out of Scope (as planned)

Setting the real `CF_ZONE_ID` / `CLOUDFLARE_API_TOKEN` values and changing the Cloudflare cache-rule / TTL configuration were explicitly excluded. Neither credential is set on this repo, so the very next release run will surface this new annotation.

## Known Stubs

None.

## Commits

| Commit | Message |
|---|---|
| e22f1a0 | `fix(ci): warn loudly when CDN cache purge is skipped` |

## Self-Check: PASSED

- `.github/workflows/release.yml` exists and contains the annotation — FOUND
- Commit `e22f1a0` exists in git history — FOUND
- `actionlint` clean — PASS
- `yq -e` gate (corrected form) exits 0 — PASS
- Only `.github/workflows/release.yml` in the code commit — PASS
