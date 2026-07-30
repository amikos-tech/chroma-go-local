---
status: complete
quick_id: 260730-pii
date: 2026-07-30
commits: [3d3a7e3, 5d3d437, 3dd112f]
---

# Quick Task 260730-pii: Setup-protoc local action follow-up

## Completed

- Added `chroma/` to `.gitignore`, so the generated `chroma/chroma.sqlite3` test artifact is no longer included by `git add -A`.
- Corrected the prior quick-task plan to explicitly document the verified Windows handling, the external-fork availability risk, and the inherent review risk of a generated action bundle.
- Added the reviewed action locally under `.github/actions/setup-protoc`, including the exact executable bundle, TypeScript source, lockfile, configuration, license, and provenance.
- Repointed the three CI and one release callers to `./.github/actions/setup-protoc`, removing their external action-repository dependency.
- Added a weekly, manually dispatchable, read-only workflow that fails visibly when `arduino/setup-protoc` advances beyond the recorded base.

## Bundle review record

The local action retains the exact `dist/index.js` supplied by the reviewed source commit. A clean rebuild of that source does not reproduce the supplied bundle byte-for-byte, so `dist/index.js` remains the authoritative executable and must be reviewed directly with every update. `UPSTREAM.md` records this constraint and the update procedure.

## Validation

- The action source snapshot's Jest suite passed: 9 tests, including Windows archive-name coverage and retry handling.
- The local generated bundle ran in an isolated runtime directory, installed `libprotoc 31.0`, and passed `protoc --version`.
- The recorded upstream-base comparison currently returns `ahead_by: 0` and `status: identical`.
- `yq` parses all three affected workflows.
- `actionlint` passes for the added workflow and passes for all affected workflows when excluding existing `SC2129` in `ci.yml`; no new lint finding was introduced.
- `git check-ignore -v chroma/chroma.sqlite3` confirms the generated database is ignored.
- `make test` and `make lint` were not run: the implementation changes the local JavaScript action and workflow configuration, not Go, Rust, or Java source.

## Follow-up

The next push should validate the local action on the repository's existing Ubuntu, macOS, and Windows matrix. Once those jobs pass, this directory can be extracted unchanged to a dedicated organization repository and callers can move to its full commit SHA.
