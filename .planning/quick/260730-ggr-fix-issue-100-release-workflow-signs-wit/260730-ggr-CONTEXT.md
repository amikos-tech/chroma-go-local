# Quick Task 260730-ggr: Fix issue #100: release workflow signs with refs/heads/main identity instead of refs/tags/<version> when dispatched from main - Context

**Gathered:** 2026-07-30
**Status:** Ready for planning

<domain>
## Task Boundary

GitHub issue #100: `release.yml` runs via `workflow_dispatch` from `main`, builds artifacts from the checked-out tag, but signs them with the `refs/heads/main` cosign identity because the identity is derived from `github.workflow_ref` (the ref the workflow file itself was loaded from), not the ref that was actually checked out. This forces downstream consumers (chroma-go) to widen a cosign identity allowlist per affected release (v0.3.4, v0.3.5).

</domain>

<decisions>
## Implementation Decisions

### Fix approach
- Add a fail-loud CI guard to `release.yml`. Do NOT restructure into a reusable workflow (`workflow_call`) and do NOT rely on a process-only/documentation-only fix. The workflow itself must refuse to proceed when the signing identity would not match the released ref.

### Guard placement and behavior
- Insert the check immediately before the cosign/signing step (around `.github/workflows/release.yml:265,289` where `WORKFLOW_REF`/`IDENTITY` are used).
- Compare the ref embedded in `github.workflow_ref` against the ref that was actually checked out (on `workflow_dispatch`, that's `refs/tags/${{ github.event.inputs.release_tag }}`; on a tag-push trigger it's `github.ref`).
- On mismatch, hard-fail the job (non-zero exit) with a clear error message pointing at the correct dispatch invocation (`gh workflow run release.yml --ref <tag> -f release_tag=<tag>`) — do not sign or publish artifacts.

### Claude's Discretion
- Exact bash/shell implementation of the ref comparison (e.g. string comparison of `github.workflow_ref` suffix vs. expected `refs/tags/<tag>`), as long as it fails loud on mismatch and passes for the correct tag-based invocation.
- Whether to add a short comment/test note in the workflow explaining why the guard exists.

</decisions>

<specifics>
## Specific Ideas

Issue text confirms the mismatch via a decoded `v0.3.5` sigstore bundle:
```
URI:https://github.com/amikos-tech/chroma-go-local/.github/workflows/release.yml@refs/heads/main
1.3.6.1.4.1.57264.1.6:  refs/heads/main
```
Expected identity for a correctly-scoped release: `.../release.yml@refs/tags/<version>`.

Acceptance criteria from the issue: a release published through the intended path signs as `release.yml@refs/tags/<version>`; no new allowlist entries are needed per release going forward.

</specifics>

<canonical_refs>
## Canonical References

- GitHub issue #100 (this repo) — full problem writeup and suggested fixes
- Related: issue #99 (published tags are unprotected — root cause of the original re-tag that started this)
- Related downstream: amikos-tech/chroma-go#512, #514 (the widened cosign allowlist this fix aims to stop growing)

</canonical_refs>
