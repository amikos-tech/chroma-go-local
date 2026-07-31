---
phase: quick/260731-eho
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - .github/workflows/release.yml
  - .github/workflows/ci.yml
  - .yamllint
  - .planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-PLAN.md
autonomous: true
requirements:
  - EHO-01
  - EHO-02
  - EHO-03
  - EHO-04
  - EHO-05
  - EHO-06
  - EHO-07
  - EHO-08
  - EHO-09

must_haves:
  truths:
    - "A retried Cloudflare purge evaluates only the final response body and final HTTP status; a failed attempt cannot contaminate a later successful response"
    - "Transient transport errors and Cloudflare 5xx responses remain retryable, while a permanent response such as HTTP 403 is evaluated after one request rather than retried"
    - "Only a successful HTTP response whose JSON contains success:true is reported as a successful purge"
    - "Transport/HTTP failures and Cloudflare success:false failures produce distinct, fixed Actions annotations while release publication remains reachable"
    - "Empty or malformed JSON, a missing success field, and unavailable jq produce accurate ordinary-log diagnostics rather than being mislabeled as success:false"
    - "Workflow linting runs in its own Linux job, with ShellCheck enabled except for SC2129 and with an explicitly pinned yamllint version using root configuration"
    - "The historical 260730-sz8 plan no longer claims that errexit or ShellCheck proves behavior that neither mechanism guarantees"
  artifacts:
    - path: ".github/workflows/release.yml"
      provides: "Retry-safe Cloudflare response capture plus status- and JSON-aware non-fatal outcome handling"
      contains: "PURGE_CODE=000"
    - path: ".github/workflows/ci.yml"
      provides: "Standalone Linux workflow-lint job independent of the build/test OS matrix"
      contains: "workflow-lint"
    - path: ".yamllint"
      provides: "Repository-local yamllint policy shared by CI and local runs"
      contains: "extends: default"
    - path: ".planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-PLAN.md"
      provides: "Accurate historical explanation of AND-list exit status and actionlint/ShellCheck coverage"
  key_links:
    - from: ".github/workflows/release.yml curl invocation"
      to: "final body and HTTP status validation"
      via: "separate temporary body/stderr files plus -o and -w '%{http_code}'"
      pattern: "-w ['\"]?%\\{http_code\\}"
    - from: ".github/workflows/release.yml curl assignment"
      to: "non-fatal transport handling under set -e"
      via: "explicit assignment guard"
      pattern: "\\|\\| PURGE_CODE=000"
    - from: ".github/workflows/release.yml HTTP-success branch"
      to: "Cloudflare API contract"
      via: "captured jq exit status for .success == true"
      pattern: "jq -e.*success == true"
    - from: ".github/workflows/ci.yml workflow-lint job"
      to: ".yamllint"
      via: "yamllint -c .yamllint over every workflow"
      pattern: "yamllint.*-c \\.yamllint.*\\.github/workflows"
    - from: ".github/workflows/ci.yml actionlint command"
      to: "embedded Bash ShellCheck coverage"
      via: "one scoped SC2129 ignore, without -shellcheck="
      pattern: "actionlint.*-ignore ['\"]SC2129['\"]"
---

<objective>
Correct the CDN purge response-capture regression and make workflow linting a durable, reproducible CI gate.

Purpose: Cloudflare failures must remain non-fatal to release publication, but the purge step must evaluate the final retry result accurately and tell maintainers whether transport/HTTP handling or the Cloudflare API contract failed. Workflow lint coverage must remain present even if the build matrix changes.

Output: retry-safe purge shell in `release.yml`, a standalone workflow-lint job in `ci.yml`, root `.yamllint` configuration, and corrected historical claims in the prior quick-task plan.
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
@.planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-PLAN.md
@.planning/quick/260731-dgm-address-cdn-purge-reliability-diagnostic/260731-dgm-PLAN.md

<interfaces>
Current purge-step contract:

- The step is named `Purge release metadata from CDN cache`, has no step-level condition, and sits immediately before `Publish GitHub release`.
- Missing credentials warn and exit 0; preserve that behavior and never print credential values.
- The current curl uses `--fail-with-body`, `--retry 3`, `--retry-all-errors`, and captures combined stdout/stderr in `PURGE_RESPONSE`. This is the defect: retry response bodies can be combined before JSON validation.
- `jq` is already used earlier in the same release job.
- Remote response text belongs in an ordinary diagnostic log line, never inside an Actions workflow-command annotation.

Current CI lint contract:

- `Lint GitHub Actions workflows` is currently a Linux-only step inside the three-OS `build-test-lint` matrix.
- Current actionlint is pinned to v1.7.11 but invoked with `-shellcheck=`, which disables all embedded-shell analysis.
- The complete workflow corpus passes `actionlint -ignore 'SC2129' .github/workflows/*.yml`.
- The complete workflow corpus passes yamllint 1.38.0 with the current inline policy.
- Ubuntu 24.04's signed repository exposes yamllint `1.33.0-1`; pin both the runner (`ubuntu-24.04`) and package version when retaining the existing apt-based installation path.

Validation pattern:

- No committed workflow-shell test framework exists.
- The preceding 260731-dgm plan established a disposable `/tmp` harness that extracts the purge `run:` body with yq and executes it with controlled dependencies. Reuse and strengthen that pattern instead of adding a repository testing subsystem.
</interfaces>
</context>

<source_audit>

| Source | ID | Item | Task | Status |
|--------|----|------|------|--------|
| GOAL | QG-01 | Address CDN purge and CI workflow lint findings atomically | 1, 2 | COVERED |
| REQ | EHO-01 | Separate retry-safe curl body/stderr capture, `-o`, `-w`, no `--fail-with-body`, guarded command substitution, HTTP + JSON validation | 1 | COVERED |
| REQ | EHO-02 | Retry transient 5xx but not permanent 403 | 1 | COVERED |
| REQ | EHO-03 | Distinct annotations and practical jq-status diagnostics | 1 | COVERED |
| REQ | EHO-04 | Standalone Linux workflow-lint job outside the OS matrix | 2 | COVERED |
| REQ | EHO-05 | Preserve embedded-shell analysis with only SC2129 ignored | 2 | COVERED |
| REQ | EHO-06 | Explicit yamllint version and root configuration | 2 | COVERED |
| REQ | EHO-07 | Simplify failure branches only after correcting response capture | 1 | COVERED |
| REQ | EHO-08 | Correct stale rationale and ShellCheck claims in 260730-sz8 PLAN.md | 2 | COVERED |
| REQ | EHO-09 | Deterministic shell validation plus workflow and Make checks | 1, 2 | COVERED |
| RESEARCH | — | Research phase | — | EXCLUDED — quick-task constraint says no research phase; repository patterns and installed tooling are sufficient |
| CONTEXT | C-01 | Preserve no-cgo/runtime behavior and avoid unrelated API changes | 1, 2 | COVERED — workflow/planning files only |
| CONTEXT | C-02 | Do not place prohibited internal repository information in artifacts | 1, 2 | COVERED |

</source_audit>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Make CDN purge retries evaluate one final response</name>
  <files>.github/workflows/release.yml</files>
  <behavior>
    - A local endpoint returning HTTP 503 with a non-JSON marker and then HTTP 200 with `{"success":true}` is requested twice and ends in the success branch, proving the first body did not contaminate the final body.
    - A local endpoint returning HTTP 403 is requested exactly once, emits the transport/HTTP annotation, never emits the API `success:false` annotation, and exits 0.
    - A transport failure maps to status `000`, emits the transport/HTTP annotation plus a useful ordinary-log diagnostic, and exits 0.
    - HTTP 200 with `success:false` emits the API-rejection annotation, distinct from the transport/HTTP annotation, and exits 0.
    - HTTP 200 with malformed JSON, an empty body, a missing `success` field, or unavailable jq produces a diagnosis specific to that condition and exits 0.
    - HTTP 200 with `success:true` is the only response that prints the purge-success message.
    - Every modeled failure leaves a parent `PUBLISH_REACHABLE` sentinel executable, and no bearer-token sentinel appears in output.
  </behavior>
  <action>
Per EHO-09, first create `/tmp/260731-eho-cdn-purge-test.sh` from the established disposable harness pattern. Extract the purge step's `run:` body with yq. For retry cases, wrap the real curl only to rewrite the Cloudflare URL to a local Python HTTP server; let real curl implement retries, and have the server record request counts. Cover the behavior matrix above, including a 503 body that makes concatenated content invalid before the final success JSON and a 403 request-count assertion. Add controlled transport and jq-availability cases. Run the harness against the current workflow and record the expected red results before editing.

Implement EHO-01 and EHO-02 by correcting capture before changing the failure branches:

- Create separate temporary files for the response body and curl stderr and register cleanup with a trap.
- Capture only curl's write-out value in `PURGE_CODE`; direct the response body with `-o` and use `-w '%{http_code}'`. Redirect curl stderr to its own file.
- Guard the assignment exactly with `|| PURGE_CODE=000` so a failed command substitution cannot terminate the step under `set -e`. Do not invert the command with `!` and then read `$?`, because that loses the original curl status.
- Remove `--fail-with-body` and do not substitute another `--fail` flag. Preserve `--retry 3`, `--retry-all-errors`, and the existing timeout. With no HTTP-fail flag, curl's transient-status retry policy still handles 408/429/5xx, while HTTP 403 completes with status 403 and does not become a retryable curl error.
- Preserve the request method, URL, authorization header, JSON request, credential guard, step name, environment, and position immediately before publication.

Only after the corrected capture passes the 503-then-200 and 403 harness cases, re-derive the branches per EHO-03 and EHO-07:

- Treat `000` and every non-success HTTP code as transport/HTTP failure. Compact body and stderr for ordinary logs, keep all remote text out of workflow-command annotations, emit a fixed transport/HTTP-specific annotation, then explicitly exit 0.
- For a successful HTTP code, check `command -v jq` before parsing. Run `jq -e '.success == true'` against the body file and preserve its original exit status with a guarded assignment; do not collapse all jq outcomes through `if ! ...`.
- Exit status 0 is success. For the false/null status, inspect valid JSON far enough to distinguish exact boolean `success:false` from a missing or wrongly typed field. Give exact `success:false` its own API-rejection annotation. Diagnose a missing field separately.
- Diagnose empty input and parse-invalid input separately when jq's status/stderr allows it, and diagnose unavailable jq explicitly. These validation failures remain non-fatal but must never print the success message.
- Keep explicit exit 0 statements for expected purge failures so publication reachability does not depend on the incidental status of the last diagnostic command.
  </action>
  <verify>
    <automated>bash /tmp/260731-eho-cdn-purge-test.sh &amp;&amp; actionlint .github/workflows/release.yml &amp;&amp; yq -e '.jobs.publish-release.steps[] | select(.name == "Purge release metadata from CDN cache") | .run | test("-o") and test("-w [^\\n]*%\\\\{http_code\\\\}") and test("\\\\|\\\\| PURGE_CODE=000") and (test("--fail-with-body") | not)' .github/workflows/release.yml</automated>
  </verify>
  <done>
The red-to-green harness passes every response, retry, jq, non-leakage, and publication-reachability case. A 503 followed by success consumes only the final body; 403 is attempted once; transport/HTTP and API false annotations are distinguishable; only HTTP success plus JSON success:true reports success; full actionlint passes the changed embedded Bash.
  </done>
</task>

<task type="auto">
  <name>Task 2: Isolate and pin workflow linting, then correct the historical plan</name>
  <files>.github/workflows/ci.yml, .yamllint, .planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-PLAN.md</files>
  <action>
Implement EHO-04 through EHO-06 by removing `Lint GitHub Actions workflows` from `build-test-lint` and creating a top-level `workflow-lint` job with no matrix dependency. Run it on `ubuntu-24.04`; give it its own checkout and Go setup, then keep actionlint pinned to v1.7.11.

Invoke actionlint over `.github/workflows/*.yml` with `-ignore 'SC2129'`. Do not use `-shellcheck=` anywhere: SC2129 is the one accepted style advisory, while every other ShellCheck finding in embedded Bash must continue to fail CI.

Move the existing inline yamllint rules unchanged into root `.yamllint`: extend default, disable document-start and line-length, allow one space before inline comments, allow only quoted `true`, `false`, and `on` under the truthy rule, and permit one terminal blank line. Retain the signed Ubuntu apt installation route and pin `yamllint=1.33.0-1`, the package version for the pinned Ubuntu 24.04 runner. Invoke `yamllint -c .yamllint .github/workflows/*.yml` so the same command works locally with any compatible yamllint installation. Print both linter versions in CI diagnostics.

Implement EHO-08 in the historical `260730-sz8-PLAN.md` without changing what that task delivered:

- Replace every claim that a failed left-hand test in an AND-list immediately aborts because of `set -e`. Explain that non-final AND-list commands are exempt from immediate errexit, but the list still returns non-zero; when it is the step's final command, that status becomes the step result.
- Keep the recommendation to use explicit `if` blocks, but tie it to unambiguous control flow and final status rather than a false immediate-errexit claim.
- Remove claims that actionlint or its ShellCheck integration proves runtime `set -e` behavior. Limit those claims to Actions schema/expression validation and the shell diagnostics ShellCheck actually reports.
- Update the related threat/verification wording at all matching locations, not only the first paragraph. Preserve historical hashes, scope, behavior matrix, and implementation instructions unrelated to those factual corrections.

