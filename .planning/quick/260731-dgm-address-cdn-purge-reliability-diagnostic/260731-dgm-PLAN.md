---
phase: quick/260731-dgm
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - .github/workflows/release.yml
  - .github/workflows/ci.yml
  - .planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-SUMMARY.md
autonomous: true
requirements: [ISSUE-97, REVIEW-DGM]

must_haves:
  truths:
    - "Every missing-credential combination emits one accurate warning, exits 0, and gives stable remediation without claiming a fixed cache TTL"
    - "Bad credentials, rate limits, HTTP 5xx responses, and transport failures expose a useful Cloudflare diagnostic but do not prevent GitHub Release publication"
    - "An HTTP 200 Cloudflare response with success:false is treated as a failed purge, while success:true is the only successful purge result"
    - "The purge step remains immediately before Publish GitHub release and always returns success after reporting a purge failure"
    - "Pull requests and main-branch pushes automatically run actionlint and yamllint against every GitHub Actions workflow"
    - "The prior summary correctly explains final-command exit-status propagation and retains e22f1a0 as the implementation commit"
  artifacts:
    - path: ".github/workflows/release.yml"
      provides: "Non-fatal, response-aware Cloudflare cache purge with actionable warnings"
      contains: "--fail-with-body"
    - path: ".github/workflows/ci.yml"
      provides: "Automated actionlint and yamllint workflow gate"
      contains: "actionlint"
    - path: ".planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-SUMMARY.md"
      provides: "Correct shell exit-status rationale and preserved implementation hash"
      contains: "e22f1a0"
  key_links:
    - from: ".github/workflows/release.yml purge curl"
      to: "Cloudflare response validation"
      via: "captured curl status/body followed by jq -e success == true"
      pattern: "jq.*success.*true"
    - from: ".github/workflows/release.yml purge failure branches"
      to: "Publish GitHub release"
      via: "diagnostic warning followed by exit 0, with the publish step still immediately next"
      pattern: "exit 0"
    - from: ".github/workflows/ci.yml workflow-lint step"
      to: ".github/workflows/*.yml"
      via: "actionlint and yamllint commands over the complete workflow glob"
      pattern: "yamllint.*\\.github/workflows"
---

<objective>
Close the supplied CDN purge reliability and diagnostics review findings on the current issue #97 branch.

Purpose: A Cloudflare purge is useful release maintenance, but it must never block publication. Maintainers must see why a purge failed, including HTTP-200 API-level failures, and future workflow regressions must be caught automatically.

Output: A hardened purge step in `release.yml`, a small Linux-only workflow-lint step in the existing CI matrix, and a corrected rationale in the prior quick-task summary.

RF-05 is already satisfied by commit `6ef47ce`: implementation references use `e22f1a0`, not `c2fda6e`. Do not rewrite hashes or recreate that correction; only verify it remains true while editing the same summary for RF-06.
</objective>

<execution_context>
@/Users/tazarov/.codex/get-shit-done/workflows/execute-plan.md
@/Users/tazarov/.codex/get-shit-done/templates/summary.md
</execution_context>

<context>
@AGENTS.md
@.planning/STATE.md
@.github/workflows/release.yml
@.github/workflows/ci.yml
@.planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-SUMMARY.md

<interfaces>
Current release ordering:

`Upload artifacts to R2` -> `Purge release metadata from CDN cache` -> `Publish GitHub release`

The purge step receives these existing values through its `env:` block:

- `CF_ZONE_ID` from `vars.CF_ZONE_ID`
- `CLOUDFLARE_API_TOKEN` from `secrets.CLOUDFLARE_API_TOKEN`
- `PROJECT` set to `chroma-go-local`
- `RELEASES_DOMAIN` from the repository variable with its existing domain fallback

Relevant established repository patterns:

- Workflow shell blocks use `set -euo pipefail`.
- Existing download curls use `--fail-with-body --retry 3 --retry-all-errors --max-time 180`.
- `jq` is already used in the same `publish-release` job.
- The existing CI `build-test-lint` matrix sets up Go on Linux, macOS, and Windows.
- Local validation has actionlint v1.7.11, yamllint 1.38.0, yq v4.53.3, and shellcheck 0.11.0.
- Full actionlint currently has one unrelated SC2129 style advisory in `ci.yml`; actionlint with its optional shellcheck integration disabled is clean.
- Yamllint is clean for the current workflow corpus when document-start and line-length are disabled, GitHub's `on` key is allowed by the truthy rule, one-space inline comments are allowed, and one terminal blank line is allowed.
</interfaces>
</context>

