---
phase: quick/260731-nkz
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - .github/workflows/release.yml
  - Makefile
  - scripts/dev-windows.ps1
  - README.md
  - CLAUDE.md
  - AGENTS.md
autonomous: true
requirements: [NKZ-01, NKZ-02, NKZ-03, NKZ-04, NKZ-05, NKZ-06, NKZ-07, NKZ-08, NKZ-09, NKZ-10]

must_haves:
  truths:
    - "Every handled CDN-purge failure emits one distinct warning plus one status/body diagnostic and exits successfully from the shared helper, so GitHub release publication remains reachable"
    - "An unset CF_ZONE_ID or CLOUDFLARE_API_TOKEN is reported as missing configuration even with set -u enabled; neither credential value is logged"
    - "A remote response contributes at most 512 bytes to diagnostics, CR/LF cannot inject annotations, and an ASCII marker states when more response bytes existed"
    - "Body sanitization cannot suppress the diagnostic or warning when locale-sensitive processing or a helper command fails"
    - "Any syntactically valid JSON, including null and false, passes the parse-only check; only a 2xx object with boolean success:true reports purge success"
    - "The redundant Cloudflare code-1012 branch and repeated post-helper exit statements are absent while retry behavior and branch-specific remediation remain intact"
    - "Contributor guidance derives the actionlint version from .actionlint-version, explains conditional Go toolchain switching accurately, and documents ShellCheck 0.9 or newer without an exact-version gate"
    - "Make and PowerShell retain their already-equivalent actionlint, ShellCheck/SC2129, and yamllint behavior without a new parity framework"
  artifacts:
    - path: ".github/workflows/release.yml"
      provides: "Compact, best-effort Cloudflare purge with bounded and failure-safe diagnostics"
      contains: "warn_purge_failure"
    - path: "Makefile"
      provides: "Accurate missing-Go guidance for the POSIX workflow-lint entry point"
      contains: "GOTOOLCHAIN"
    - path: "scripts/dev-windows.ps1"
      provides: "Accurate missing-Go guidance for the Windows workflow-lint entry point"
      contains: "GOTOOLCHAIN"
    - path: "README.md"
      provides: "User-facing lint prerequisites without a duplicated actionlint numeral"
      contains: ".actionlint-version"
    - path: "CLAUDE.md"
      provides: "Agent-facing conditional toolchain and ShellCheck guidance"
      contains: ".actionlint-version"
    - path: "AGENTS.md"
      provides: "Repository instructions aligned with the executable lint contract"
      contains: ".actionlint-version"
  key_links:
    - from: ".github/workflows/release.yml purge failure branches"
      to: ".github/workflows/release.yml warn_purge_failure"
      via: "one-argument calls use PURGE_CODE/PURGE_BODY_FILE globals and the helper terminates handled failures with exit 0"
      pattern: "warn_purge_failure"
    - from: ".github/workflows/release.yml warn_purge_failure"
      to: "GitHub Actions logs"
      via: "fixed ordinary-log prefix, LC_ALL=C single-line sanitization, 512-byte cap, and fixed warning text"
      pattern: "CDN purge diagnostic"
    - from: "Makefile and scripts/dev-windows.ps1"
      to: ".actionlint-version and .yamllint"
      via: "both resolve the same actionlint pin and invoke the same SC2129 exception plus repository-wide YAML policy"
      pattern: "actionlint-version|SC2129|yamllint"
    - from: "README.md, CLAUDE.md, and AGENTS.md"
      to: ".actionlint-version"
      via: "human prose names the pin file rather than copying its current numeric value"
      pattern: "\.actionlint-version"
---

<objective>
Validate and address release-workflow review findings 1-10 with the smallest complete set of workflow, lint-message, and documentation changes.

Purpose: Keep CDN purging safely best-effort and diagnostically useful while removing fragile shell repetition and making contributor toolchain guidance match Go's automatic toolchain behavior.