After both workflow edits, rerun the Task 1 harness so the final combined workflow state is validated. Run the exact CI lint commands locally, followed by the repository Make checks required by AGENTS.md.
  </action>
  <verify>
    <automated>actionlint -ignore 'SC2129' .github/workflows/*.yml &amp;&amp; yamllint -c .yamllint .github/workflows/*.yml &amp;&amp; yq -e '(.jobs["workflow-lint"]["runs-on"] == "ubuntu-24.04") and (.jobs["workflow-lint"] | has("strategy") | not) and ([.jobs["build-test-lint"].steps[].name] | index("Lint GitHub Actions workflows") == null)' .github/workflows/ci.yml &amp;&amp; rg -q 'yamllint=1\\.33\\.0-1' .github/workflows/ci.yml &amp;&amp; rg -q -- '-ignore .SC2129.' .github/workflows/ci.yml &amp;&amp; ! rg -q -- '-shellcheck=' .github/workflows/ci.yml &amp;&amp; rg -q 'final command|final-command' .planning/quick/260730-sz8-fix-silent-cdn-cache-purge-skip-in-relea/260730-sz8-PLAN.md</automated>
  </verify>
  <done>
Workflow linting is a standalone Ubuntu 24.04 job, independent of every build matrix. Actionlint checks embedded shell with only SC2129 ignored, yamllint is explicitly pinned and reads root `.yamllint`, all workflows pass both commands, and the historical plan accurately separates shell exit behavior from static-linter guarantees.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| GitHub Actions runner → Cloudflare API | The bearer credential and purge request cross into an external service |
| Cloudflare response → runner logs / Actions annotations | Untrusted status, JSON, body, and stderr enter release diagnostics |
| GitHub-hosted runner → Go and Ubuntu package repositories | CI obtains the two workflow linters |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-eho-01 | Information Disclosure | purge diagnostics | mitigate | Never print the bearer token; the harness injects a sentinel and fails on leakage; annotations contain fixed text rather than remote content |
| T-eho-02 | Tampering | Cloudflare response | mitigate | Require a successful HTTP status and independently require JSON boolean `success:true`; preserve and branch on jq status rather than trusting parse output blindly |
| T-eho-03 | Spoofing | Actions annotations | mitigate | Compact remote body/stderr into ordinary logs only, preventing a response newline from creating a forged workflow command |
| T-eho-04 | Denial of Service | release publication | mitigate | Expected credential, transport, HTTP, and API-validation failures explicitly exit 0; the harness proves the publication sentinel remains reachable |
| T-eho-05 | Repudiation | purge outcome | mitigate | Distinct fixed annotations identify transport/HTTP versus API rejection, with condition-specific ordinary diagnostics for jq and response-shape failures |
| T-eho-SC | Tampering | workflow-linter installation | mitigate | Keep actionlint at v1.7.11; pin the runner and Ubuntu-signed yamllint package to `ubuntu-24.04` and `1.33.0-1`; introduce no npm, pip, or cargo package install |
</threat_model>

<verification>
Run all checks against the final combined tree:

1. `bash /tmp/260731-eho-cdn-purge-test.sh`
2. `actionlint -ignore 'SC2129' .github/workflows/*.yml`
3. `yamllint -c .yamllint .github/workflows/*.yml`
4. `make lint`
5. `make test`
6. `git diff --check`
7. Confirm `git diff --name-only` contains only the four declared files.

If an environment prerequisite prevents either Make target from completing, record the exact command, failure, and missing prerequisite in the summary as required by `AGENTS.md`; do not replace the workflow-specific gates with that explanation.
</verification>

<success_criteria>
- Curl response body and stderr are separate temporary files, write-out captures only the final HTTP code, and the assignment is guarded with `|| PURGE_CODE=000`.
- `--fail-with-body` is absent; real curl validation proves transient 5xx retry and single-attempt 403 behavior.
- Both HTTP status and JSON `success:true` are required, with distinct transport/HTTP and API-false annotations and practical jq diagnostics.
- Release publication remains reachable after every modeled purge failure, and credential values never enter logs.
- Workflow linting is a standalone Linux job with actionlint v1.7.11, ShellCheck enabled except SC2129, pinned yamllint, and root `.yamllint`.
- The prior 260730-sz8 plan contains correct AND-list and static-analysis explanations.
- The harness, actionlint, yamllint, relevant Make targets, and diff checks complete successfully or any environment-only Make limitation is documented exactly.
</success_criteria>

<output>
Create `.planning/quick/260731-eho-address-cdn-purge-and-ci-workflow-lint-f/260731-eho-SUMMARY.md` when done.
</output>
