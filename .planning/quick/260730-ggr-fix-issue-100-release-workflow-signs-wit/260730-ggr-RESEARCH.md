# Quick Task 260730-ggr: Fix issue #100 (release workflow signs with wrong ref identity) - Research

**Researched:** 2026-07-30
**Domain:** GitHub Actions workflow contexts + Sigstore/cosign keyless identity
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Fix approach**
- Add a fail-loud CI guard to `release.yml`. Do NOT restructure into a reusable workflow (`workflow_call`) and do NOT rely on a process-only/documentation-only fix. The workflow itself must refuse to proceed when the signing identity would not match the released ref.

**Guard placement and behavior**
- Insert the check immediately before the cosign/signing step (around `.github/workflows/release.yml:265,289` where `WORKFLOW_REF`/`IDENTITY` are used).
- Compare the ref embedded in `github.workflow_ref` against the ref that was actually checked out (on `workflow_dispatch`, that's `refs/tags/${{ github.event.inputs.release_tag }}`; on a tag-push trigger it's `github.ref`).
- On mismatch, hard-fail the job (non-zero exit) with a clear error message pointing at the correct dispatch invocation (`gh workflow run release.yml --ref <tag> -f release_tag=<tag>`) — do not sign or publish artifacts.

### Claude's Discretion
- Exact bash/shell implementation of the ref comparison (e.g. string comparison of `github.workflow_ref` suffix vs. expected `refs/tags/<tag>`), as long as it fails loud on mismatch and passes for the correct tag-based invocation.
- Whether to add a short comment/test note in the workflow explaining why the guard exists.

### Deferred Ideas (OUT OF SCOPE)
- `workflow_call` / reusable-workflow restructuring.
- Documentation-only or process-only fixes.
</user_constraints>

## Summary

The bug is fully explained by one fact: the cosign certificate SAN URI is built from the OIDC
`job_workflow_ref` claim, which for a non-reusable workflow equals `github.workflow_ref` — the ref the
**workflow file** was loaded from. `actions/checkout` at `release.yml:33` overrides the ref for the
*build*, but nothing can override the ref baked into the OIDC token. So dispatch-from-`main` builds tag
content and signs it `release.yml@refs/heads/main`. The only way to get a `refs/tags/<v>` identity is to
start the run on the tag ref itself. [VERIFIED: docs.github.com + sigstore/fulcio oid-info.md]

Dispatching on a tag ref is fully supported and **already proven in this repo**: run `22517529944`
(`v0.3.0` backfill) was `event: workflow_dispatch` with `headBranch: v0.3.0`, and at `v0.3.0` the
`publish-release` job gate was the bare `if: startsWith(github.ref, 'refs/tags/')` — it published, which
proves `github.ref == refs/tags/v0.3.0` on a tag dispatch. [VERIFIED: gh run list / git show v0.3.0]

Therefore the guard is purely a comparison of two values both available as expressions, needs no new
inputs or permissions, and is ~12 lines of bash. A candidate patch was written and validated locally:
`actionlint` clean, `yq` parses, `shellcheck` clean, and all four behavioral cases (dispatch-from-main,
dispatch-from-tag, tag-push, tag-dispatch-with-mismatched-input) produce the correct exit codes.

**Primary recommendation:** Insert one `shell: bash` step named **"Verify signing identity matches
released ref"** immediately before `- name: Install cosign` (currently `.github/workflows/release.yml:256`)
in the `publish-release` job. Compare `${WORKFLOW_REF##*@}` against the same ternary expression already
used by the checkout steps. Pass all values via `env:` — never inline `${{ github.event.inputs.* }}` into
the `run:` body.

## Root-Cause Chain (verified)

