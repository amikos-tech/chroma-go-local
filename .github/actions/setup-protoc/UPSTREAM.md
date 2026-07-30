# Vendored setup-protoc action

This directory is a deliberate local copy of the action that installs `protoc` in this repository's CI and release workflows.

## Provenance

- Source: `chroma-core/setup-protoc`
- Source commit: `df9e7872eaabfd0ddfafd9e27fe77c6229bc7d22`
- Upstream base: `arduino/setup-protoc@3ea1d70ac22caff0b66ed6cb37d5b7aadebd4623`
- License: MIT, included as `LICENSE`

The source commit contains the Node 24 runtime change and retry/backoff handling for GitHub release requests. Its `src/installer.ts` handles Windows through the `win32` platform branch and selects `protoc-<version>-win32.zip` or `protoc-<version>-win64.zip`.

## What is required at runtime

GitHub Actions reads `action.yml` and executes the generated `dist/index.js`. The adjacent TypeScript source, lockfile, and configuration are retained so that the generated file can be reviewed and rebuilt; they are not installed or executed by CI merely because this action is present.

At the recorded source commit, a clean rebuild does not reproduce the supplied bundle byte-for-byte. Both files are retained exactly as supplied, but `dist/index.js` is the authoritative runtime artifact. Review it directly alongside the source on every update; source equivalence alone is not sufficient.

## Updating or extracting this action

1. Review upstream changes from the recorded base and select a source commit.
2. Update the source, test, lockfile, license, and provenance together.
3. Run the upstream snapshot's tests, then from this directory run `npm ci --ignore-scripts`, `npx tsc`, and `npx ncc build`; review the source and the entire generated-bundle diff before committing it. Do not require a byte-for-byte match with the previous bundle.
4. Run `actionlint` and let the existing Ubuntu, macOS, and Windows CI matrix validate the local action.
5. Once the matrix has proved the action in practice, it may be extracted unchanged to an organization-owned repository. Update callers then to that repository's full commit SHA.

The scheduled `check-setup-protoc-upstream.yml` workflow compares the recorded base with the upstream default branch weekly. A failed run is a reminder to review, not an automatic update.