Output: One compact purge implementation, accurate cross-platform lint prerequisites, and explicit evidence that the suggested parity-enforcement code is unnecessary. Runtime, Go/Rust/Java API, FFI, CI-job, pin-file, and YAML-policy changes are outside this quick task.
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
@Makefile
@scripts/dev-windows.ps1
@README.md
@CLAUDE.md
@.actionlint-version
@.yamllint
@.planning/quick/260731-j64-address-release-purge-diagnostics-lint-d/260731-j64-SUMMARY.md

<interfaces>
Current purge contract:

- The `Purge release metadata from CDN cache` run body is 147 lines and immediately precedes `Publish GitHub release`; preserve `continue-on-error: true` and final successful completion.
- `warn_purge_failure` currently accepts `reason`, `http_status`, and an optional response-body file. Twelve failure sites call it and then repeat `exit 0`.
- The current response diagnostic flattens CR/LF and reads at most 512 bytes, but does not mark truncation and can terminate under `set -euo pipefail` if its sanitization pipeline fails.
- Purge success requires a 2xx response and exact boolean `.success == true`. Retry flags are `--retry 3`, `--retry-all-errors`, and `--max-time 180`.
- Warning annotations contain fixed remediation only; untrusted response bytes stay in an ordinary log line behind the `CDN purge diagnostic:` prefix. Preserve this injection boundary and never print the bearer token.

Current lint contract:

- `.actionlint-version` is the executable pin source. Make and PowerShell both read it, build the same Go module path, pass an explicit ShellCheck path, ignore SC2129, and run `yamllint -c .yamllint .`.
- Baseline verification on 2026-07-31 passed both `make lint-workflows` and `pwsh -NoProfile -File scripts/dev-windows.ps1 -Task lint-workflows`; both reported ShellCheck 0.11.0, yamllint 1.38.0, and the same actionlint module.
- Go's current `GOTOOLCHAIN` default is `auto`. Go 1.21+ can therefore select the newer toolchain requested by the pinned actionlint module; a locally installed Go 1.24+ toolchain is required only when automatic switching is unavailable or disabled, including `GOTOOLCHAIN=local` or an older pinned toolchain.
</interfaces>
</context>

<finding_validation>

| Finding | Verdict | Current-tree evidence | Planned disposition |
|---------|---------|-----------------------|---------------------|
| 1 | CONFIRMED | The 147-line run body contains a message-qualified code-1012 jq branch even though the bounded raw body exposes the Cloudflare JSON and existing permission/generic branches provide remediation. | Task 1 removes only the 1012 special case. |
| 2 | CONFIRMED | Eight documentation locations and the Make/PowerShell missing-Go hints state Go 1.24+ unconditionally; `go env GOTOOLCHAIN` is `auto`. | Task 2 documents the conditional requirement and fixes both entry-point hints. |
| 3 | CONFIRMED | `rg` finds the literal pin on eight human-prose lines across README.md, CLAUDE.md, and AGENTS.md in addition to the authoritative pin file. | Task 2 replaces prose numerals with references to `.actionlint-version`; the pin file remains unchanged. |
| 4 | CONFIRMED | Twelve failure call sites repeat `warn_purge_failure ...` followed by `exit 0`; status and body-file values are passed repeatedly. | Task 1 moves handled-failure termination into the helper and reads the two globals there. |
| 5 | VALIDATED — NO CHANGE | Make and PowerShell currently use the same version file, module path, explicit ShellCheck path, SC2129 ignore, `.yamllint`, and repository root; both baseline commands pass with identical tool/module output. | Reject a new wrapper/config/parity-test subsystem: there is no drift to repair, and shared pin/config files already centralize the values worth centralizing. Task 2's automated check protects parity across its message-only edits. |
| 6 | CONFIRMED | Contributor docs require ShellCheck but give no minimum or recommended floor; the current local tool is 0.11.0. | Task 2 documents ShellCheck 0.9 or newer without adding an exact-version runtime gate. |
| 7 | CONFIRMED | Under `set -u`, `${CF_ZONE_ID}` and `${CLOUDFLARE_API_TOKEN}` abort when truly unset; a shell probe returned 127, while `${name:-}` returned zero. | Task 1 uses default-value expansion in preflight checks. |
| 8 | CONFIRMED | `head -c 512` silently cuts longer bodies and no truncation marker exists. | Task 1 appends fixed ASCII `...[truncated]` only when the source exceeds 512 bytes. |
| 9 | CONFIRMED | `jq -e '.'` returns 1 for valid `null` and `false`, while `jq empty` returns zero for both. | Task 1 changes only the syntax-validation invocation to `jq empty`; `.success == true` keeps `jq -e`. |
| 10 | CONFIRMED | With `set -euo pipefail`, a failing `tr` makes the current assignment exit before either diagnostic; a controlled probe returned 9. | Task 1 makes byte handling locale-stable and explicitly non-fatal. |