| Step | Evidence | Confidence |
|------|----------|-----------|
| `github.workflow_ref` = `owner/repo/.github/workflows/f.yml@<ref>` | "The ref path to the workflow. For example, `octocat/hello-world/.github/workflows/my-workflow.yml@refs/heads/my_branch`." [CITED: docs.github.com/en/actions/reference/workflows-and-actions/contexts] | HIGH |
| cosign cert SAN comes from `job_workflow_ref`, not `ref` | Fulcio OID map: `1.3.6.1.4.1.57264.1.9` Build Signer URI = `server_url + job_workflow_ref` [CITED: github.com/sigstore/fulcio/blob/main/docs/oid-info.md] | HIGH |
| `job_workflow_ref == workflow_ref` for non-reusable workflows | `job_workflow_ref` is documented as "For jobs using a reusable workflow, the ref path to the reusable workflow." Release logic lives inline in `release.yml`, so they coincide — confirmed by the decoded `v0.3.5` bundle in issue #100 showing `release.yml@refs/heads/main`. [VERIFIED: docs.github.com/en/actions/reference/security/oidc + issue #100 cert dump] | HIGH |
| No expression exists to override the signing ref | There is no `github.job_workflow_ref` context property; `checkout` only affects the working tree. [VERIFIED: contexts reference — property absent] | HIGH |
| `workflow_dispatch` accepts a tag as `ref` | "The git reference for the workflow. The reference can be a branch or tag name." [CITED: docs.github.com/en/rest/actions/workflows] | HIGH |
| Tag dispatch sets `github.ref = refs/tags/<v>` | Run `22517529944` dispatched on `v0.3.0` passed a bare `startsWith(github.ref, 'refs/tags/')` gate. [VERIFIED: gh run list --workflow=release.yml; git show v0.3.0:.github/workflows/release.yml:109] | HIGH |

Why this repo drifted to dispatch-from-main: the tag-push runs for both `v0.3.4` and `v0.3.5` **failed**
(`gh run list` shows `push`/`v0.3.4` and `push`/`v0.3.5` with `conclusion: failure`), so both were
backfilled by dispatch from `main`. The guard closes that escape hatch deliberately. [VERIFIED: gh run list]

## Exact Insertion Point

`publish-release` job step order today (verified via `yq '.jobs.publish-release.steps | map(.name)'`):

| # | Step name | Line |
|---|-----------|------|
| 1 | Checkout repository scripts | 200 |
| 2 | Download artifact bundles | 203 |
| 3 | Download Java artifact bundles | 210 |
| 4 | Normalize artifact names | 216 |
| 5 | Build checksum manifest | 242 |
| — | **← INSERT GUARD HERE** | **before 256** |
| 6 | Install cosign | 256 |
| 7 | Sign and verify artifacts | 262 (`WORKFLOW_REF` at 265, `IDENTITY` at 269) |
| 8 | Upload artifacts to R2 | 306 (second `WORKFLOW_REF` at 316, second `IDENTITY` at 359) |
| 9 | Purge release metadata from CDN cache | 381 |
| 10 | Publish GitHub release | 401 |

**One guard covers both signing sites.** `WORKFLOW_REF` is consumed twice (steps 7 and 8), but a
non-zero exit at the guard fails the job and every later step is skipped — including the GitHub release
publish at 401 and the R2 upload at 306. No second guard needed. [VERIFIED: Actions default step
behavior — steps without `if:` do not run after a failure]

**On placing it earlier (focus item 2):** placing it as the first step of `publish-release` saves only the
artifact-download seconds, because `publish-release` has `needs: [build-artifacts, build-java-artifacts]`
— all ~3-OS matrix build minutes are already spent before this job starts. Moving the guard into the
build jobs would save real runner time but requires duplicating it across two jobs, which conflicts with
CLAUDE.md's "radically simple" rule and with the locked "immediately before the cosign step" decision.
Recommend the single placement above; see Open Questions if fail-fast matters more than simplicity.

## Proposed Guard Step (validated)

Insert verbatim before line 256 (`- name: Install cosign`):

```yaml
      # Guard for issue #100: cosign derives the certificate identity from
      # github.workflow_ref (the ref the workflow FILE was loaded from), not the
      # ref that actions/checkout pulled. Dispatching from main therefore signs
      # tag content with a refs/heads/main identity, forcing downstream consumers
      # to widen their cosign allowlist. Refuse to sign on mismatch.
      # NOTE: this holds only while the signing job lives in this file — if it
      # ever moves to a reusable workflow the identity follows job_workflow_ref.
      - name: Verify signing identity matches released ref
        shell: bash
        env:
          WORKFLOW_REF: ${{ github.workflow_ref }}
          EXPECTED_REF: ${{ github.event_name == 'workflow_dispatch' && format('refs/tags/{0}', github.event.inputs.release_tag) || github.ref }}
          RELEASE_TAG: ${{ github.event_name == 'workflow_dispatch' && github.event.inputs.release_tag || github.ref_name }}
        run: |
          set -euo pipefail
          SIGNING_REF="${WORKFLOW_REF##*@}"
          if [ "${SIGNING_REF}" != "${EXPECTED_REF}" ]; then
            echo "::error title=Signing identity mismatch::Artifacts would be signed as release.yml@${SIGNING_REF} but were built from ${EXPECTED_REF}"
            echo "Refusing to sign: the cosign certificate identity must match the released ref (see issue #100)."
            echo "Re-run the release from the tag itself:"
            echo "  gh workflow run release.yml --ref ${RELEASE_TAG} -f release_tag=${RELEASE_TAG}"
            exit 1
          fi
          echo "Signing identity ref ${SIGNING_REF} matches released ref ${EXPECTED_REF}."
```

