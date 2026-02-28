# Java Bindings Scaffold

This directory contains a minimal Java scaffold for `local-go-chroma` native bindings.

Modules:

- `core` (Java 17): shared API and exceptions
- `jna` (Java 17): JNA-based FFI backend
- `panama` (Java 22): Foreign Function & Memory API backend

## Build

```bash
gradle --no-daemon :core:build :jna:build :panama:build -x test
```

## Smoke tests

Set `CHROMA_LIB_PATH` to a built shim binary (`libchroma_shim.so`, `libchroma_shim.dylib`, or `chroma_shim.dll`) and run:

```bash
gradle --no-daemon :jna:test
gradle --no-daemon :panama:test
```