</finding_validation>

<source_audit>

| Source | ID | Item | Task | Status |
|--------|----|------|------|--------|
| GOAL | QG-01 | Validate findings 1-10 and implement only confirmed, useful corrections | 1, 2 | COVERED |
| REQ | NKZ-01 | Remove redundant code-1012 handling | 1 | COVERED |
| REQ | NKZ-02 | Make Go 1.24+ language conditional on toolchain switching | 2 | COVERED |
| REQ | NKZ-03 | Keep the numeric actionlint pin only in `.actionlint-version` | 2 | COVERED |
| REQ | NKZ-04 | Collapse helper arguments and repeated failure exits | 1 | COVERED |
| REQ | NKZ-05 | Assess Make/PowerShell parity enforcement | 2 | COVERED — observation validated; new enforcement rejected with executable evidence |
| REQ | NKZ-06 | Document ShellCheck 0.9+ | 2 | COVERED |
| REQ | NKZ-07 | Handle truly unset Cloudflare env variables under `set -u` | 1 | COVERED |
| REQ | NKZ-08 | Mark response truncation explicitly | 1 | COVERED |
| REQ | NKZ-09 | Parse JSON without rejecting false/null | 1 | COVERED |
| REQ | NKZ-10 | Prevent sanitizer failures from suppressing diagnostics | 1 | COVERED |
| RESEARCH | L0-01 | Existing Bash, jq, Make, and PowerShell patterns suffice; no dependency or external integration decision is introduced | 1, 2 | COVERED — Level 0 discovery |
| CONTEXT | CTX-01 | Preserve annotation-injection defense, credential secrecy, 512-byte remote-body bound, retries, publication reachability, and distinct warning messages | 1 | COVERED |
| CONTEXT | CTX-02 | Preserve no-cgo and public/FFI behavior | 1, 2 | COVERED — no runtime source touched |
| CONTEXT | CTX-03 | Prefer the least new code and Make targets for reproducibility | 1, 2 | COVERED |
| CONTEXT | CTX-04 | Keep public artifacts and commits free of prohibited internal repository information; integrate only by squash merge | 1, 2 | COVERED |

</source_audit>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Collapse and harden the best-effort CDN purge path</name>
  <files>.github/workflows/release.yml</files>
  <behavior>
    - A truly unset or empty Cloudflare zone/token produces the named missing-configuration warning, status `not-requested`, body `<empty>`, and exit code zero without exposing either value.
    - Missing jq or failed temporary-file creation produces its existing distinct warning plus the same two-part diagnostic and exit code zero.
    - Transport failure retains retry flags, reports `000` when curl supplies no status, includes any safe partial body, and exits zero.
    - HTTP 401, 403, 408/429/5xx, and catch-all non-2xx responses retain distinct remediation and a status/body diagnostic.
    - Empty, malformed, valid-null, valid-false, and valid non-success JSON responses are distinguished correctly: only malformed input receives the malformed-JSON warning; valid values without exact boolean success:true receive generic API-failure guidance.
    - A body longer than 512 bytes logs exactly the first 512 remote bytes after CR/LF flattening, followed by `...[truncated]`; a 512-byte body has no marker.
    - Sanitizer or byte-count command failure cannot prevent both the ordinary diagnostic and fixed warning annotation from being emitted.
    - A response containing newlines and workflow-command-looking text cannot create an extra annotation, and no credential sentinel appears in output.
    - A 2xx object with exact boolean success:true reports success and returns zero; all modeled outcomes leave the following Publish GitHub release step reachable.
  </behavior>
  <action>
