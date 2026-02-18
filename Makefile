.PHONY: build build-debug build-release clean test test-go test-rust test-all bench bench-go bench-rust lint lint-go lint-rust fmt fmt-go fmt-rust help

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    LIB_EXT := dylib
    LIB_NAME := libchroma_go_shim.dylib
else ifeq ($(UNAME_S),Windows_NT)
    LIB_EXT := dll
    LIB_NAME := chroma_go_shim.dll
else
    LIB_EXT := so
    LIB_NAME := libchroma_go_shim.so
endif

SHIM_DIR := shim
SHIM_TARGET_DEBUG := $(SHIM_DIR)/target/debug/$(LIB_NAME)
SHIM_TARGET_RELEASE := $(SHIM_DIR)/target/release/$(LIB_NAME)

help:
	@echo "Chroma Go Shim Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build the Rust shim in debug mode"
	@echo "  build-debug   - Build the Rust shim in debug mode"
	@echo "  build-release - Build the Rust shim in release mode"
	@echo "  test          - Run Go tests (requires debug build)"
	@echo "  test-go       - Run Go tests (requires debug build)"
	@echo "  test-rust     - Run Rust shim tests"
	@echo "  test-all      - Run Go and Rust tests"
	@echo "  test-release  - Run Go tests with release build"
	@echo "  bench         - Run Go and Rust benchmarks"
	@echo "  bench-go      - Run Go benchmarks"
	@echo "  bench-rust    - Run Rust criterion benchmarks"
	@echo "  lint          - Run linters for Go and Rust"
	@echo "  lint-go       - Run golangci-lint"
	@echo "  lint-rust     - Run cargo clippy"
	@echo "  fmt           - Format Go and Rust code"
	@echo "  fmt-go        - Format Go code with gofmt and goimports"
	@echo "  fmt-rust      - Format Rust code with cargo fmt"
	@echo "  clean         - Clean build artifacts"
	@echo ""
	@echo "Environment variables:"
	@echo "  CHROMA_LIB_PATH - Path to the shared library (auto-set during tests)"

build: build-debug

build-debug:
	cd $(SHIM_DIR) && cargo build
	@echo "Built debug library at $(SHIM_TARGET_DEBUG)"

build-release:
	cd $(SHIM_DIR) && cargo build --release
	@echo "Built release library at $(SHIM_TARGET_RELEASE)"

test: test-go

test-go: build-debug
	CHROMA_LIB_PATH=$(shell pwd)/$(SHIM_TARGET_DEBUG) go test -v ./...

test-rust:
	cd $(SHIM_DIR) && cargo test

test-all: test-go test-rust

test-release: build-release
	CHROMA_LIB_PATH=$(shell pwd)/$(SHIM_TARGET_RELEASE) go test -v ./...

bench: bench-go bench-rust

bench-go: build-debug
	CHROMA_LIB_PATH=$(shell pwd)/$(SHIM_TARGET_DEBUG) go test -run '^$$' -bench . -benchmem ./...

bench-rust:
	cd $(SHIM_DIR) && cargo bench --bench ffi_bench

clean:
	cd $(SHIM_DIR) && cargo clean
	rm -rf ./chroma_test_data

lint: lint-go lint-rust

lint-go:
	golangci-lint run ./...

lint-rust:
	cd $(SHIM_DIR) && cargo clippy -- -D warnings

fmt: fmt-go fmt-rust

fmt-go:
	gofmt -w .
	goimports -w .

fmt-rust:
	cd $(SHIM_DIR) && cargo fmt