<source_audit>

| Source | ID | Finding | Task | Status |
|--------|----|---------|------|--------|
| GOAL | — | Address supplied CDN purge review findings without blocking releases | 1, 2 | COVERED |
| REQ | ISSUE-97 | Make CDN purge failures visible while release publication continues | 1 | COVERED |
| RESEARCH | — | No research artifact; quick-task constraint explicitly forbids a research phase | — | EXCLUDED |
| CONTEXT | RF-01 | All purge failures are non-fatal | 1 | COVERED |
| CONTEXT | RF-02 | Curl exposes HTTP response diagnostics | 1 | COVERED |
| CONTEXT | RF-03 | HTTP 200 with success:false is failure | 1 | COVERED |
| CONTEXT | RF-04 | Use leading-comma accumulator and `${MISSING#, }` | 1 | COVERED |
| CONTEXT | RF-05 | Hash references already corrected to e22f1a0 | 2 verification only | SATISFIED — NO REDO |
| CONTEXT | RF-06 | Correct the prior summary's `set -e` rationale | 2 | COVERED |
| CONTEXT | RF-07 | Add actionlint/yamllint CI gate | 2 | COVERED |
| CONTEXT | RF-08 | Warning gives stable remediation without a hardcoded TTL | 1 | COVERED |

</source_audit>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Make Cloudflare purge outcomes diagnostic and non-fatal</name>
  <files>.github/workflows/release.yml</files>
  <behavior>
    - Both credentials absent: one warning names both credentials in order, gives repository-setting and same-tag rerun remediation, makes no curl call, and exits 0.
    - Only CF_ZONE_ID absent: one warning names only CF_ZONE_ID, makes no curl call, and exits 0.
    - Only CLOUDFLARE_API_TOKEN absent: one warning names only CLOUDFLARE_API_TOKEN, makes no curl call, and exits 0.
    - Both credentials present and curl returns HTTP/transport failure: Cloudflare body/curl error is visible, a purge-failed warning is emitted, the token value is absent, and the step exits 0.
    - Both credentials present and HTTP 200 JSON has success:false: the compact Cloudflare errors/messages are visible, a purge-failed warning is emitted, and the step exits 0.
    - Both credentials present and HTTP 200 JSON has success:true: no warning is emitted and the step exits 0.
    - After every exit-0 case, a parent harness can run a PUBLISH_REACHABLE sentinel, proving the following GitHub Release step is not suppressed by purge status.
  </behavior>
  <action>
Before changing the workflow, create a disposable Bash harness at `/tmp/260731-dgm-cdn-purge-test.sh`. It must extract the purge step's `run` body with yq, place a controllable curl stub ahead of the real curl, and exercise the six cases in the behavior block. The curl stub must support: success:true with rc 0, success:false with rc 0, and an HTTP-style Cloudflare error body plus curl diagnostic with rc 22. Each case runs the extracted step as a child process, asserts rc 0, and then emits/asserts `PUBLISH_REACHABLE` in the parent. Also assert exact missing-name combinations, one warning only, expected diagnostic text, and that a sentinel token never appears in output. Run the harness once against the current workflow and confirm the HTTP-error and success:false cases fail before implementation.

Then edit only `Purge release metadata from CDN cache` in the `publish-release` job:

- Implement RF-04 exactly. Start `MISSING` empty; each missing check appends `, CF_ZONE_ID (repository variable)` or `, CLOUDFLARE_API_TOKEN (repository secret)` without branching on prior contents; normalize once with `MISSING="${MISSING#, }"` before testing it.
- For RF-08, keep one GitHub `::warning title=CDN cache not purged::` annotation when credentials are missing. Name only the missing variable(s), tell the maintainer to add each named item under repository Actions secrets/variables and rerun the release for the same tag, and say metadata can remain stale until a successful purge. Do not state or imply a numeric TTL. Never interpolate the credential values.
- Preserve `set -euo pipefail`, the existing env block, request URL, authorization header, JSON request body, step name, and step position.
- Replace the current curl flags with the repository's diagnostic/retry pattern using `-sS --fail-with-body`, `--retry 3`, `--retry-all-errors`, and `--max-time 180`. Capture combined output so both Cloudflare's response body and curl's own HTTP/transport diagnostic remain available when curl is non-zero.
- A non-zero curl result covers bad token/zone responses, 429, 5xx, timeout, DNS, and other transport failures. Compact CR/LF before printing it as a single ordinary log line prefixed with `Cloudflare purge diagnostic:`; keep the annotation text fixed rather than embedding the untrusted response into a workflow command. Emit the purge-failed warning and explicitly `exit 0`.
- When curl returns 0, require `jq -e '.success == true'` to pass. Treat success:false, malformed JSON, or missing success as purge failure; print a compact diagnostic (prefer valid compact JSON, with a single-line raw fallback), emit the same non-fatal warning, and `exit 0`.
- On success:true, print a short success message. Do not add `continue-on-error`: the shell handles only the expected purge failures, while syntax/programming failures should remain visible during CI validation.

Keep the two explicit `exit 0` failure paths. This makes the release behavior unambiguous and avoids relying on incidental final-command status. Do not reintroduce a step-level `if:` guard.
  </action>
  <verify>
    <automated>bash /tmp/260731-dgm-cdn-purge-test.sh &amp;&amp; actionlint .github/workflows/release.yml &amp;&amp; yamllint -d '{extends: default, rules: {document-start: disable, line-length: disable, comments: {min-spaces-from-content: 1}, truthy: {allowed-values: ["true", "false", "on"]}, empty-lines: {max-end: 1}}}' .github/workflows/release.yml &amp;&amp; yq -e '.jobs.publish-release.steps | map(.name) | join("\n") | test("Purge release metadata from CDN cache\\nPublish GitHub release")' .github/workflows/release.yml</automated>
  </verify>
  <done>
The red-to-green harness passes all three credential-availability cases plus success:true, success:false, and HTTP/transport failure outcomes. Every failure warns with useful diagnostics and rc 0, the token sentinel never leaks, the next-step sentinel is reachable, release.yml is lint-clean, and Publish GitHub release remains immediately after the purge.
  </done>
</task>

<task type="auto">
  <name>Task 2: Gate workflow syntax in CI and correct the historical shell rationale</name>
  <files>.github/workflows/ci.yml, .planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-SUMMARY.md</files>
  <action>
In the existing `build-test-lint` job, add one Linux-only `Lint GitHub Actions workflows` step after `Set up Go`, so it reuses checkout and the configured Go toolchain and does not duplicate a job. Under `set -euo pipefail`, install actionlint with the exact existing verified version using `go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.11`, install yamllint from the GitHub-hosted Ubuntu runner's signed apt repositories, then run both linters against `.github/workflows/*.yml`.

Run actionlint with `-shellcheck=` in this all-workflow CI gate. This intentionally scopes the new gate to GitHub Actions syntax/expressions and avoids turning the known unrelated SC2129 style advisory into scope expansion. Task 1 still runs full actionlint, including shellcheck, against the changed release workflow. Run yamllint with the inline configuration already proven against this repository: disable document-start and line-length, allow one space before inline comments, allow `true`, `false`, and GitHub's `on` key under the truthy rule, and permit one terminal blank line. Keep the configuration inline so this review fix needs no new config file.

