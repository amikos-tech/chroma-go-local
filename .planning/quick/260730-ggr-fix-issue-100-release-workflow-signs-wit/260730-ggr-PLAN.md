---
phase: quick/260730-ggr
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - .github/workflows/release.yml
autonomous: true
requirements: [ISSUE-100]

must_haves:
  truths:
    - "The release workflow hard-fails before signing when the ref baked into github.workflow_ref differs from the ref that was actually built"
    - "The guard passes unchanged for a tag push (v* tag) and for a workflow_dispatch started on the tag itself"
    - "On mismatch the run summary shows an error annotation naming both refs and the exact `gh workflow run release.yml --ref <tag> -f release_tag=<tag>` remediation command"
    - "No artifact is signed, uploaded to R2, or published to a GitHub release when the guard fails"
    - "release.yml remains valid GitHub Actions syntax (actionlint clean) and the guard bash is shellcheck clean"
  artifacts:
    - path: ".github/workflows/release.yml"
      provides: "publish-release step 'Verify signing identity matches released ref' placed before 'Install cosign'"
      contains: "Verify signing identity matches released ref"
  key_links:
    - from: "guard step env WORKFLOW_REF"
      to: "github.workflow_ref context"
      via: "env: block (never inlined into run:)"
      pattern: "WORKFLOW_REF: \\$\\{\\{ github.workflow_ref \\}\\}"
    - from: "guard step env EXPECTED_REF"
      to: "the same ternary used by actions/checkout at release.yml:33 and :128"
      via: "verbatim expression reuse so guard and checkout cannot drift"
      pattern: "format\\('refs/tags/\\{0\\}', github.event.inputs.release_tag\\) \\|\\| github.ref"
    - from: "guard step position"
      to: "'Install cosign' / 'Sign and verify artifacts' / 'Upload artifacts to R2' / 'Publish GitHub release'"
      via: "non-zero exit skips all subsequent steps in the job"
      pattern: "steps ordered: Build checksum manifest -> guard -> Install cosign"
---