Before editing, create a disposable `/tmp/260731-nkz-cache-purge-test.sh` behavior harness around the extracted `Purge release metadata from CDN cache` run body and demonstrate RED for the newly required cases. Stub curl, jq, mktemp, body contents, environment presence, and sanitizer commands without making network calls. Cover every behavior listed above, assert the existing retry flags, count exactly one real `::warning title=CDN cache not purged::` plus one `CDN purge diagnostic:` line per handled failure, and assert zero exit status. Do not commit the harness.

In `.github/workflows/release.yml`, initialize `PURGE_CODE` to `not-requested` and `PURGE_BODY_FILE` to empty before defining `warn_purge_failure`. Change that helper to accept only the fixed reason string, read status/body path from those globals, emit both diagnostics, and terminate handled failures itself with `exit 0`. Before the transport warning, normalize an empty curl status to `000`. Remove every now-redundant `exit 0` immediately following a helper call, but retain the explicit exit after the success message.

Keep untrusted body text out of the workflow annotation. In the ordinary diagnostic, read no more than 512 source bytes, flatten CR/LF under `LC_ALL=C`, and guard the complete sanitization pipeline with a non-fatal fallback so `set -euo pipefail` cannot interrupt the helper. Determine source length with a guarded, locale-stable byte count; validate numeric output before comparing it; append the portable fixed suffix `...[truncated]` only when the source contains more than 512 bytes. If a non-empty file cannot be sanitized, use a fixed safe fallback such as `<unreadable>` and still emit both lines.

Use `${CF_ZONE_ID:-}` and `${CLOUDFLARE_API_TOKEN:-}` in missing-value checks so a missing binding follows the diagnostic path under `set -u`. Preserve the existing named remediation, request URL/payload, credential headers, retry settings, response-file trap, status classes, 512-byte remote-data limit, and `continue-on-error: true`.

Replace the JSON syntax probe `jq -e '.'` with parse-only `jq empty`. Keep `jq -e '.success == true'` for the semantic success check. Delete the entire message-qualified code-1012 special-case query and warning; its JSON remains visible in the bounded diagnostic and the existing 403 or generic API-failure paths retain actionable permission/response guidance. Keep every other branch-specific warning string distinct. Run the disposable harness GREEN, then workflow lint.
  </action>
  <verify>
    <automated>bash /tmp/260731-nkz-cache-purge-test.sh &amp;&amp; make lint-workflows &amp;&amp; yq -e '.jobs["publish-release"].steps[] | select(.name == "Purge release metadata from CDN cache") | select(.["continue-on-error"] == true) | .run | select(contains("jq empty")) | select(contains("...[truncated]")) | select(contains("${CF_ZONE_ID:-}")) | select(contains("${CLOUDFLARE_API_TOKEN:-}")) | select(contains("LC_ALL=C")) | select(contains(".code == 1012") | not)' .github/workflows/release.yml &amp;&amp; git diff --check -- .github/workflows/release.yml</automated>
  </verify>
  <done>
All modeled purge outcomes satisfy the status/body/warning contract, failure sites delegate termination to one helper, JSON classification is correct for every valid value, remote bytes are bounded and visibly truncated, and publication remains reachable without weakening credential or annotation safety.
  </done>
</task>

<task type="auto">
  <name>Task 2: Correct lint toolchain guidance while preserving proven cross-platform parity</name>
  <files>Makefile, scripts/dev-windows.ps1, README.md, CLAUDE.md, AGENTS.md</files>
  <action>
