---
phase: quick/260731-fqz
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - .github/workflows/ci.yml
  - .github/workflows/release.yml
  - .yamllint
  - Makefile
  - .planning/STATE.md
  - .planning/quick/260731-eho-address-cdn-purge-and-ci-workflow-lint-f/260731-eho-SUMMARY.md
  - .planning/quick/260731-fqz-address-ci-lint-reproducibility-and-rele/260731-fqz-SUMMARY.md
autonomous: true
requirements:
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

must_haves:
  truths:
    - "CI uses the Ubuntu 24.04 runner's yamllint 1.38.0 without apt-get and records the actual yamllint, ShellCheck, and actionlint versions."
    - "CI fails clearly if its expected ShellCheck 0.9.0 executable is missing or drifts, and actionlint is given that executable explicitly instead of discovering it silently."
    - "make lint runs the same actionlint v1.7.11, SC2129-ignore, and repository-wide yamllint gate used by CI."
    - "Every Cloudflare purge failure is best-effort: expected failures warn and exit successfully, while continue-on-error keeps GitHub release publication reachable after unexpected shell failures."
    - "Cloudflare 401/403 messages point to authentication or configuration, while transport, rate-limit, and server failures report retry exhaustion separately."
    - "Only a 2xx response for which jq -e '.success == true' passes is reported as a successful purge."
    - "Active GSD state and summary text no longer claims the ineffective yamllint 1.33.0 apt pin controlled the executable."
  artifacts:
    - path: ".github/workflows/ci.yml"
      provides: "Package-install-free workflow lint job using explicit, versioned linter contracts"
    - path: "Makefile"
      provides: "Shared lint-workflows target included by make lint"
      contains: "lint-workflows"
    - path: ".yamllint"
      provides: "Repository-wide YAML policy excluding generated Rust build output"
      contains: "shim/target/"
    - path: ".github/workflows/release.yml"
      provides: "Short, structurally non-blocking Cloudflare purge step"
      contains: "continue-on-error: true"
  key_links:
    - from: ".github/workflows/ci.yml"
      to: "Makefile"
      via: "workflow-lint job invokes make lint-workflows"
      pattern: "make lint-workflows"
    - from: "Makefile"
      to: "actionlint v1.7.11 and ShellCheck"
      via: "version-pinned go run plus explicit -shellcheck path and -ignore SC2129"
      pattern: "actionlint/cmd/actionlint@v1\\.7\\.11"
    - from: ".github/workflows/release.yml purge step"
      to: "Publish GitHub release"
      via: "continue-on-error plus non-fatal expected branches"
      pattern: "continue-on-error: true"
---

<objective>
Make workflow linting reproducible in CI and through `make lint`, then reduce the Cloudflare cache-purge step to a small, robust best-effort operation.

Purpose: Remove a no-op package pin and hidden runner dependencies, while keeping release publication independent from both expected and unexpected cache-purge failures.
Output: Two atomic implementation commits, truthful uncommitted GSD state/summary updates, and a final quick-task summary.
</objective>

<execution_context>
@/Users/tazarov/.codex/get-shit-done/workflows/execute-plan.md
@/Users/tazarov/.codex/get-shit-done/templates/summary.md
</execution_context>

<context>
@AGENTS.md
@.planning/STATE.md
@.github/workflows/ci.yml
@.github/workflows/release.yml
@.yamllint
@Makefile
@.planning/quick/260731-eho-address-cdn-purge-and-ci-workflow-lint-f/260731-eho-SUMMARY.md

<interfaces>
- `jobs.workflow-lint` currently installs actionlint v1.7.11 with Go, runs an ineffective apt pin for yamllint 1.33.0, and then invokes the runner's preinstalled yamllint 1.38.0.
- The runner also supplies ShellCheck 0.9.0, but actionlint currently discovers it implicitly; local ShellCheck may be a different version.
- `Makefile` currently defines `lint: lint-go lint-rust`; there is no workflow-lint target.
- Repository-wide `yamllint -c .yamllint .` currently fails only on generated YAML under `shim/target/`.
- The release purge step precedes `Publish GitHub release`, uses two temporary files and several jq classification passes, and lacks step-level `continue-on-error`.
</interfaces>
</context>

<source_audit>

