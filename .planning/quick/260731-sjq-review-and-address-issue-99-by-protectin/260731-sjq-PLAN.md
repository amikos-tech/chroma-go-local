---
phase: quick/260731-sjq
plan: 01
type: execute
wave: 1
depends_on: []
files_modified: []
autonomous: true
requirements: [ISSUE-99]

must_haves:
  truths:
    - "Published tags matching refs/tags/v* cannot be updated, force-pushed, or deleted."
    - "New refs/tags/v* tags can still be created because the ruleset has no creation restriction."
    - "Exactly one active repository ruleset named 'Protect published version tags' owns the intended v* protection, with no bypass actors."
    - "The release workflow remains unchanged, so legitimate same-tag retries and backfills are not rejected by a sum.golang.org existence check."
  artifacts:
    - path: "GitHub repository ruleset: Protect published version tags"
      provides: "Active tag protection for refs/tags/v*"
      contains: "update, deletion, non_fast_forward"
  key_links:
    - from: "ruleset conditions.ref_name.include"
      to: "refs/tags/v*"
      via: "exact repository ruleset ref-name condition"
      pattern: "refs/tags/v\\*"
    - from: "ruleset rules"
      to: "published tag immutability"
      via: "update, deletion, and non_fast_forward rule types with no bypass actors"
      pattern: "update|deletion|non_fast_forward"
---

<objective>
Protect published version tags by creating one active GitHub repository ruleset for `refs/tags/v*` after a read-before-write duplicate check.

Purpose: Prevent another published Go module tag from being re-pointed or deleted, which would permanently conflict with the checksum already recorded by the public module ecosystem.
Output: One verified repository-level tag ruleset. No source or workflow changes.
</objective>

<execution_context>
@/Users/tazarov/.codex/get-shit-done/workflows/execute-plan.md
@/Users/tazarov/.codex/get-shit-done/templates/summary.md
</execution_context>

<context>
@AGENTS.md
@.planning/STATE.md
@.planning/config.json
@.github/workflows/release.yml

<interfaces>
- Use the repository REST endpoint `repos/{owner}/{repo}/rulesets` through `gh api`; placeholders must resolve from the current checkout rather than hard-coding a repository name.
- The accepted GitHub tag-ruleset representation stores the exact rule objects `{type: "update"}`, `{type: "deletion"}`, and `{type: "non_fast_forward"}`. For a tag target, the `update` rule has no `parameters`; `update_allows_fetch_and_merge` is branch-only.
- Creating a repository ruleset requires repository Administration write permission. The supplied live state says the current user has ADMIN permission, but execution must verify this before mutation.
- `.github/workflows/release.yml` intentionally accepts existing tags through `workflow_dispatch` for retries and backfills. Do not add a sum.golang.org or other fail-on-existing-tag preflight: it would reject valid reruns and can race with first publication.
</interfaces>
</context>

<source_audit>

| Source | ID | Item | Task | Status |
|--------|----|------|------|--------|
| GOAL | QG-01 | Protect published `v*` tags from rewrites and deletions | 1 | COVERED |
| REQ | ISSUE-99 | Add an active repository tag ruleset for `refs/tags/v*` | 1 | COVERED |
| REQ | ISSUE-99 | Prevent updates, force pushes, and deletions without blocking new tag creation | 1 | COVERED |
| RESEARCH | API-01 | Use the GitHub repository rulesets REST endpoint and current tag-rule schema | 1 | COVERED |
| CONTEXT | C-01 | Read before write and avoid duplicate or conflicting rulesets | 1 | COVERED |
| CONTEXT | C-02 | Verify the stored remote ruleset after creation | 2 | COVERED |
| CONTEXT | C-03 | Do not add a release-workflow checksum preflight or otherwise change `release.yml` | 2 | COVERED |
| CONTEXT | C-04 | Keep artifacts free of prohibited repository information; use squash merge if a PR is later needed | 1, 2 | COVERED |

