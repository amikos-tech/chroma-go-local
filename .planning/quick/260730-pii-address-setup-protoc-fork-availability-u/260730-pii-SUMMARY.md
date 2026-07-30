---
status: incomplete
quick_id: 260730-pii
date: 2026-07-30
commit: 3d3a7e3
---

# Quick Task 260730-pii: Setup-protoc availability follow-up

## Completed

- Added `chroma/` to `.gitignore`, so the generated `chroma/chroma.sqlite3` test artifact is no longer included by `git add -A`.
- Corrected the prior quick-task plan to explicitly document the verified Windows handling, the external-fork availability risk, and the inherent review risk of a generated action bundle.
- Added an executable follow-up plan for an organization-owned fork, the four immutable action pins, and a weekly upstream-drift signal.

## Blocked

Creating `amikos/setup-protoc` could not be completed: the active GitHub identity does not have the organization permission needed to create that fork. The workflow callers deliberately remain unchanged because an Actions reference to a repository that does not exist would break CI and releases.

## Resume

After an `amikos` organization owner creates a public fork of `chroma-core/setup-protoc` at `amikos/setup-protoc`, resume this quick task to:

1. Verify that commit `df9e7872eaabfd0ddfafd9e27fe77c6229bc7d22` exists in the fork.
2. Repoint all four callers to the same full SHA under `amikos/setup-protoc`.
3. Add the weekly, read-only workflow that fails visibly when the fork falls behind `arduino/setup-protoc`.
4. Run `actionlint` for all affected workflows.

## Validation

- `git check-ignore -v chroma/chroma.sqlite3` confirms the generated database is ignored.
- `rg -n 'win32|T-p2k-04|T-p2k-05'` confirms the prior plan records the review findings.
- `make test` and `make lint` were not run: this completed portion changes only ignore and planning files; no Go, Rust, Java, or active workflow logic changed.