<objective>
Add a fail-loud guard to `.github/workflows/release.yml` that refuses to sign release artifacts when the cosign keyless identity would not match the ref the artifacts were built from (GitHub issue #100).

Purpose: cosign derives the certificate SAN from the OIDC `job_workflow_ref` claim — i.e. the ref the workflow *file* was loaded from — not from whatever `actions/checkout` pulled. Dispatching `release.yml` from `main` therefore builds tag content but signs it `release.yml@refs/heads/main`, forcing downstream consumers (chroma-go) to widen their cosign identity allowlist for every affected release. The identity cannot be overridden from inside the workflow, so the only correct fix is to detect the mismatch and refuse to proceed.

Output: one new `shell: bash` step in the `publish-release` job. No other workflow changes, no reusable-workflow restructuring, no docs-only fix.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/quick/260730-ggr-fix-issue-100-release-workflow-signs-wit/260730-ggr-CONTEXT.md
@.planning/quick/260730-ggr-fix-issue-100-release-workflow-signs-wit/260730-ggr-RESEARCH.md
@.github/workflows/release.yml

The RESEARCH.md section **"Proposed Guard Step (validated)"** contains the exact YAML to insert. It was
already validated locally with `actionlint` v1.7.11, `yq` v4.53.3, `shellcheck`, and four behavioral cases.
Use it as written — Task 1 is an insertion, not a redesign.

Read RESEARCH.md **"Gotchas"** before editing. Load-bearing points:
- `${WORKFLOW_REF##*@}` (strip through the LAST `@`). `%%@*` is wrong — it yields the repo/path prefix.
- `github.event.inputs` is null outside `workflow_dispatch`; the `event_name == 'workflow_dispatch' && ... || ...`
  ternary short-circuits before `format()` runs. Reuse the existing shape from lines 33/128/151 rather than
  inventing a new expression.
- Never inline `${{ github.event.inputs.release_tag }}` into a `run:` body — script-injection sink. Pass via `env:`.
- Do NOT add normalization/sanitization for a malformed `release_tag`; failing loud is intended (Gotcha 3).
</context>

<interfaces>
Existing expressions in `.github/workflows/release.yml` that the guard must reuse verbatim:

- Built ref (used by `actions/checkout` at lines 33 and 128):
  `${{ github.event_name == 'workflow_dispatch' && format('refs/tags/{0}', github.event.inputs.release_tag) || github.ref }}`
- Bare tag name (used at lines 151, 219, 313, 404):
  `${{ github.event_name == 'workflow_dispatch' && github.event.inputs.release_tag || github.ref_name }}`
- Signing identity, consumed twice (lines 265-269 and 316/359):
  `WORKFLOW_REF: ${{ github.workflow_ref }}` then `IDENTITY="https://github.com/${WORKFLOW_REF}"`

Current `publish-release` step order (names are the stable handle for verification):
`Checkout repository scripts` -> `Download artifact bundles` -> `Download Java artifact bundles` ->
`Normalize artifact names` -> `Build checksum manifest` -> **[insert guard here]** -> `Install cosign` ->
`Sign and verify artifacts` -> `Upload artifacts to R2` -> `Purge release metadata from CDN cache` ->
`Publish GitHub release`
</interfaces>

<tasks>

<task type="auto">
  <name>Task 1: Insert the signing-identity guard into publish-release</name>
  <files>.github/workflows/release.yml</files>
  <action>
Insert a single new step named exactly `Verify signing identity matches released ref` into the
`publish-release` job, immediately before the existing `- name: Install cosign` step (currently line 256)
and immediately after `Build checksum manifest`. Copy the YAML verbatim from the RESEARCH.md section
"Proposed Guard Step (validated)", including its leading comment block.

Structural requirements for the step:
- `shell: bash`.
- Three `env:` keys, no `${{ }}` interpolation anywhere inside `run:`:
  - `WORKFLOW_REF` -> `${{ github.workflow_ref }}`
  - `EXPECTED_REF` -> the checkout ternary reproduced verbatim from line 33
  - `RELEASE_TAG` -> the bare-tag ternary reproduced verbatim from line 151
- `run:` body opens with `set -euo pipefail` (house style, 8 existing occurrences in this file), derives
  `SIGNING_REF="${WORKFLOW_REF##*@}"`, and compares it against `EXPECTED_REF`.
- On mismatch: emit a `::error title=Signing identity mismatch::` annotation naming both refs, print the
  refusal reason with an issue #100 reference, print the copy-pasteable
  `gh workflow run release.yml --ref "${RELEASE_TAG}" -f release_tag="${RELEASE_TAG}"` remediation, then
  `exit 1`. On match: print a one-line confirmation and fall through.
- Quote every variable expansion.

Keep the comment block: it records *why* the guard exists (identity comes from `workflow_ref`, not the
checkout) and the caveat that the premise holds only while signing stays inline in this file — neither is
inferable from the code. Per CLAUDE.md, that is the one warranted comment; add nothing beyond it.

Do not touch any other step. Do not add a second guard to the build jobs, do not add a `workflow_call`
trigger, and do not modify the `publish-release` `if:` gate at line 193.
  </action>
  <verify>
    <automated>
cd /Users/tazarov/experiments/amikos/local-go-chroma
set -euo pipefail

# 1. Workflow still valid Actions syntax
actionlint .github/workflows/release.yml

# 2. Guard exists and lands before the cosign install
NAMES=$(yq -r '.jobs.publish-release.steps[].name' .github/workflows/release.yml)
G=$(printf '%s\n' "$NAMES" | grep -n '^Verify signing identity matches released ref$' | cut -d: -f1)
C=$(printf '%s\n' "$NAMES" | grep -n '^Install cosign$' | cut -d: -f1)
B=$(printf '%s\n' "$NAMES" | grep -n '^Build checksum manifest$' | cut -d: -f1)
test -n "$G" && test "$B" -lt "$G" && test "$G" -lt "$C"

# 3. Guard bash is shellcheck clean
yq -r '.jobs.publish-release.steps[] | select(.name == "Verify signing identity matches released ref") | .run' \
  .github/workflows/release.yml | shellcheck -s bash -

# 4. No script-injection sink: no ${{ }} inside the guard run body
! yq -r '.jobs.publish-release.steps[] | select(.name == "Verify signing identity matches released ref") | .run' \
  .github/workflows/release.yml | grep -q '\${{'

# 5. Guard reads the identity from github.workflow_ref, not a reinvented value
yq -r '.jobs.publish-release.steps[] | select(.name == "Verify signing identity matches released ref") | .env.WORKFLOW_REF' \
  .github/workflows/release.yml | grep -q 'github.workflow_ref'

# 6. EXPECTED_REF matches the checkout ternary byte-for-byte
CHECKOUT_REF=$(yq -r '.jobs.build-artifacts.steps[] | select(.name == "Checkout") | .with.ref' .github/workflows/release.yml)
GUARD_REF=$(yq -r '.jobs.publish-release.steps[] | select(.name == "Verify signing identity matches released ref") | .env.EXPECTED_REF' .github/workflows/release.yml)
test "$CHECKOUT_REF" = "$GUARD_REF"
    </automated>
  </verify>
  <done>
`actionlint` is clean; `yq` reports the guard step positioned between `Build checksum manifest` and
`Install cosign`; the guard's bash passes `shellcheck -s bash`; the `run:` body contains zero `${{`
sequences; `EXPECTED_REF` is byte-identical to the `build-artifacts` checkout `ref` expression.
  </done>
</task>

<task type="auto">
  <name>Task 2: Prove the guard's exit-code matrix locally</name>
  <files>.github/workflows/release.yml (read-only — no modifications in this task)</files>
  <action>
Extract the guard's `run:` body straight out of the committed YAML with `yq` and execute it under the four
environment combinations from RESEARCH.md "Local Validation Performed", asserting exit codes. Extracting
from the YAML (rather than re-typing the script) is what makes this a real regression check on the shipped
artifact.

Expected matrix:
- Case A — dispatch from `main`, `release_tag=v0.3.6` (`WORKFLOW_REF` ends `@refs/heads/main`,
  `EXPECTED_REF=refs/tags/v0.3.6`) -> exit 1. This is the issue #100 bug.
- Case B — dispatch from tag `v0.3.6` (both refs `refs/tags/v0.3.6`) -> exit 0.
- Case C — tag push `v0.3.6` (same values as B, since `github.ref` is the tag) -> exit 0.
- Case D — dispatched on `v0.3.6` but `release_tag=v0.3.5` -> exit 1. Free bonus catch: build would check
  out `v0.3.5` while signing as `v0.3.6`.

Also confirm the Case A stderr/stdout contains the remediation command with the correct tag substituted,
so an operator hitting this in CI is not left guessing.

If any case deviates, fix the guard in Task 1's file rather than adjusting the expectations — the matrix
is the specification.

Write no new files. This is verification only; no test script is committed (out of scope per CONTEXT.md).
  </action>
  <verify>
    <automated>
cd /Users/tazarov/experiments/amikos/local-go-chroma
set -euo pipefail

GUARD=$(yq -r '.jobs.publish-release.steps[] | select(.name == "Verify signing identity matches released ref") | .run' .github/workflows/release.yml)
test -n "$GUARD"
REPO_PATH='amikos-tech/chroma-go-local/.github/workflows/release.yml'

run_case() {
  set +e
  env WORKFLOW_REF="$1" EXPECTED_REF="$2" RELEASE_TAG="$3" bash -c "$GUARD" >/tmp/guard.out 2>&1
  local rc=$?
  set -e
  echo "$rc"
}

# Case A: dispatch from main -> must refuse
test "$(run_case "${REPO_PATH}@refs/heads/main" 'refs/tags/v0.3.6' 'v0.3.6')" = "1"
grep -q 'gh workflow run release.yml --ref' /tmp/guard.out
grep -q 'v0.3.6' /tmp/guard.out
grep -q 'Signing identity mismatch' /tmp/guard.out

# Case B: dispatch from the tag -> must pass
test "$(run_case "${REPO_PATH}@refs/tags/v0.3.6" 'refs/tags/v0.3.6' 'v0.3.6')" = "0"

# Case C: tag push -> must pass
test "$(run_case "${REPO_PATH}@refs/tags/v0.3.6" 'refs/tags/v0.3.6' 'v0.3.6')" = "0"

# Case D: dispatched on v0.3.6 with release_tag=v0.3.5 -> must refuse
test "$(run_case "${REPO_PATH}@refs/tags/v0.3.6" 'refs/tags/v0.3.5' 'v0.3.5')" = "1"

rm -f /tmp/guard.out
echo "guard matrix A=1 B=0 C=0 D=1 confirmed"
    </automated>
  </verify>
  <done>
All four cases produce the expected exit codes (A=1, B=0, C=0, D=1) against the script extracted from the
committed `release.yml`, and the Case A output contains the `Signing identity mismatch` annotation plus the
tag-substituted `gh workflow run release.yml --ref ...` remediation line.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| operator -> `workflow_dispatch` input | `release_tag` is free-form untrusted string entering a CI shell |
| GitHub OIDC -> Fulcio -> downstream verifier | certificate identity is the trust anchor consumers pin |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-ggr-01 | Spoofing | cosign identity in `Sign and verify artifacts` | mitigate | The guard itself: refuse to sign when `${WORKFLOW_REF##*@}` != the built ref, so a `refs/heads/main` identity can never be attached to tag content (issue #100 root cause) |
| T-ggr-02 | Tampering / EoP | guard `run:` body | mitigate | `release_tag` reaches the shell only via `env: RELEASE_TAG`; zero `${{ }}` inside `run:` (asserted by Task 1 verify check 4); all expansions quoted |
| T-ggr-03 | Spoofing | downstream chroma-go allowlist | mitigate | Fixing the root cause stops per-release allowlist growth and lets the `v0.3.4`/`v0.3.5` exceptions be retired |
| T-ggr-04 | Spoofing | future `workflow_call` refactor | accept | SAN comes from `job_workflow_ref`, which has no matching context expression; recorded in the step's inline comment so a reusable-workflow refactor cannot silently defeat the guard |
| T-ggr-SC | Tampering | npm/pip/cargo installs | n/a | No package installs in this change; `actionlint`/`yq`/`shellcheck`/`cosign` are pre-installed locally, all workflow actions remain SHA-pinned and unmodified |
</threat_model>

<verification>
Full static suite (CLAUDE.md mandates `yq` for YAML validation):

```
actionlint .github/workflows/*.yml
yq '.jobs.publish-release.steps | map(.name)' .github/workflows/release.yml
git diff --stat   # must show exactly one file, .github/workflows/release.yml
```

The diff must be additive-only within `publish-release`: no changed action SHAs, no altered `if:` gates, no
edits to `build-artifacts` or `build-java-artifacts`.

**Deferred, manual, post-merge:** end-to-end identity confirmation requires a real tag plus live OIDC. On
the next release run `cosign verify-blob --bundle <artifact>.sigstore.json --certificate-identity
"https://github.com/amikos-tech/chroma-go-local/.github/workflows/release.yml@refs/tags/<version>"
--certificate-oidc-issuer "https://token.actions.githubusercontent.com" --use-signed-timestamps <artifact>`
and confirm it passes without an allowlist widening. This cannot be automated pre-merge.
</verification>

<success_criteria>
- [ ] `.github/workflows/release.yml` contains exactly one new step, `Verify signing identity matches released ref`, between `Build checksum manifest` and `Install cosign` in `publish-release`
- [ ] `actionlint .github/workflows/*.yml` clean; guard `run:` body clean under `shellcheck -s bash`
- [ ] Guard `EXPECTED_REF` is byte-identical to the `actions/checkout` ref ternary (no independent reimplementation)
- [ ] Exit-code matrix confirmed: dispatch-from-main = 1, tag-dispatch = 0, tag-push = 0, mismatched-input = 1
- [ ] Mismatch output carries an `::error title=Signing identity mismatch::` annotation and the tag-substituted `gh workflow run release.yml --ref <tag> -f release_tag=<tag>` command
- [ ] No `${{ }}` interpolation inside the guard's `run:` body
- [ ] `git diff` touches only `.github/workflows/release.yml`; no `workflow_call` trigger, no docs-only changes, no new files
- [ ] Committed with a conventional commit (`fix(ci): ...`), no Claude Code attribution trailer, delivered via PR — never pushed straight to `main`
</success_criteria>

<output>
Create `.planning/quick/260730-ggr-fix-issue-100-release-workflow-signs-wit/260730-ggr-SUMMARY.md` when done.
</output>
