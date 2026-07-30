# Deferred Items — quick/260730-ggr

Out-of-scope discoveries found while executing this plan. Not fixed here per the
executor scope boundary (only issues directly caused by this task's changes are auto-fixed).

## 1. Pre-existing shellcheck SC2129 in `.github/workflows/ci.yml`

- **Found during:** plan-level verification (`actionlint .github/workflows/*.yml`)
- **Detail:** `.github/workflows/ci.yml:313:9` — `SC2129:style:15:3: Consider using
  { cmd1; cmd2; } >> file instead of individual redirects`
- **Why deferred:** `ci.yml` is byte-identical to the plan base commit
  (`git diff --name-only 5e5ded0 HEAD -- .github/workflows/ci.yml` is empty), and the same
  warning reproduces when linting the base version of the file in isolation. It is unrelated
  to the release-signing guard and is a style-only finding (no behavioral or security impact).
- **Suggested follow-up:** separate `[CLN]` issue — group the `>> "$GITHUB_OUTPUT"` /
  `>> "$GITHUB_STEP_SUMMARY"` redirects in that step into a single block.

> Note: `actionlint .github/workflows/release.yml` (the file this plan changed) is clean.
> The only `actionlint` non-zero exit across the workflow directory comes from this
> pre-existing `ci.yml` finding.