Update only the missing-Go diagnostics in `Makefile`'s `lint-workflows` recipe and `scripts/dev-windows.ps1`'s `Lint-Workflows` preflight. State that contributors need Go 1.21+ with automatic toolchain switching enabled, or a locally installed Go 1.24+ toolchain when switching is unavailable/disabled; name `GOTOOLCHAIN=local` and an older pinned `GOTOOLCHAIN` value as examples. A completely missing `go` executable must still fail clearly. Do not alter module resolution, `-shellcheck`, `-ignore SC2129`, yamllint arguments, task aggregation, or command order: finding 5's baseline commands already prove behavioral parity, and adding a wrapper or enforcement subsystem would create maintenance work without correcting a current mismatch.

In `README.md`, `CLAUDE.md`, and `AGENTS.md`, replace every human-prose occurrence of the current actionlint numeral with `the version pinned in .actionlint-version` (using the existing Markdown code formatting). Preserve `.actionlint-version` itself as the sole numeric pin. Rewrite unconditional Go 1.24+ statements so Go 1.21+ remains the project baseline and automatic toolchain switching is the ordinary workflow-lint path; explain that Go 1.24+ must be installed locally only when switching is disabled or unavailable, notably with `GOTOOLCHAIN=local` or an older pinned toolchain. Do not imply that a separately installed actionlint binary changes what repository targets execute.

Document ShellCheck 0.9 or newer as the supported/recommended minimum for workflow linting in all three contributor guides. Explain that CI and local installations may use newer versions and keep version output diagnostic-only; do not add an exact version pin or equality gate. Keep install commands, lint scope, `.yamllint` behavior, CI delegation, and the Go 1.21 library/runtime baseline accurate.

Verify dynamically that the value read from `.actionlint-version` appears nowhere in the three prose files, that Make and PowerShell still contain the same shared-pin/SC2129/yamllint contract, and that both workflow-lint entry points pass. Finish with the repository-required relevant lint and test checks; if an environmental prerequisite prevents a full command, record the exact command and output in the quick-task summary rather than weakening the acceptance criteria. Keep commits and generated planning artifacts free of prohibited internal repository information, and leave integration to the repository's required squash-merge flow.
  </action>
  <verify>
    <automated>set -eu
pin="$(sed -n '1p' .actionlint-version)"
test -n "${pin}"
require_match() {
  pattern=$1
  file=$2
  if ! rg -Uqi -- "${pattern}" "${file}"; then
    printf 'required pattern missing from %s: %s\n' "${file}" "${pattern}"
    return 1
  fi
}
for guide in README.md CLAUDE.md AGENTS.md; do
  if rg -Fq -- "${pin}" "${guide}"; then
    printf 'numeric actionlint pin leaked into contributor prose: %s\n' "${guide}"
    exit 1
  fi
  require_match '\.actionlint-version' "${guide}"
  require_match 'ShellCheck (0\.9\+|0\.9 or newer|&gt;= ?0\.9)' "${guide}"
  require_match 'automatic toolchain switching' "${guide}"
  require_match 'Go 1\.24\+' "${guide}"
  require_match '(?s)(Go 1\.24\+.{0,240}(only|when)|(only|when).{0,240}Go 1\.24\+)' "${guide}"
  require_match 'GOTOOLCHAIN=local' "${guide}"
  require_match '(?s)(older.{0,80}pinned.{0,80}toolchain|toolchain.{0,80}pinned.{0,80}older|GOTOOLCHAIN.{0,80}pinned.{0,80}older)' "${guide}"
done
for entry in Makefile scripts/dev-windows.ps1; do
  if rg -Fq -- "${pin}" "${entry}"; then
    printf 'numeric actionlint pin bypasses shared pin file: %s\n' "${entry}"
    exit 1
  fi
  require_match '\.actionlint-version' "${entry}"
  require_match 'github\.com/rhysd/actionlint/cmd/actionlint@' "${entry}"
  require_match 'shellcheck_?path' "${entry}"
  require_match '-shellcheck=' "${entry}"
  require_match 'SC2129' "${entry}"
  require_match '\.yamllint' "${entry}"
  require_match 'automatic toolchain switching' "${entry}"
  require_match 'Go 1\.21\+' "${entry}"
  require_match 'Go 1\.24\+' "${entry}"
  require_match '(?s)(Go 1\.24\+.{0,240}(only|when)|(only|when).{0,240}Go 1\.24\+)' "${entry}"
  require_match 'GOTOOLCHAIN=local' "${entry}"
  require_match '(?s)(older.{0,80}pinned.{0,80}toolchain|toolchain.{0,80}pinned.{0,80}older|GOTOOLCHAIN.{0,80}pinned.{0,80}older)' "${entry}"
  case "${entry}" in
    Makefile)
      require_match 'yamllint_path.*-c \.yamllint \.' "${entry}"
      ;;
    scripts/dev-windows.ps1)
      require_match 'Invoke-CommandChecked -Name "yamllint" -Arguments @\("-c", "\.yamllint", "\."\)' "${entry}"
      ;;
  esac