Design notes:
- `EXPECTED_REF` is the **identical ternary** already used at lines 33 and 128 for `actions/checkout`, so
  the guard compares against literally the ref that was built — no independent reimplementation to drift.
- `RELEASE_TAG` mirrors the ternary at lines 151/219/313 (`github.ref_name` fallback) so the remediation
  command is copy-pasteable.
- `${WORKFLOW_REF##*@}` strips through the **last** `@`. Safe: GitHub owner/repo names permit only
  alphanumerics, `-`, `_`, `.` — no `@` — and the workflow path is `.github/workflows/release.yml`.
  Do **not** use `%%@*` (that yields the repo/path prefix).
- `set -euo pipefail` matches the existing house style (8 occurrences in this file).
- `::error title=...::` renders an annotation on the run summary rather than burying the reason in logs.

## Local Validation Performed

| Check | Command | Result |
|-------|---------|--------|
| Workflow lints | `actionlint .github/workflows/release.yml` (on patched copy) | clean |
| YAML parses / step order | `yq '.jobs.publish-release.steps \| map(.name)'` | guard lands between "Build checksum manifest" and "Install cosign" |
| Shell lints | `shellcheck -s bash guard.sh` | clean |
| Case A — dispatch from `main`, `release_tag=v0.3.6` | `WORKFLOW_REF=...@refs/heads/main EXPECTED_REF=refs/tags/v0.3.6` | **exit 1** with actionable message |
| Case B — dispatch from tag `v0.3.6` | `WORKFLOW_REF=...@refs/tags/v0.3.6 EXPECTED_REF=refs/tags/v0.3.6` | exit 0 |
| Case C — tag push `v0.3.6` | same values as B | exit 0 |
| Case D — dispatched on `v0.3.6` but `release_tag=v0.3.5` | `EXPECTED_REF=refs/tags/v0.3.5` | **exit 1** (bonus catch) |

Case D is a free extra: the guard also catches the "dispatched from the wrong tag" mistake, where the
build would check out `v0.3.5` while signing as `v0.3.6`.

## Gotchas