| Source | ID | Item | Task | Status |
|--------|----|------|------|--------|
| GOAL | QG-01 | Reproducible workflow linting and robust best-effort release purge | 1, 2 | COVERED |
| REQ | FQZ-01 | Remove ineffective yamllint apt pin; use/log preinstalled 1.38.0 | 1 | COVERED |
| REQ | FQZ-02 | Add structural continue-on-error to purge step | 2 | COVERED |
| REQ | FQZ-03 | Make ShellCheck explicit, deterministic, and versioned in CI | 1 | COVERED |
| REQ | FQZ-04 | Distinguish 401/403 from retryable transport/server failures | 2 | COVERED |
| REQ | FQZ-05 | Simplify purge around jq success gate | 2 | COVERED |
| REQ | FQZ-06 | Remove incorrect JQ_STATUS/stderr pairing | 2 | COVERED |
| REQ | FQZ-07 | Run reproducible workflow linting through make lint | 1 | COVERED |
| REQ | FQZ-08 | Ignore shim/target and lint YAML repo-wide | 1 | COVERED |
| REQ | FQZ-09 | Correct false active GSD claims | 3 | COVERED |
| REQ | FQZ-10 | Preserve warnings and publication after purge failures | 2 | COVERED |
| RESEARCH | — | Research phase | — | EXCLUDED — quick-task context explicitly forbids research; live tools and repository patterns are sufficient |
| CONTEXT | C-01 | Prefer Make targets and the least new code | 1, 2 | COVERED |
| CONTEXT | C-02 | Keep public artifacts free of prohibited internal repository information | 1, 2, 3 | COVERED |
| CONTEXT | C-03 | Commit implementation tasks atomically; do not commit GSD docs | 1, 2, 3 | COVERED |

</source_audit>

<tasks>

<task type="auto">
  <name>Task 1: Make workflow linting one explicit CI and Make contract</name>
  <files>.github/workflows/ci.yml, .yamllint, Makefile</files>
  <action>
Implement FQZ-01, FQZ-03, FQZ-07, and FQZ-08 as one shared lint path.

Add `lint-workflows` to Makefile's phony targets and help, and make `lint` depend on it alongside the existing Go and Rust targets. The workflow target must run actionlint exactly at v1.7.11 using `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.11`, pass an explicitly resolved ShellCheck executable with `-shellcheck=...`, retain `-ignore 'SC2129'`, and run yamllint as `yamllint -c .yamllint .`. Fail with a clear message before actionlint if ShellCheck or yamllint is absent. Log all three tool versions. Support optional expected-version environment values so CI can require ShellCheck 0.9.0 and yamllint 1.38.0 while local installations may differ visibly without being silently substituted.

In `ci.yml`, delete the entire apt-get/update/install path and the ineffective `yamllint=1.33.0-1` pin. Keep the pinned Ubuntu 24.04 job and Go setup, declare the expected preinstalled ShellCheck and yamllint versions, and invoke only `make lint-workflows` for the lint gate. Do not add a package manager, downloaded binary, wrapper script, or new action.

Add top-level `.yamllint` ignore coverage for `shim/target/`, preserving the existing rules. The Make target and CI must lint from repository root rather than limiting yamllint to tracked workflow files.

Run the task verification, then create one atomic implementation commit containing exactly `.github/workflows/ci.yml`, `.yamllint`, and `Makefile`, with message `fix(ci): make workflow lint reproducible`. Do not stage `.planning/**`.
  </action>
  <verify>
    <automated>make lint-workflows &amp;&amp; yamllint -c .yamllint . &amp;&amp; ! rg -q 'apt-get|yamllint=1\.33\.0-1' .github/workflows/ci.yml &amp;&amp; rg -q 'actionlint/cmd/actionlint@v1\.7\.11' Makefile &amp;&amp; rg -q -- '-ignore .SC2129.' Makefile &amp;&amp; rg -q -- '-shellcheck' Makefile &amp;&amp; rg -q 'shim/target/' .yamllint &amp;&amp; rg -q 'make lint-workflows' .github/workflows/ci.yml</automated>
  </verify>
  <done>
