# Chroma 1.5.9 Native Export Comparison

RESULT: identical

- Comparison timestamp (UTC): `2026-08-02T12:37:17Z`
- Operating system and architecture: `Darwin arm64`
- Pre-migration source SHA: `c26f89c480578c0bc53513b5158c9f287aabd5a6`
- Post-migration source SHA inspected: `73e25696e854ebeb74d41da8b991f23364d7fff2`
- Built library: `shim/target/debug/libchroma_shim.dylib`
- Build command: `make build`
- Build exit status: `0`
- Inspector executable: `/usr/bin/nm`
- Inspector command: `LC_ALL=C /usr/bin/nm -gjU shim/target/debug/libchroma_shim.dylib | sed -n 's/^_\(chroma_.*\)$/\1/p' | sort -u`
- Inspector pipeline exit status (with `set -o pipefail`): `0`
- Before list: `.planning/phases/11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9/11-abi-before.exports`
- After list: `.planning/phases/11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9/11-abi-after.exports`
- Normalized exports in each list: `47`
- Before list SHA-256: `d1360de902b9d17c30c36e6785f2c808386a4c6fd3d102e6580bae2baf33c3f8`
- After list SHA-256: `d1360de902b9d17c30c36e6785f2c808386a4c6fd3d102e6580bae2baf33c3f8`
- Post-migration library SHA-256: `2904d1bf458e6ac84c6ef6c7a0c437ac6d2f8f991113ea3715b69c74a43f0703`
- Comparison command: `diff -u .planning/phases/11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9/11-abi-before.exports .planning/phases/11-migrate-rust-shim-from-chroma-1-5-5-to-1-5-9/11-abi-after.exports`
- Comparison exit status: `0`
- Comparison output: empty

The actual rebuilt Chroma 1.5.9-backed Mach-O library exports the same sorted `chroma_*` symbol set as the Chroma 1.5.5 baseline. The Mach-O C symbol prefix `_` was removed in both captures; no export was inferred from Rust source.
