use std::ffi::CString;

use chroma_go_shim::{
    chroma_embedded_free, chroma_embedded_get_max_batch_size, chroma_embedded_heartbeat,
    chroma_embedded_start_from_string, SUCCESS,
};
use criterion::{black_box, criterion_group, criterion_main, Criterion};
use tempfile::TempDir;

struct EmbeddedFixture {
    handle: *mut std::ffi::c_void,
    _temp_dir: TempDir,
}

impl EmbeddedFixture {
    fn new() -> Self {
        let temp_dir = tempfile::tempdir().expect("tempdir should be created");
        let yaml = format!(
            "persist_path: \"{}\"\nsqlite_filename: \"chroma.sqlite3\"\nallow_reset: true\n",
            temp_dir.path().display()
        );
        let yaml_c = CString::new(yaml).expect("yaml should not contain null bytes");
        let handle = unsafe { chroma_embedded_start_from_string(yaml_c.as_ptr()) };
        assert!(!handle.is_null(), "failed to start embedded fixture");
        Self {
            handle,
            _temp_dir: temp_dir,
        }
    }
}

impl Drop for EmbeddedFixture {
    fn drop(&mut self) {
        unsafe { chroma_embedded_free(self.handle) };
    }
}

fn bench_embedded_heartbeat(c: &mut Criterion) {
    let fixture = EmbeddedFixture::new();
    c.bench_function("ffi_embedded_heartbeat", |b| {
        b.iter(|| {
            let mut heartbeat = 0u64;
            let rc = unsafe { chroma_embedded_heartbeat(fixture.handle, &mut heartbeat) };
            assert_eq!(rc, SUCCESS);
            black_box(heartbeat);
        });
    });
}

fn bench_embedded_max_batch_size(c: &mut Criterion) {
    let fixture = EmbeddedFixture::new();
    c.bench_function("ffi_embedded_get_max_batch_size", |b| {
        b.iter(|| {
            let mut max_batch_size = 0u32;
            let rc =
                unsafe { chroma_embedded_get_max_batch_size(fixture.handle, &mut max_batch_size) };
            assert_eq!(rc, SUCCESS);
            black_box(max_batch_size);
        });
    });
}

fn bench_embedded_startup(c: &mut Criterion) {
    c.bench_function("ffi_embedded_startup_and_free", |b| {
        b.iter(|| {
            let fixture = EmbeddedFixture::new();
            black_box(fixture.handle);
        });
    });
}

criterion_group!(
    benches,
    bench_embedded_heartbeat,
    bench_embedded_max_batch_size,
    bench_embedded_startup
);
criterion_main!(benches);
