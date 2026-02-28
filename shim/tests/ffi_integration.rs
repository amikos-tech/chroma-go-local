use std::ffi::{CStr, CString};

use chroma_shim::{
    chroma_embedded_create_tenant, chroma_embedded_free, chroma_embedded_get_tenant,
    chroma_embedded_heartbeat, chroma_embedded_start_from_string, chroma_string_free, SUCCESS,
};
use tempfile::TempDir;

fn last_error() -> String {
    let err_ptr = unsafe { chroma_shim::chroma_get_last_error() };
    if err_ptr.is_null() {
        return String::new();
    }
    unsafe { CStr::from_ptr(err_ptr).to_string_lossy().to_string() }
}

fn start_embedded() -> (*mut std::ffi::c_void, TempDir) {
    let temp_dir = tempfile::tempdir().expect("tempdir should be created");
    let yaml = format!(
        "persist_path: \"{}\"\nsqlite_filename: \"chroma.sqlite3\"\nallow_reset: true\n",
        temp_dir.path().display()
    );
    let yaml_c = CString::new(yaml).expect("yaml should not contain null bytes");
    let handle = unsafe { chroma_embedded_start_from_string(yaml_c.as_ptr()) };
    assert!(
        !handle.is_null(),
        "failed to start embedded: {}",
        last_error()
    );
    (handle, temp_dir)
}

#[test]
fn test_embedded_heartbeat_via_ffi() {
    let (handle, _temp_dir) = start_embedded();
    let mut heartbeat: u64 = 0;

    let rc = unsafe { chroma_embedded_heartbeat(handle, &mut heartbeat) };
    assert_eq!(rc, SUCCESS, "heartbeat failed: {}", last_error());
    assert!(heartbeat > 0, "heartbeat should be non-zero");

    unsafe { chroma_embedded_free(handle) };
}

#[test]
fn test_tenant_create_and_get_via_ffi() {
    let (handle, _temp_dir) = start_embedded();

    let create_req = CString::new(r#"{"name":"tenant_integration"}"#)
        .expect("request should not contain null bytes");
    let rc = unsafe { chroma_embedded_create_tenant(handle, create_req.as_ptr()) };
    assert_eq!(rc, SUCCESS, "create tenant failed: {}", last_error());

    let get_req = CString::new(r#"{"name":"tenant_integration"}"#)
        .expect("request should not contain null bytes");
    let tenant_ptr = unsafe { chroma_embedded_get_tenant(handle, get_req.as_ptr()) };
    assert!(!tenant_ptr.is_null(), "get tenant failed: {}", last_error());

    let tenant_json = unsafe { CStr::from_ptr(tenant_ptr) }
        .to_string_lossy()
        .to_string();
    let tenant: serde_json::Value =
        serde_json::from_str(&tenant_json).expect("tenant json should be valid");
    assert_eq!(tenant["name"], "tenant_integration");

    unsafe { chroma_string_free(tenant_ptr) };
    unsafe { chroma_embedded_free(handle) };
}
