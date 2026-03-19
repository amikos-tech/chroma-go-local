# External Integrations

**Analysis Date:** 2026-03-19

## APIs & External Services

**Chroma Vector Database:**
- Chroma 1.5.5 - Core vector database engine
  - SDK/Client: Multiple Rust crates (chroma-frontend, chroma-config, chroma-system, chroma-types, chroma-log, chroma-index, chroma-segment, chroma-sysdb, chroma-sqlite)
  - Integration: Compiled directly into Rust FFI shim; no external API calls
  - Documentation: https://github.com/chroma-core/chroma.git (tag: 1.5.5)

**OpenTelemetry (Optional):**
- OpenTelemetry distributed tracing
  - Configuration: Via `WithOpenTelemetry(endpoint, serviceName)` in `config.go`
  - Env vars: `OTelEndpoint` and `OTelServiceName` can be set through YAML config
  - Flow: Go config options pass through to Rust shim via YAML, which forwards to Chroma runtime

## Data Storage

**Databases:**
- SQLite (embedded)
  - Connection: Local filesystem at `persist_path/sqlite_filename`
  - Client: sqlx 0.8.3 (Rust), configured via Figment YAML
  - Default file: `chroma.sqlite3` in persist directory
  - Configuration options: `PersistPath`, `SQLiteFilename` (see `config.go`)

**File Storage:**
- Local filesystem only
  - Persist path: Configurable via `WithPersistPath()` or YAML `persist_path`
  - Default: `./chroma`
  - Contains SQLite database, HNSW indices, write-ahead logs (WAL)
  - Also stores HNSW metadata in serialized format

**Caching:**
- None (in-memory only via Rust runtime)
- No external cache layer; all caching happens in Chroma runtime memory

## Authentication & Identity

**Auth Provider:**
- Custom - No external authentication
- Implementation: None - Direct embedded execution
- Server runs on localhost (default 127.0.0.1:8000) with no built-in auth
- CORS origins configurable via `WithCORSAllowOrigins()` in `config.go`

## Monitoring & Observability

**Error Tracking:**
- None - Errors returned via error codes and last-error string mechanism
- Error codes defined in `shim/src/lib.rs`: SUCCESS, ERROR_NULL_INPUT, ERROR_INVALID_UTF8, ERROR_CONFIG_PARSE, ERROR_SERVER_START, ERROR_INVALID_HANDLE, ERROR_RUNTIME_CREATE, ERROR_ALREADY_STOPPED, ERROR_OPERATION_FAILED

**Logs:**
- Console/stdout only
- No structured logging framework integrated
- Chroma runtime logs to stdout; access via parent process capture

**Distributed Tracing:**
- OpenTelemetry endpoint can be configured but is optional
- Passed through Go config to Rust shim via YAML configuration

## CI/CD & Deployment

**Hosting:**
- GitHub (source)
- Cross-platform builds: Linux, macOS, Windows (GitHub Actions runners)

**CI Pipeline:**
- GitHub Actions (`.github/workflows/ci.yml`)
  - Triggers: push to main, pull requests
  - Runs: Multi-OS matrix (ubuntu-latest, macos-latest, windows-latest)
  - Steps: Build (Rust debug), test (Go, Java JNA, Java Panama), lint (golangci-lint, cargo clippy)

**Release Pipeline:**
- GitHub Actions (`.github/workflows/release.yml`)
  - Triggers: Push to version tags (v*), workflow_dispatch for backfill
  - Builds release artifacts on all platforms
  - Cosign v3 bundle signing (based on recent commit history)
  - Uploads to GitHub Releases

**Dependency Management:**
- Cargo.lock for Rust (locked, committed)
- go.mod/go.sum for Go (locked via go.mod)
- gradle dependencies in build.gradle.kts (transitive from Maven Central)

## Environment Configuration

**Required env vars:**
- `CHROMA_LIB_PATH` - Path to compiled Rust shim library (libchroma_shim.so, libchroma_shim.dylib, or chroma_shim.dll)
  - Auto-set by Makefile during testing
  - Set by CI/CD workflows

**Optional env vars:**
- `CARGO_TARGET_DIR` - Override Rust build artifact directory
- `RELEASE_VERSION` - For Java artifact versioning
- `GITHUB_TOKEN` - GitHub Actions (for release workflows)

**Secrets location:**
- GitHub Actions secrets (GITHUB_TOKEN implicitly available)
- No .env file pattern used (no environment variables with secrets)
- Config passed via YAML strings or files

## Webhooks & Callbacks

**Incoming:**
- None - Embedded runtime only, not a service accepting webhooks

**Outgoing:**
- None - No outbound integrations

## Network & Communication

**HTTP Server:**
- Embedded HTTP server (Rust tokio)
- Runs on configurable `ListenAddress` and `Port` (default 127.0.0.1:8000)
- Configuration: `WithPort()`, `WithListenAddress()` in `config.go`
- CORS support via `WithCORSAllowOrigins()`
- Max payload size: configurable via `WithMaxPayloadSize()` (default 40 MB)

**FFI Boundaries:**
- Go↔Rust: purego calling Rust FFI symbols (chroma_server_*, chroma_embedded_*)
- Java↔Rust: JNA (Java 17) or Panama FFI (Java 22) calling same Rust symbols
- All calls marshalled through C-compatible types (pointers, byte arrays, handles)

## Dependency Updates

**Pinned Versions:**
- Chroma: 1.5.5 (Git tag, not updated unless explicitly bumped)
- Rust toolchain: 1.93.1 (pinned in CI)
- Protoc: 31.x (pinned in CI, required for Chroma dependencies)
- Go: 1.21+ (flexible, from go.mod)
- Java: 17 (JNA) and 22 (Panama) - both tested in CI

---

*Integration audit: 2026-03-19*