### 1. Never inline `github.event.inputs.release_tag` into a `run:` body
`release_tag` is free-form user input. Inlining `${{ github.event.inputs.release_tag }}` inside a `run:`
block is a script-injection sink (GitHub's documented untrusted-input class). Pass it through `env:` and
reference `"${RELEASE_TAG}"` — which is what lines 218-220 and 264-265 already do correctly.
[CITED: docs.github.com — Security hardening for GitHub Actions, script injection]

### 2. `github.event.inputs` is null outside `workflow_dispatch`
On a tag push, `github.event.inputs.release_tag` evaluates to null and `format('refs/tags/{0}', null)`
would produce a bare `refs/tags/`. The `github.event_name == 'workflow_dispatch' && ... || ...` ternary
short-circuits before `format` runs, which is why the existing lines 33/128/151 use exactly this shape.
Reuse it rather than inventing a new expression. [VERIFIED: pattern already in-repo and shipping]

### 3. A malformed `release_tag` will now hard-fail (intended)
If someone types `refs/tags/v0.3.6` into the input, `EXPECTED_REF` becomes
`refs/tags/refs/tags/v0.3.6` and the guard fails even on a correct tag dispatch. That is the desired
fail-loud behavior — but note the same malformed input already breaks the checkout at line 33 and the
artifact naming at line 238, so the guard just surfaces it earlier. Do not add normalization/sanitization
logic; that is scope creep beyond the locked decision.

### 4. Tag dispatch runs the workflow file *as it exists at that tag*
This is the caveat issue #100 flags, and it has a benign consequence worth knowing:
- Backfilling an **old** tag by dispatching on that tag runs the old `release.yml`, which has no guard —
  but `workflow_ref` is already `refs/tags/<v>`, so the identity is correct anyway. Verified that
  `v0.3.3`, `v0.3.4`, `v0.3.5` all carry a `workflow_dispatch` trigger with the `release_tag` input, so
  this path works today. [VERIFIED: git show <tag>:.github/workflows/release.yml]
- Backfilling an old tag by dispatching from `main` runs the new guarded file → correctly blocked.

### 5. Tag dispatch also changes which `scripts/` version runs (pre-existing, not new)
The `publish-release` checkout at line 200-201 has **no** `ref:`, so it checks out `github.ref`. Under
tag dispatch that is the tag, meaning `scripts/build_releases_index.sh` (invoked at line 351) comes from
the tag rather than from `main`. This was already the behavior for every normal tag-push release; only
the dispatch-from-`main` path was the anomaly. Verified the script exists at `v0.3.3`/`v0.3.4`/`v0.3.5`
and its CLI (`--existing --output --project --version --date --max`) is unchanged since `v0.3.1`
(single commit `0ddd5fc`). No action needed. [VERIFIED: git cat-file -e + git log]

### 6. Re-runs preserve the original ref
Re-running a failed run reuses the triggering event's ref, so `github.workflow_ref` does not silently
become `refs/heads/main` on a re-run of a tag-based run. A re-run of a *main-dispatched* run will keep
failing the guard, which is correct. [ASSUMED: standard re-run semantics; not separately verified]

### 7. The guard's premise breaks under `workflow_call`
The SAN comes from `job_workflow_ref`, and there is no `github.job_workflow_ref` expression to compare
against. While signing stays inline in `release.yml` the two claims are equal. The inline comment in the
proposed snippet records this so a future reusable-workflow refactor does not silently defeat the guard.
[VERIFIED: fulcio oid-info.md + absence of the context property]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Extracting the ref from `workflow_ref` | `awk -F@`, `cut`, `sed`, regex | `${WORKFLOW_REF##*@}` | Pure bash, no subprocess, shellcheck-clean, no quoting hazards |
| Computing "the ref that was built" | New bespoke expression | Copy the ternary from line 33 verbatim | Guarantees the guard and the checkout can never disagree |
| Failing the job | `exit 1` bare | `::error title=...::` + `exit 1` | Surfaces the reason as a run-summary annotation |
| Overriding the signing ref | Any attempt to force `refs/tags/...` into the cert | Nothing — it is not possible | Identity comes from the OIDC token, not from workflow-controlled values |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | `actionlint` v1.7.11 + `yq` v4.53.3 + `shellcheck` (all locally installed) |
| Config file | none needed |
| Quick run command | `actionlint .github/workflows/release.yml` |
| Full suite command | `actionlint .github/workflows/*.yml && yq '.jobs.publish-release.steps \| map(.name)' .github/workflows/release.yml` |

### Requirements → Test Map
| Behavior | Test Type | Command | Exists? |
|----------|-----------|---------|---------|
| Workflow still valid YAML/Actions syntax | static | `actionlint .github/workflows/release.yml` | yes |
| Guard step lands before signing | static | `yq '.jobs.publish-release.steps \| map(.name)' .github/workflows/release.yml` | yes |
| Guard bash is lint-clean | static | extract `run:` body → `shellcheck -s bash -` | yes |
| Guard fails on ref mismatch | behavioral (local) | run the `run:` body with `WORKFLOW_REF=...@refs/heads/main EXPECTED_REF=refs/tags/vX`; expect exit 1 | yes (Case A above) |
| Guard passes on tag dispatch / tag push | behavioral (local) | same with matching refs; expect exit 0 | yes (Cases B/C above) |
| End-to-end identity is `@refs/tags/<v>` | manual, post-merge | next real release; then `cosign verify-blob --certificate-identity .../release.yml@refs/tags/<v>` | manual-only — requires a real tag + OIDC |

### Wave 0 Gaps
None — all static and behavioral checks are runnable with tooling already installed. Only the end-to-end
identity assertion is manual-only, and it is inherently deferred to the next release.

## Project Constraints (from CLAUDE.md)

- Conventional commits (`fix(ci): ...` / `build(ci): ...`).
- No "Generated with Claude Code" trailers in commits, PRs, or issues.
- No push to `main` without a PR.
- Keep comments minimal and non-verbose — the one comment block justifying the guard is warranted because
  the *reason* (identity comes from `workflow_ref`, not the checkout) is not inferable from the code.
- **Use `yq` for validating YAML files** — satisfied above.
- Do exactly what is asked; no unrequested refactoring of `release.yml`.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `actionlint` | workflow validation | yes | v1.7.11 | — |
| `yq` | YAML validation (CLAUDE.md mandate) | yes | v4.53.3 | — |
| `shellcheck` | guard bash lint | yes | installed | — |
| `gh` | issue/run inspection, dispatch remediation | yes | 2.96.0 | — |
| `cosign` | manual post-release identity verify | yes | installed (`cosign version`) | — |

No missing dependencies.

## Security Domain

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | yes | `release_tag` reaches the shell only via `env:` — no `${{ }}` interpolation inside `run:` |
| V6 Cryptography | yes | cosign keyless / Sigstore only; the change adds no crypto logic |
| V4 Access Control | indirect | the guard tightens what identities downstream consumers must trust |

| Pattern | STRIDE | Mitigation |
|---------|--------|-----------|
| Script injection via `workflow_dispatch` input | Tampering / EoP | pass through `env:`, quote all expansions |
| Signing-identity spoof-by-drift (this bug) | Spoofing | the guard itself — refuse to sign when identity ≠ built ref |
| Trust-boundary erosion via growing allowlist | Spoofing | fixing the root cause lets downstream retire the `v0.3.4`/`v0.3.5` exceptions |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Re-runs preserve the triggering event's ref, so `workflow_ref` is stable across re-runs | Gotcha 6 | Low — a drifting re-run ref would fail the guard (fail-safe direction), not silently mis-sign |

## Open Questions

1. **Fail-fast placement vs. locked placement**
   - Known: `publish-release` runs only after both build jobs finish, so guarding there wastes the full
     matrix build time on a bad invocation (~10-20 min of runner minutes).
   - Unclear: whether that waste is worth duplicating the guard into `build-artifacts` and
     `build-java-artifacts`.
   - Recommendation: ship the single guard as decided. The failure is loud and cheap to retry; two extra
     copies of the same bash violate "radically simple" for no correctness gain.

2. **Whether to document the tag-dispatch invocation in the repo**
   - Known: the guard's error message already prints the correct `gh workflow run ... --ref <tag>` command.
   - Unclear: whether a `RELEASING.md` / README note is also wanted.
   - Recommendation: out of scope — CONTEXT explicitly rejects docs-only fixes, and the error message is
     the point-of-need documentation. Raise separately if desired.

## Sources

### Primary (HIGH confidence)
- docs.github.com/en/actions/reference/workflows-and-actions/contexts — `github.workflow_ref` format, `github.ref`, `github.ref_name`
- docs.github.com/en/actions/reference/security/oidc — `workflow_ref` vs `job_workflow_ref` claim definitions
- github.com/sigstore/fulcio/blob/main/docs/oid-info.md — SAN/Build Signer URI = `server_url + job_workflow_ref`; `.1.5`/`.1.6` deprecated
- docs.github.com/en/rest/actions/workflows — dispatch `ref` accepts a branch **or tag** name
- Local repo evidence: `gh run list --workflow=release.yml` (run 22517529944 = tag dispatch on `v0.3.0`), `git show v0.3.0:.github/workflows/release.yml:109`, `git show v0.3.{3,4,5}:.github/workflows/release.yml`, `gh issue view 100` (decoded `v0.3.5` cert)
- Local validation: `actionlint` / `yq` / `shellcheck` runs on the candidate patch

### Secondary (MEDIUM confidence)
- sigstore/cosign discussion #2936 — certificate identity for reusable-workflow keyless signing (context for why `workflow_call` is a trap)

### Tertiary (LOW confidence — not relied upon)
- community discussion #75513 claiming dispatch accepts branches only. **Contradicted** by the official
  REST docs and by this repo's own `v0.3.0` tag dispatch. Disregarded.

## Metadata

**Confidence breakdown:**
- Root cause: HIGH — official docs plus a decoded certificate plus in-repo run history all agree
- Fix mechanism: HIGH — candidate patch written and validated with actionlint/yq/shellcheck and 4 behavioral cases
- Gotchas: HIGH except A1 (re-run semantics, ASSUMED, fail-safe direction)

**Research date:** 2026-07-30
**Valid until:** 2026-08-29 (GitHub Actions contexts and Fulcio OID mapping are stable surfaces)
