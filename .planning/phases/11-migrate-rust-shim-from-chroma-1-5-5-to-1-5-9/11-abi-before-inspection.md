# Chroma 1.5.5 Native Export Baseline

- Inspection timestamp (UTC): `2026-08-02T11:43:11Z`
- Pre-migration SHA: `c26f89c480578c0bc53513b5158c9f287aabd5a6`
- Operating system: `Darwin`
- Built library: `shim/target/debug/libchroma_shim.dylib`
- Build command: `make build`
- Build exit status: `0`
- Inspector executable: `/usr/bin/nm`
- Inspector command: `LC_ALL=C nm -gjU shim/target/debug/libchroma_shim.dylib | sed -n 's/^_\(chroma_.*\)$/\1/p' | sort -u`
- Inspector pipeline exit status (with `set -o pipefail`): `0`
- Observed normalized exports: `47`
- Export list: `11-abi-before.exports`
- Export list SHA-256: `d1360de902b9d17c30c36e6785f2c808386a4c6fd3d102e6580bae2baf33c3f8`

The export list contains only names observed in the built Mach-O dynamic library. The single Mach-O C symbol prefix `_` was removed before sorting; no names were inferred from Rust source annotations.