CI performs no linter package installation, asserts and logs its preinstalled ShellCheck 0.9.0 and yamllint 1.38.0, and calls the same v1.7.11 actionlint/SC2129/repo-wide yamllint target included in local `make lint`.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Reduce cache purging to a structurally best-effort result check</name>
  <files>.github/workflows/release.yml</files>
  <behavior>
    - Missing Cloudflare credentials emit the existing actionable warning and return success.
    - A temporary-file creation failure emits a fixed warning and returns success; any other unexpected shell failure is still non-blocking at the workflow level.
    - Curl transport failure emits a transport/retry warning and returns success.
    - HTTP 401 or 403 emits an authentication/zone/token-permission warning distinct from retry exhaustion.
    - HTTP 408, 429, or 5xx emits a transport/rate-limit/server warning stating retries were exhausted.
    - Other non-2xx statuses emit a fixed HTTP failure warning.
    - A 2xx body reports success only when jq -e '.success == true' passes; success:false, malformed JSON, empty JSON, and missing success warn and return success.
    - The following Publish GitHub release step remains reachable after every failure.
  </behavior>
  <action>
Before editing, create a disposable `/tmp/260731-fqz-cache-purge-test.sh` harness from the prior quick-task pattern. Extract the purge run body with yq, stub `curl`/`mktemp` only as needed, and cover the behavior matrix above. Keep the harness outside the repository.

Implement FQZ-02 and FQZ-04 through FQZ-06 and FQZ-10. Add `continue-on-error: true` directly to the `Purge release metadata from CDN cache` step so unexpected errors, including setup failures, cannot prevent the next publication step. Keep explicit warning-and-exit-zero handling for expected failures so maintainers retain useful annotations.

Reduce the shell to one guarded response-body temporary file, one curl call, one HTTP-status decision, and `jq -e '.success == true'` as the central and only success predicate. Let curl stderr remain visible in the ordinary log instead of capturing and later pairing it with unrelated jq statuses. Preserve the credential guard, request URL, authorization header, payload URLs, `--retry 3`, `--retry-all-errors`, and timeout.

Classify curl transport failure separately. For completed HTTP responses, give 401/403 an authentication/configuration warning; group 408, 429, and 5xx as retryable transport/rate-limit/server outcomes that failed after retries; give other non-2xx statuses a generic fixed HTTP warning. Only 2xx reaches jq. If the jq success predicate fails for any reason, emit one fixed API/response-validation warning and continue.

Delete the stderr temp file, `command -v jq` probe, warning helper, JQ_STATUS/JSON_STATUS variables, response-shape classifier, redundant jq passes, remote-body compaction pipelines, and associated branches. Do not echo the bearer token or place remote response text inside workflow-command annotations.

Run the task verification, then create one atomic implementation commit containing exactly `.github/workflows/release.yml`, with message `fix(release): simplify best-effort cache purge`. Do not stage `.planning/**`.
  </action>
  <verify>
    <automated>bash /tmp/260731-fqz-cache-purge-test.sh &amp;&amp; make lint-workflows &amp;&amp; yq -e '.jobs["publish-release"].steps[] | select(.name == "Purge release metadata from CDN cache") | .["continue-on-error"] == true' .github/workflows/release.yml &amp;&amp; rg -q \"jq -e ['\\\"]\\.success == true['\\\"]\" .github/workflows/release.yml &amp;&amp; ! rg -q 'JQ_STATUS|JSON_STATUS|SUCCESS_CLASS|PURGE_STDERR_FILE|command -v jq|warn_purge_validation' .github/workflows/release.yml</automated>
  </verify>
  <done>
The harness passes all credential, setup, transport, HTTP-category, JSON, secrecy, and publication-reachability cases; the step is explicitly continue-on-error; and the implementation has one jq success gate without the removed classifiers or mismatched diagnostics.
  </done>
</task>

<task type="auto">
  <name>Task 3: Correct active GSD claims and verify the combined result</name>
  <files>.planning/STATE.md, .planning/quick/260731-eho-address-cdn-purge-and-ci-workflow-lint-f/260731-eho-SUMMARY.md, .planning/quick/260731-fqz-address-ci-lint-reproducibility-and-rele/260731-fqz-SUMMARY.md</files>
  <action>
