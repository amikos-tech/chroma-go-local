# Technology Stack

**Analysis Date:** 2026-03-19

## Languages

**Primary:**
- Go 1.21+ - Main package and FFI bindings
- Rust 1.70+ - Shim layer (FFI exports and Chroma integration)
- Java 17+ (JNA path), Java 22+ (Panama path) - Scaffold bindings

**Secondary:**
- Python - Project configuration and testing (see Makefile and scripts)
- YAML - Configuration format for Chroma server

## Runtime

**Environment:**
- Go runtime (native binary)
- Rust runtime (compiled to shared library: `.so`, `.dylib`, `.dll`)
- Java Virtual Machine (JDK 17 or 22 depending on binding)

**Package Manager:**
- Go: go modules (go.mod/go.sum)
- Rust: Cargo with locked dependencies (Cargo.lock)
- Java: Gradle 9+ with Kotlin DSL

## Frameworks

**Core:**
- Chroma 1.5.5 - Vector database/embedding engine (Git dependency from `https://github.com/chroma-core/chroma.git` at tag `1.5.5`)

**Go FFI:**
- ebitengine/purego v0.9.1 - Pure Go FFI (no cgo) for calling Rust shim

**Rust Runtime:**
- tokio 1.41 - Async runtime with features: fs, macros, rt-multi-thread, time, io-util, signal, sync
- figment 0.10.12 - Configuration management with YAML and env parsing
- serde 1.x - Serialization framework
- serde_json 1.x - JSON serialization
- serde-pickle 1.x - Pickle format serialization (for HNSW metadata)
- sqlx 0.8.3 - Async SQL toolkit with tokio runtime and SQLite support

**Testing:**
- Go: testify v1.11.1, gopter v0.2.11 (property-based testing)
- Rust: criterion 0.5 (benchmarking), proptest 1.6 (property-based testing), tempfile 3 (test fixtures)
- Java: JUnit 5 (jupiter platform)

**Build/Dev:**
- golangci-lint - Go linting
- cargo clippy - Rust linting
- protoc 31.x - Protocol Buffers compiler (required for Chroma dependencies)
- goimports - Go import management
- gofmt - Go code formatting
- cargo fmt - Rust formatting

## Key Dependencies

**Critical:**
- chroma-frontend, chroma-config, chroma-system, chroma-types, chroma-log, chroma-index, chroma-segment, chroma-sysdb, chroma-sqlite (all from Chroma 1.5.5 Git tag) - Core vector database functionality
- sqlx 0.8.3 - SQLite async access; required for embedded database operations
- tokio 1.41 - Async task scheduling and concurrency; powers Rust shim runtime

**Infrastructure:**
- libc 0.2 - C standard library FFI bindings
- pkg/errors v0.9.1 - Error handling with context
- golang.org/x/sys v0.6.0 - Platform-specific system calls (Windows support)
- net.java.dev.jna:jna 5.14.0 - JNA bindings for Java 17 fallback path
- leanovate/gopter v0.2.11 - Property-based testing for Go

## Configuration

**Environment:**
- Go: ServerConfig struct with builder pattern (see `config.go`)
- Rust: Figment-based YAML + environment variable configuration
- Java: CHROMA_LIB_PATH environment variable for runtime artifact location

**Build:**
- `.golangci.yml` - Go linting configuration with gci import formatter
- `Makefile` - Multi-language build orchestration (Go, Rust, Java)
- `shim/Cargo.toml` - Rust configuration with locked dependency versions
- `java/build.gradle.kts` - Gradle root build script
- `java/core/build.gradle.kts` - Core shared API models
- `java/jna/build.gradle.kts` - JNA bindings for Java 17
- `java/panama/build.gradle.kts` - Panama bindings for Java 22

**CI/CD:**
- GitHub Actions (`.github/workflows/ci.yml`, `.github/workflows/release.yml`)
- Cross-platform testing: Linux, macOS, Windows
- Protoc version: 31.x (required for Chroma dependencies)

## Platform Requirements

**Development:**
- Go 1.21+ with module support
- Rust 1.93.1+ with clippy component
- Java 17 (JNA) or Java 22 (Panama)
- Gradle 9+
- golangci-lint
- protoc 31.x
- goimports and gofmt (Go 1.21+ standard)
- cargo fmt and cargo clippy (Rust standard tools)
- macOS, Linux, or Windows with POSIX-compatible shell (Makefile)

**Production:**
- Shared library artifact (libchroma_shim.so, libchroma_shim.dylib, or chroma_shim.dll)
- Runtime artifact name: `chroma_shim` (platform-specific extension added)
- Requires filesystem access for SQLite database persistence
- Optional: OpenTelemetry collector endpoint for distributed tracing

---

*Stack analysis: 2026-03-19*
