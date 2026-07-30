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
  - .github/actions/setup-protoc/
  - .gitignore
  - .planning/quick/260730-p2k-replace-arduino-setup-protoc-with-chroma/260730-p2k-PLAN.md
autonomous: true
requirements: [ISSUE-96-FOLLOW-UP]

must_haves:
  truths:
    - "The existing action is vendored locally with its TypeScript source, bundled runtime, lockfile, and MIT license."
    - "All four setup-protoc callers use the local action from the checked-out repository revision."
    - "A weekly, read-only GitHub Actions check fails visibly when `arduino/setup-protoc` has moved beyond the recorded upstream base."
    - "The `chroma/` runtime-data directory is ignored."
    - "The prior plan explicitly records Windows evidence, fork availability, and bundled-action review risks."
  artifacts:
    - path: ".github/actions/setup-protoc/action.yml"
      provides: "Local Node 24 protoc setup action"
      contains: "using: 'node24'"
    - path: ".github/workflows/check-setup-protoc-upstream.yml"
      provides: "Scheduled upstream-drift signal using a read-only GitHub token"
      contains: "schedule"
    - path: ".gitignore"
      provides: "Exclusion for generated Chroma runtime data"
      contains: "chroma/"
  key_links:
    - from: ".github/workflows/ci.yml and .github/workflows/release.yml"
      to: ".github/actions/setup-protoc/action.yml"
      via: "local action path resolved after checkout"
      pattern: "uses: ./\\.github/actions/setup-protoc"
---

<objective>
Remove the external action-repository availability dependency, make upstream security drift visible, and prevent generated Chroma runtime data from being committed accidentally.

Vendor the already reviewed `chroma-core/setup-protoc` commit `df9e7872eaabfd0ddfafd9e27fe77c6229bc7d22` under `.github/actions/setup-protoc`. The local action is versioned by the same repository commit or release tag as the workflow that invokes it. Once it has passed the existing CI matrix in practice, it can be extracted to a dedicated organization repository without altering its source.
</objective>

<tasks>

<task type="auto">
  <name>Task 1: Vendor the reviewed action locally with provenance</name>
  <files>.github/actions/setup-protoc/action.yml, .github/actions/setup-protoc/dist/index.js, .github/actions/setup-protoc/src/, .github/actions/setup-protoc/package.json, .github/actions/setup-protoc/package-lock.json, .github/actions/setup-protoc/LICENSE, .github/actions/setup-protoc/UPSTREAM.md</files>
  <action>
Copy the runtime files and audit material from `chroma-core/setup-protoc` at commit `df9e7872eaabfd0ddfafd9e27fe77c6229bc7d22`: `action.yml`, the generated `dist/index.js`, TypeScript source, `package.json`, `package-lock.json`, and `LICENSE`. Add `UPSTREAM.md` identifying this exact source commit, its `arduino/setup-protoc` base `3ea1d70ac22caff0b66ed6cb37d5b7aadebd4623`, the Node 24 and Windows support, and the procedure for reviewing and updating both source and bundle. Do not include tests, dependency directories, or unrelated upstream repository files.
  </action>
  <verify>
    <automated>test -f .github/actions/setup-protoc/action.yml && test -f .github/actions/setup-protoc/dist/index.js && test -f .github/actions/setup-protoc/src/installer.ts && test -f .github/actions/setup-protoc/LICENSE && rg -n "df9e7872|3ea1d70|win32|node24" .github/actions/setup-protoc</automated>
  </verify>
  <done>The local action can execute without a remote action repository, and its source, generated bundle, license, and provenance are available for review.</done>
</task>

<task type="auto">
  <name>Task 2: Use the local action and add a weekly upstream-drift signal</name>
  <files>.github/workflows/ci.yml, .github/workflows/release.yml, .github/workflows/check-setup-protoc-upstream.yml</files>
  <action>
After Task 1 succeeds, replace each of the three CI and one release `uses:` lines with `./.github/actions/setup-protoc`. Preserve `version: "31.x"` and `repo-token` inputs. Add a scheduled and manually dispatchable workflow with only `contents: read` permission. It must call GitHub's compare API for the recorded base `3ea1d70ac22caff0b66ed6cb37d5b7aadebd4623...master`, log `ahead_by`, and fail with a clear remediation message when upstream has commits beyond that base. No third-party action may run in the monitoring job.
  </action>
  <verify>
    <automated>test "$(rg -c 'uses: ./\\.github/actions/setup-protoc' .github/workflows/ci.yml .github/workflows/release.yml | awk -F: '{sum += $2} END {print sum}')" = 4 && actionlint .github/workflows/ci.yml .github/workflows/release.yml .github/workflows/check-setup-protoc-upstream.yml</automated>
  </verify>
  <done>All four callers use the repository-local action and the weekly monitor surfaces upstream drift without write permissions.</done>
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
