# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.4.0] - UNRELEASED

### Changed
- Internal structure reorganized: Go implementation moved from repo root
  into `internal/runtime/` and `internal/library/` subtrees. The root
  package is now a thin facade that re-exports all public symbols via
  type aliases and function wrappers.

### Compatibility
- No breaking changes. Your existing code using
  `github.com/amikos-tech/chroma-go-local` requires zero modifications.
- Import path unchanged: `github.com/amikos-tech/chroma-go-local`
- All 110+ exported symbols preserved with identical signatures
- Verified via `go-apidiff` against v0.3.4 baseline

[0.4.0]: https://github.com/amikos-tech/chroma-go-local/compare/v0.3.4...v0.4.0