</source_audit>

<tasks>

<task type="auto">
  <name>Task 1: Guard and create the immutable published-tag ruleset</name>
  <files>Remote artifact: GitHub repository ruleset `Protect published version tags`</files>
  <action>
Run from the repository root. Confirm `gh auth status` succeeds and `gh api 'repos/{owner}/{repo}' --jq '.permissions.admin'` returns `true`; stop without mutation if not.

Before any POST, list repository-owned tag rulesets with `GET repos/{owner}/{repo}/rulesets?includes_parents=false&amp;targets=tag&amp;per_page=100`, following pagination, then fetch each returned ruleset by ID so conditions, rules, enforcement, and bypass actors are available. Use the deterministic name `Protect published version tags`. Treat an existing ruleset as the desired no-op result only when it is active, targets tags, has `include: ["refs/tags/v*"]` and an empty exclude list, has no bypass actors, contains exactly the `update`, `deletion`, and `non_fast_forward` rule types, and the update rule's `parameters` field is absent or null.

If a non-equivalent ruleset has the deterministic name or overlaps `refs/tags/v*` through that exact include, `refs/tags/*`, or `~ALL`, stop before mutation and report its ID and full configuration. Do not create a duplicate, overwrite an existing ruleset, or delete anything. If one exact desired ruleset already exists, skip creation and continue to Task 2.

When no candidate exists, submit one POST to `repos/{owner}/{repo}/rulesets` with the recommended GitHub media type and API version `2026-03-10`. Send a JSON body with name `Protect published version tags`, `target: "tag"`, `enforcement: "active"`, `bypass_actors: []`, `conditions.ref_name.include: ["refs/tags/v*"]`, `conditions.ref_name.exclude: []`, and the exact rules `{type: "update"}`, `{type: "deletion"}`, and `{type: "non_fast_forward"}`. Do not add parameters to the tag `update` rule. Do not add a `creation` rule: new release tags must remain creatable. Capture the returned ruleset ID for diagnostics.

Do not edit `.github/workflows/release.yml`, query sum.golang.org, alter issue #99, or make any other repository-setting change. No PR is required for the remote ruleset; if later work introduces a PR, follow the repository's squash-merge-only rule.
  </action>
  <verify>
    <automated>set -euo pipefail
name='Protect published version tags'
pages="$(gh api 'repos/{owner}/{repo}/rulesets?includes_parents=false&amp;targets=tag&amp;per_page=100' --paginate --slurp)"
test "$(jq --arg name "${name}" '[.[][] | select(.name == $name)] | length' &lt;&lt;&lt;"${pages}")" -eq 1
id="$(jq -r --arg name "${name}" '.[][] | select(.name == $name) | .id' &lt;&lt;&lt;"${pages}")"
gh api "repos/{owner}/{repo}/rulesets/${id}" --jq '.id' | grep -Eq '^[0-9]+$'</automated>
  </verify>
  <done>
The authenticated administrator either creates exactly one desired ruleset or safely reuses an exact existing one; any collision stops before write, and the resulting ruleset is addressable by a single repository ruleset ID.
  </done>
</task>

<task type="auto">
  <name>Task 2: Verify the stored protection and unchanged release path</name>
  <files>.github/workflows/release.yml (read-only), remote repository ruleset (read-only)</files>
  <action>
Re-read the repository-owned tag rulesets from GitHub after Task 1 and resolve the single ruleset named `Protect published version tags` to its full configuration. Verify the remote stored representation, not the request or Task 1 response: name, active enforcement, tag target, exact include/exclude conditions, empty bypass list, the three exact rule types, and absent/null `parameters` on the `update` rule must all match. Also confirm no `creation` rule is present.

Verify `.github/workflows/release.yml` is byte-for-byte unchanged from `HEAD`. Record in the quick-task summary that no sum.golang.org preflight was added because same-tag release retries/backfills are intentional and an existence lookup would both reject them and race first publication. Record the ruleset ID and HTML link returned by GitHub, but do not expose credentials or unrelated repository metadata.