In the prior `260730-sz8-SUMMARY.md`, correct every claim that says the left side of an `&amp;&amp;` list aborts because of `set -e`. State the real hazard precisely: a failed left-hand test in an AND-list is exempt from immediate `set -e` termination, but the whole list can still return rc 1; when that list is the step's final command, rc 1 propagates as the step result. Preserve the record that full if/then/fi blocks avoided that final-status hazard. Do not alter the implementation hash: `e22f1a0` is already correct per RF-05 and commit `6ef47ce`.
  </action>
  <verify>
    <automated>actionlint -shellcheck= .github/workflows/*.yml &amp;&amp; actionlint .github/workflows/release.yml &amp;&amp; yamllint -d '{extends: default, rules: {document-start: disable, line-length: disable, comments: {min-spaces-from-content: 1}, truthy: {allowed-values: ["true", "false", "on"]}, empty-lines: {max-end: 1}}}' .github/workflows/*.yml &amp;&amp; rg -q 'final-command position|final command' .planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-SUMMARY.md &amp;&amp; rg -q 'e22f1a0' .planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-SUMMARY.md &amp;&amp; ! rg -q 'c2fda6e' .planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-SUMMARY.md</automated>
  </verify>
  <done>
The existing CI workflow automatically installs and runs actionlint/yamllint on Linux for every PR and main push; both commands pass the complete workflow corpus. The prior summary explains AND-list final-status propagation correctly and retains only e22f1a0 as the implementation commit.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| GitHub Actions runner -> Cloudflare API | Zone identifier and bearer token authorize the purge request |
| Cloudflare API response -> public workflow log | Remote response content becomes diagnostic output |
| Purge step -> GitHub Release step | Purge exit status determines whether publication remains reachable |
| CI runner -> Go module proxy / Ubuntu apt repositories | Workflow-linter tooling enters the runner |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-dgm-01 | Information Disclosure | purge diagnostics and annotations | mitigate | Never print the bearer token; missing warnings interpolate names only; the harness uses a sentinel token and fails if it appears |
| T-dgm-02 | Denial of Service | purge step exit status | mitigate | Missing credentials, curl failures, success:false, and malformed responses each warn then explicitly exit 0; harness asserts the publication sentinel is reachable |
| T-dgm-03 | Tampering | HTTP 200 Cloudflare JSON | mitigate | Do not trust HTTP status alone; jq must establish the exact boolean `success == true` |
| T-dgm-04 | Spoofing / log-command injection | Cloudflare response logging | mitigate | Compact CR/LF and print response behind a fixed ordinary-log prefix; keep remote content out of `::warning::` command text |
| T-dgm-05 | Repudiation | silent cache-purge failure | mitigate | Every failure produces a titled warning plus diagnostic/remediation while the release continues |
| T-dgm-SC | Tampering | workflow-linter installation | mitigate | Pin actionlint to v1.7.11; source yamllint from the runner's signed Ubuntu apt repositories; introduce no npm, pip, or cargo install and no new third-party action |
</threat_model>

<verification>
Run the Task 1 behavior harness after both tasks so it validates the final workflow, not an intermediate copy.

Required behavior matrix:

| CF_ZONE_ID | Token | Curl/API outcome | Expected warning | Exit | Publish sentinel |
|------------|-------|------------------|------------------|------|------------------|
| absent | absent | no call | both names + remediation | 0 | reached |
| present | absent | no call | token only + remediation | 0 | reached |
| absent | present | no call | zone only + remediation | 0 | reached |
| present | present | success:true | none | 0 | reached |
| present | present | success:false | purge failed + response errors | 0 | reached |
| present | present | rc 22 / 4xx, 429, or 5xx body | purge failed + curl/body diagnostic | 0 | reached |

For transport coverage, run the same failure stub with a timeout/DNS-style non-zero code and confirm the result is identical: warning, diagnostic, rc 0, publication sentinel reached.

Static checks:

- Full actionlint passes for `release.yml`, including shellcheck of its changed run body.
- CI's scoped actionlint command passes all `.github/workflows/*.yml`.
- The exact inline yamllint command in CI passes all `.github/workflows/*.yml`.
- Yq confirms the purge step still has no step-level `if:` and is immediately followed by `Publish GitHub release`.
- `git diff --check` passes.
- The release warning contains no numeric TTL and no secret value.
- `git show --quiet 6ef47ce` succeeds; summary references remain `e22f1a0` with no `c2fda6e`.

No Go, Rust, Java, FFI, or runtime behavior changes in this plan, so the repository's language build/test targets are not required for this workflow-only change.
</verification>

<success_criteria>
- Missing credentials use the exact leading-comma accumulator plus `${MISSING#, }` normalization and retain all four credential outcomes.
- Missing-credential remediation names the repository configuration location and same-tag rerun action without hardcoding a TTL.
- Curl exposes response bodies and diagnostics, retries transient failures, and cannot fail the release for Cloudflare/API/transport outcomes.
- Only Cloudflare JSON with `success: true` counts as a successful purge.
- `Publish GitHub release` remains immediately after the purge and is reachable after every modeled purge failure.
- CI automatically gates all workflows with actionlint and yamllint using the repository-compatible invocations.
- The old summary contains the correct AND-list/final-command rationale and retains the already-correct implementation hash.
</success_criteria>

<output>
Create `.planning/quick/260731-dgm-address-cdn-purge-reliability-diagnostic/260731-dgm-SUMMARY.md` when done.
</output>
