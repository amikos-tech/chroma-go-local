---
phase: quick/260730-pii
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - .github/workflows/ci.yml
  - .github/workflows/release.yml
  - .github/workflows/check-setup-protoc-upstream.yml
  - .gitignore
  - .planning/quick/260730-p2k-replace-arduino-setup-protoc-with-chroma/260730-p2k-PLAN.md
autonomous: false
requirements: [ISSUE-96-FOLLOW-UP]

must_haves:
  truths:
    - "The `amikos/setup-protoc` fork exists, is public, and contains the existing pinned commit."
    - "All four setup-protoc callers use the exact same full SHA from the `amikos` fork."
    - "A weekly, read-only GitHub Actions check fails visibly when the fork is behind `arduino/setup-protoc`."
    - "The `chroma/` runtime-data directory is ignored."
    - "The prior plan explicitly records Windows evidence, fork availability, and bundled-action review risks."
  artifacts:
    - path: ".github/workflows/check-setup-protoc-upstream.yml"
      provides: "Scheduled upstream-drift signal using a read-only GitHub token"
      contains: "schedule"
    - path: ".gitignore"
      provides: "Exclusion for generated Chroma runtime data"
      contains: "chroma/"
  key_links:
    - from: ".github/workflows/ci.yml and .github/workflows/release.yml"
      to: "amikos/setup-protoc"
      via: "same immutable 40-character commit SHA at all four action call sites"
      pattern: "amikos/setup-protoc@[0-9a-f]{40}"
---

<objective>
Remove the availability dependency on `chroma-core/setup-protoc`, make upstream security drift visible, and prevent generated Chroma runtime data from being committed accidentally.

This task is blocked until an `amikos` organization owner creates the public `amikos/setup-protoc` fork of `chroma-core/setup-protoc`. The current GitHub identity does not have permission to create that organizational fork. The existing SHA must resolve in the new fork before any workflow reference changes.
</objective>

<tasks>

<task type="external">
  <name>Task 1: Create and validate the organization-owned action fork</name>
  <files>External repository: amikos/setup-protoc</files>
  <action>
An `amikos` organization owner creates a public fork of `chroma-core/setup-protoc` at `amikos/setup-protoc`. Verify it retains commit `df9e7872eaabfd0ddfafd9e27fe77c6229bc7d22`, is in the `arduino/setup-protoc` fork network, and its default branch contains the fork's Node 24 and Windows support.
  </action>
  <verify>
    <automated>gh api repos/amikos/setup-protoc --jq '{fork,private,parent:.parent.full_name,source:.source.full_name}' && gh api repos/amikos/setup-protoc/commits/df9e7872eaabfd0ddfafd9e27fe77c6229bc7d22 --jq .sha</automated>
  </verify>
  <done>The organization-owned public fork and the already reviewed immutable commit both resolve.</done>
</task>

<task type="auto">
  <name>Task 2: Move all callers and add a weekly upstream-drift signal</name>
  <files>.github/workflows/ci.yml, .github/workflows/release.yml, .github/workflows/check-setup-protoc-upstream.yml</files>
  <action>
After Task 1 succeeds, replace each of the three CI and one release `uses:` lines with `amikos/setup-protoc@df9e7872eaabfd0ddfafd9e27fe77c6229bc7d22`. Preserve `version: "31.x"` and `repo-token` inputs. Add a scheduled and manually dispatchable workflow with only `contents: read` permission. It must call GitHub's compare API for `arduino:master...amikos:master`, log `behind_by`, and fail with a clear remediation message when upstream has commits absent from the fork. No third-party action may run in the monitoring job.
  </action>
  <verify>
    <automated>test "$(rg -l 'uses: amikos/setup-protoc@df9e7872eaabfd0ddfafd9e27fe77c6229bc7d22' .github/workflows/ci.yml .github/workflows/release.yml | wc -l | tr -d ' ')" = 2 && test "$(rg -c 'uses: amikos/setup-protoc@df9e7872eaabfd0ddfafd9e27fe77c6229bc7d22' .github/workflows/ci.yml .github/workflows/release.yml | awk -F: '{sum += $2} END {print sum}')" = 4 && actionlint .github/workflows/ci.yml .github/workflows/release.yml .github/workflows/check-setup-protoc-upstream.yml</automated>
  </verify>
  <done>All four callers use the organization-owned immutable pin and the weekly monitor surfaces upstream drift without write permissions.</done>
</task>

<task type="auto">
  <name>Task 3: Record review evidence and ignore generated runtime data</name>
  <files>.gitignore, .planning/quick/260730-p2k-replace-arduino-setup-protoc-with-chroma/260730-p2k-PLAN.md</files>
  <action>
Add `chroma/` to the existing generated-runtime-data ignore group. Amend the earlier plan with the verified Windows evidence and explicit availability and bundled-artifact risks, including the required mitigation for the availability risk.
  </action>
  <verify>
    <automated>git check-ignore -q chroma/chroma.sqlite3 && rg -n 'win32|T-p2k-04|T-p2k-05' .planning/quick/260730-p2k-replace-arduino-setup-protoc-with-chroma/260730-p2k-PLAN.md</automated>
  </verify>
  <done>The test artifact is ignored and the reviewed plan accurately captures the risk record.</done>
</task>

</tasks>