Do not test enforcement by attempting to move or delete a live tag. Structural verification through the read API is the acceptance method because a destructive probe would put published references at risk.
  </action>
  <verify>
    <automated>set -euo pipefail
name='Protect published version tags'
pages="$(gh api 'repos/{owner}/{repo}/rulesets?includes_parents=false&amp;targets=tag&amp;per_page=100' --paginate --slurp)"
test "$(jq --arg name "${name}" '[.[][] | select(.name == $name)] | length' &lt;&lt;&lt;"${pages}")" -eq 1
id="$(jq -r --arg name "${name}" '.[][] | select(.name == $name) | .id' &lt;&lt;&lt;"${pages}")"
detail="$(gh api "repos/{owner}/{repo}/rulesets/${id}")"
jq -e --arg name "${name}" '
  .name == $name and
  .target == "tag" and
  .enforcement == "active" and
  .conditions.ref_name.include == ["refs/tags/v*"] and
  .conditions.ref_name.exclude == [] and
  ((.bypass_actors // []) | length == 0) and
  (([.rules[].type] | sort) == ["deletion", "non_fast_forward", "update"]) and
  ([.rules[] | select(.type == "update") | (.parameters // null)] == [null])
' &lt;&lt;&lt;"${detail}" &gt;/dev/null
git diff --exit-code HEAD -- .github/workflows/release.yml</automated>
  </verify>
  <done>
The read API proves one active repository ruleset exactly protects `refs/tags/v*` against updates, force pushes, and deletion without blocking creation, while the release workflow remains unchanged and supports legitimate same-tag operations.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| local authenticated `gh` client -> GitHub repository administration API | An administrator credential performs the one authorized remote settings mutation |
| ruleset ref pattern -> Git reference namespace | The condition decides which tags are protected and which tag creations remain allowed |
| repository ruleset -> release workflow | Tag immutability must not be confused with rejecting legitimate same-tag release reruns |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-sjq-01 | Tampering | ruleset target and conditions | mitigate | Use the exact `tag` target and `refs/tags/v*` include, then verify the stored representation through a fresh GET |
| T-sjq-02 | Elevation of Privilege | ruleset bypass actors | mitigate | Submit and verify an empty bypass list so no actor is intentionally exempted |
| T-sjq-03 | Denial of Service | release tag creation | mitigate | Do not add a creation rule; new `v*` tags remain creatable |
| T-sjq-04 | Tampering | duplicate or overlapping rulesets | mitigate | Fetch full existing tag rulesets before POST, reuse only an exact match, and stop on any collision rather than overwriting or duplicating it |
| T-sjq-05 | Denial of Service | release retry/backfill path | mitigate | Leave `release.yml` unchanged and explicitly omit the fail-on-existing sum.golang.org preflight |
| T-sjq-SC | Tampering | package-manager installs | accept | No npm, pip, cargo, Go module, or other package installation is introduced |
</threat_model>

<verification>
The combined verification is the Task 2 read-back predicate plus `git diff --exit-code HEAD -- .github/workflows/release.yml`. It must return one exact active ruleset and no workflow diff. Do not use a live tag move or deletion as a probe.
</verification>

<success_criteria>
- One active repository ruleset named `Protect published version tags` targets exactly `refs/tags/v*`.
- The ruleset blocks updates, non-fast-forward changes, and deletion, has no bypass actors, and does not block creation.
- A rerun is idempotent: it reuses an exact ruleset and refuses to create over a conflict.
- `.github/workflows/release.yml` has no diff and contains no new sum.golang.org preflight.
</success_criteria>

<output>
Create `.planning/quick/260731-sjq-review-and-address-issue-99-by-protectin/260731-sjq-SUMMARY.md` when done.
</output>