done
make lint-workflows
pwsh -NoProfile -File scripts/dev-windows.ps1 -Task lint-workflows
git diff --check -- Makefile scripts/dev-windows.ps1 README.md CLAUDE.md AGENTS.md</automated>
  </verify>
  <done>
The five files describe Go/toolchain switching and ShellCheck compatibility accurately, no human guide duplicates the actionlint numeric pin, both lint implementations retain their proven shared semantics, and no parity-only framework or runtime/FFI change is introduced.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| GitHub Actions runner -> Cloudflare API | Repository configuration and a bearer credential are used in an external purge request |
| Cloudflare response -> public Actions log | Untrusted remote bytes become maintainer-visible diagnostics |
| Purge step -> release publication | Purge control flow determines whether the following publication step remains reachable |
| Local/CI hosts -> actionlint Go module | Workflow linting resolves and executes the repository-pinned module with host tools |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-nkz-01 | Information Disclosure | purge request/diagnostics | mitigate | Never print token or zone values; keep response bytes out of annotations and assert a credential sentinel is absent in the harness |
| T-nkz-02 | Spoofing / Tampering | Cloudflare response log rendering | mitigate | Prefix ordinary diagnostics, flatten CR/LF under C locale, cap remote data at 512 bytes, and append only a fixed truncation marker |
| T-nkz-03 | Denial of Service | diagnostic helper | mitigate | Guard sanitation and byte counting, centralize handled-failure exit zero, retain continue-on-error, and verify every modeled failure leaves publication reachable |
| T-nkz-04 | Tampering | Cloudflare success classification | mitigate | Require 2xx plus exact boolean success:true while using parse-only jq validation for all syntactically valid JSON |
| T-nkz-05 | Repudiation | contributor lint environment | mitigate | Log tool versions, document supported minimum/toolchain selection, and preserve both reproducible entry points without brittle equality gates |
| T-nkz-SC | Tampering | package-manager installs | accept | No npm, pip, cargo, or new Go dependency installation is introduced; the existing actionlint pin and Go checksum path remain unchanged |
</threat_model>

<verification>
Run against the combined tree:

1. `bash /tmp/260731-nkz-cache-purge-test.sh`
2. `make lint-workflows`
3. `pwsh -NoProfile -File scripts/dev-windows.ps1 -Task lint-workflows`
4. `make lint`
5. `make test`
6. `git diff --check`
7. Confirm `git diff --name-only` contains only the six declared implementation/documentation files plus quick-task planning artifacts, and confirm no prohibited internal repository information appears in the diff or commit messages.
</verification>

<success_criteria>
- Findings 1-4 and 6-10 are implemented exactly as adjudicated and pass the disposable behavior harness plus workflow lint.
- Finding 5 is closed with recorded evidence that both commands already match; no speculative parity framework is added.
- Purge failure diagnostics preserve credentials, annotation safety, 512-byte remote-body bounds, retries, distinct messages, and release-publication reachability.
- Documentation contains no copy of the actionlint pin, keeps Go 1.21+ as the project baseline, explains conditional Go 1.24+ installation, and names ShellCheck 0.9+.
- Relevant repository checks pass, or the summary reports the exact unavailable prerequisite and command output without claiming success.
</success_criteria>

<output>
Create `.planning/quick/260731-nkz-validate-and-address-release-workflow-re/260731-nkz-SUMMARY.md` when done.
</output>