Implement FQZ-09 without rewriting historical intent. In the EHO summary, correct statements that say the yamllint 1.33.0 apt pin controlled the executable: record that the task intended that pin but Ubuntu 24.04's preinstalled yamllint 1.38.0 remained the command on PATH, and point to this quick task as the correction. Preserve its commit hashes, delivered behavior, and other historical facts.

Update the active STATE wording so it no longer attributes reproducible yamllint pinning to EHO; record this quick task's actual package-install-free, version-visible lint contract and simplified best-effort purge after verification. Create `260731-fqz-SUMMARY.md` with exact commands and results, the two implementation commits, and any environment-only limitation required by AGENTS.md.

Run the combined verification below. Leave the plan, both edited planning artifacts, and the new summary uncommitted. Confirm the two implementation commits contain no `.planning/**` files.
  </action>
  <verify>
    <automated>rg -q 'preinstalled yamllint 1\.38\.0' .planning/quick/260731-eho-address-cdn-purge-and-ci-workflow-lint-f/260731-eho-SUMMARY.md &amp;&amp; rg -q '260731-fqz' .planning/STATE.md &amp;&amp; make lint &amp;&amp; make test &amp;&amp; git diff --check &amp;&amp; test -z "$(git diff --cached --name-only -- .planning)"</automated>
  </verify>
  <done>
The active historical record is factually accurate, both implementation commits are atomic and exclude GSD docs, required Make checks pass or an exact environment limitation is recorded, and the uncommitted FQZ summary exists.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| GitHub runner → Go module source | CI obtains the version-pinned actionlint module |
| GitHub runner → Cloudflare API | Zone ID, bearer token, purge request, and response cross an external boundary |
| Cloudflare/curl output → public Actions log | Remote and transport diagnostics can become operator-visible output |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-fqz-01 | Tampering | workflow lint toolchain | mitigate | Pin actionlint to v1.7.11, assert CI's expected ShellCheck/yamllint versions, pass ShellCheck's resolved path explicitly, and remove the ineffective apt path |
| T-fqz-02 | Information Disclosure | purge diagnostics | mitigate | Never print the bearer token; keep workflow annotations fixed and do not embed remote response text |
| T-fqz-03 | Tampering | Cloudflare response | mitigate | Require both 2xx HTTP and jq-confirmed boolean success:true before reporting purge success |
| T-fqz-04 | Denial of Service | release publication | mitigate | Handle expected purge failures with warnings and exit 0, plus step-level continue-on-error for unexpected failures |
| T-fqz-05 | Repudiation | operator diagnostics | mitigate | Use distinct fixed warnings for auth/config, retryable transport/server, other HTTP, and API-validation failures |
| T-fqz-SC | Tampering | npm/pip/cargo installs | accept | No npm, pip, or cargo installation is introduced; actionlint uses an exact Go module version and runner tools are version-checked |
</threat_model>

<verification>
Run against the final combined tree:

1. `bash /tmp/260731-fqz-cache-purge-test.sh`
2. `make lint-workflows`
3. `make lint`
4. `make test`
5. `yamllint -c .yamllint .`
6. `git diff --check`
7. Inspect the two implementation commits and confirm they contain exactly Task 1's three files and Task 2's release workflow; `.planning/**` remains uncommitted.

If a Make target cannot complete because of an environment prerequisite, record the exact command, output, and missing prerequisite in the summary; workflow-specific gates must still pass.
</verification>

<success_criteria>
- CI has no apt-get or yamllint 1.33.0 pin and proves which preinstalled yamllint and ShellCheck versions actionlint uses.
- `make lint` includes the exact actionlint v1.7.11/SC2129 workflow gate and repo-wide yamllint ignores only generated `shim/target/`.
- The purge step is materially shorter, uses one jq success predicate, and has no stale jq-status/stderr diagnostic pairing.
- 401/403 warnings are distinct from retryable transport, rate-limit, and server warnings.
- Expected and unexpected purge failures cannot block GitHub release publication, and useful fixed warnings remain visible.
- Active GSD wording is truthful, implementation commits are atomic, and GSD artifacts remain uncommitted.
</success_criteria>

<output>
Create `.planning/quick/260731-fqz-address-ci-lint-reproducibility-and-rele/260731-fqz-SUMMARY.md` when done, and do not commit it or any other `.planning/**` file.
</output>
