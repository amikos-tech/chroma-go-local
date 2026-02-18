use std::cell::RefCell;
use std::ffi::{c_char, c_void, CStr, CString};
use std::ptr;
use std::str::FromStr;
use std::sync::{Arc, Mutex};

use chroma_config::registry::Registry;
use chroma_config::Configurable;
use chroma_frontend::config::FrontendServerConfig;
use chroma_frontend::frontend_service_entrypoint_with_config_system_registry;
use chroma_frontend::Frontend;
use chroma_system::System;
use chroma_types::{
    AddCollectionRecordsRequest, CollectionUuid, CountCollectionsRequest, CountRequest,
    CreateCollectionRequest, CreateDatabaseRequest, CreateTenantRequest,
    DeleteCollectionRecordsRequest, DeleteCollectionRequest, DeleteDatabaseRequest,
    ForkCollectionRequest, GetCollectionRequest, GetDatabaseRequest, GetRequest, GetTenantRequest,
    IncludeList, ListCollectionsRequest, ListDatabasesRequest, QueryRequest, RawWhereFields,
    UpdateCollectionRecordsRequest, UpdateCollectionRequest, UpdateTenantRequest,
    UpsertCollectionRecordsRequest, Where,
};
use figment::providers::{Env, Format, Yaml};
use serde::de::DeserializeOwned;
use serde::Deserialize;
use serde_json::Value;
use tokio::runtime::Runtime;
use tokio::sync::oneshot;

// Error codes
pub const SUCCESS: i32 = 0;
pub const ERROR_NULL_INPUT: i32 = -1;
pub const ERROR_INVALID_UTF8: i32 = -2;
pub const ERROR_CONFIG_PARSE: i32 = -3;
pub const ERROR_SERVER_START: i32 = -4;
pub const ERROR_INVALID_HANDLE: i32 = -5;
pub const ERROR_RUNTIME_CREATE: i32 = -6;
pub const ERROR_ALREADY_STOPPED: i32 = -7;
pub const ERROR_OPERATION_FAILED: i32 = -8;

const DEFAULT_TENANT: &str = "default_tenant";
const DEFAULT_DATABASE: &str = "default_database";
const DEFAULT_QUERY_RESULTS: u32 = 10;

thread_local! {
    static LAST_ERROR: RefCell<Option<CString>> = const { RefCell::new(None) };
}

fn set_last_error(msg: &str) {
    LAST_ERROR.with(|e| {
        *e.borrow_mut() = CString::new(msg).ok();
    });
}

struct ServerHandle {
    _runtime: Runtime, // kept alive to maintain the tokio runtime
    shutdown_tx: Option<oneshot::Sender<()>>,
    port: u16,
    listen_address: CString,
}

struct EmbeddedHandle {
    runtime: Runtime,
    frontend: Mutex<Frontend>,
    _system: System,
    _registry: Registry,
}

#[derive(Debug, Deserialize)]
struct EmbeddedCreateCollectionPayload {
    name: String,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
    #[serde(default)]
    get_or_create: bool,
}

impl EmbeddedCreateCollectionPayload {
    fn into_request(self) -> Result<CreateCollectionRequest, String> {
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = self
            .database_name
            .unwrap_or_else(|| DEFAULT_DATABASE.to_string());

        CreateCollectionRequest::try_new(
            tenant_id,
            database_name,
            self.name,
            None,
            None,
            None,
            self.get_or_create,
        )
        .map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedAddPayload {
    collection_id: String,
    ids: Vec<String>,
    embeddings: Vec<Vec<f32>>,
    documents: Option<Vec<Option<String>>>,
    uris: Option<Vec<Option<String>>>,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
}

impl EmbeddedAddPayload {
    fn into_request(self) -> Result<AddCollectionRecordsRequest, String> {
        let collection_id = CollectionUuid::from_str(&self.collection_id)
            .map_err(|e| format!("invalid collection_id: {e}"))?;
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = self
            .database_name
            .unwrap_or_else(|| DEFAULT_DATABASE.to_string());

        AddCollectionRecordsRequest::try_new(
            tenant_id,
            database_name,
            collection_id,
            self.ids,
            self.embeddings,
            self.documents,
            self.uris,
            None,
        )
        .map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedQueryPayload {
    collection_id: String,
    query_embeddings: Vec<Vec<f32>>,
    #[serde(default)]
    n_results: Option<u32>,
    #[serde(default)]
    ids: Option<Vec<String>>,
    #[serde(default)]
    r#where: Option<Value>,
    #[serde(default)]
    where_document: Option<Value>,
    #[serde(default)]
    include: Option<Vec<String>>,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
}

impl EmbeddedQueryPayload {
    fn into_request(self) -> Result<QueryRequest, String> {
        let collection_id = CollectionUuid::from_str(&self.collection_id)
            .map_err(|e| format!("invalid collection_id: {e}"))?;
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = self
            .database_name
            .unwrap_or_else(|| DEFAULT_DATABASE.to_string());
        let r#where = parse_where_fields(self.r#where, self.where_document)?;
        let include = match self.include {
            Some(include_values) => {
                IncludeList::try_from(include_values).map_err(|e| e.to_string())?
            }
            None => IncludeList::default_query(),
        };

        QueryRequest::try_new(
            tenant_id,
            database_name,
            collection_id,
            self.ids,
            r#where,
            self.query_embeddings,
            self.n_results.unwrap_or(DEFAULT_QUERY_RESULTS),
            include,
        )
        .map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedCreateTenantPayload {
    name: String,
}

impl EmbeddedCreateTenantPayload {
    fn into_request(self) -> Result<CreateTenantRequest, String> {
        CreateTenantRequest::try_new(self.name).map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedGetTenantPayload {
    name: String,
}

impl EmbeddedGetTenantPayload {
    fn into_request(self) -> Result<GetTenantRequest, String> {
        GetTenantRequest::try_new(self.name).map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedUpdateTenantPayload {
    tenant_id: String,
    resource_name: String,
}

impl EmbeddedUpdateTenantPayload {
    fn into_request(self) -> Result<UpdateTenantRequest, String> {
        UpdateTenantRequest::try_new(self.tenant_id, self.resource_name).map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedIndexingStatusPayload {
    collection_id: String,
}

impl EmbeddedIndexingStatusPayload {
    fn into_request(self) -> Result<CollectionUuid, String> {
        CollectionUuid::from_str(&self.collection_id)
            .map_err(|e| format!("invalid collection_id: {e}"))
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedCreateDatabasePayload {
    name: String,
    #[serde(default)]
    tenant_id: Option<String>,
}

impl EmbeddedCreateDatabasePayload {
    fn into_request(self) -> Result<CreateDatabaseRequest, String> {
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        CreateDatabaseRequest::try_new(tenant_id, self.name).map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedListDatabasesPayload {
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    limit: Option<u32>,
    #[serde(default)]
    offset: Option<u32>,
}

impl EmbeddedListDatabasesPayload {
    fn into_request(self) -> Result<ListDatabasesRequest, String> {
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        ListDatabasesRequest::try_new(tenant_id, self.limit, self.offset.unwrap_or(0))
            .map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedGetDatabasePayload {
    name: String,
    #[serde(default)]
    tenant_id: Option<String>,
}

impl EmbeddedGetDatabasePayload {
    fn into_request(self) -> Result<GetDatabaseRequest, String> {
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        GetDatabaseRequest::try_new(tenant_id, self.name).map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedDeleteDatabasePayload {
    name: String,
    #[serde(default)]
    tenant_id: Option<String>,
}

impl EmbeddedDeleteDatabasePayload {
    fn into_request(self) -> Result<DeleteDatabaseRequest, String> {
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        DeleteDatabaseRequest::try_new(tenant_id, self.name).map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedListCollectionsPayload {
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
    #[serde(default)]
    limit: Option<u32>,
    #[serde(default)]
    offset: Option<u32>,
}

impl EmbeddedListCollectionsPayload {
    fn into_request(self) -> Result<ListCollectionsRequest, String> {
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = self
            .database_name
            .unwrap_or_else(|| DEFAULT_DATABASE.to_string());
        ListCollectionsRequest::try_new(
            tenant_id,
            database_name,
            self.limit,
            self.offset.unwrap_or(0),
        )
        .map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedGetCollectionPayload {
    name: String,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
}

impl EmbeddedGetCollectionPayload {
    fn into_request(self) -> Result<GetCollectionRequest, String> {
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = self
            .database_name
            .unwrap_or_else(|| DEFAULT_DATABASE.to_string());
        GetCollectionRequest::try_new(tenant_id, database_name, self.name)
            .map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedCountCollectionsPayload {
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
}

impl EmbeddedCountCollectionsPayload {
    fn into_request(self) -> Result<CountCollectionsRequest, String> {
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = self
            .database_name
            .unwrap_or_else(|| DEFAULT_DATABASE.to_string());
        CountCollectionsRequest::try_new(tenant_id, database_name).map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedUpdateCollectionPayload {
    collection_id: String,
    new_name: String,
}

impl EmbeddedUpdateCollectionPayload {
    fn into_request(self) -> Result<UpdateCollectionRequest, String> {
        let collection_id = CollectionUuid::from_str(&self.collection_id)
            .map_err(|e| format!("invalid collection_id: {e}"))?;
        UpdateCollectionRequest::try_new(collection_id, Some(self.new_name), None, None)
            .map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedDeleteCollectionPayload {
    name: String,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
}

impl EmbeddedDeleteCollectionPayload {
    fn into_request(self) -> Result<DeleteCollectionRequest, String> {
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = self
            .database_name
            .unwrap_or_else(|| DEFAULT_DATABASE.to_string());
        DeleteCollectionRequest::try_new(tenant_id, database_name, self.name)
            .map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedForkCollectionPayload {
    source_collection_id: String,
    target_collection_name: String,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
}

impl EmbeddedForkCollectionPayload {
    fn into_request(self) -> Result<ForkCollectionRequest, String> {
        let source_collection_id = CollectionUuid::from_str(&self.source_collection_id)
            .map_err(|e| format!("invalid source_collection_id: {e}"))?;
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = self
            .database_name
            .unwrap_or_else(|| DEFAULT_DATABASE.to_string());
        ForkCollectionRequest::try_new(
            tenant_id,
            database_name,
            source_collection_id,
            self.target_collection_name,
        )
        .map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedCountPayload {
    collection_id: String,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
}

impl EmbeddedCountPayload {
    fn into_request(self) -> Result<CountRequest, String> {
        let collection_id = CollectionUuid::from_str(&self.collection_id)
            .map_err(|e| format!("invalid collection_id: {e}"))?;
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = self
            .database_name
            .unwrap_or_else(|| DEFAULT_DATABASE.to_string());
        CountRequest::try_new(tenant_id, database_name, collection_id).map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedGetPayload {
    collection_id: String,
    #[serde(default)]
    ids: Option<Vec<String>>,
    #[serde(default)]
    r#where: Option<Value>,
    #[serde(default)]
    where_document: Option<Value>,
    #[serde(default)]
    limit: Option<u32>,
    #[serde(default)]
    offset: Option<u32>,
    #[serde(default)]
    include: Option<Vec<String>>,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
}

impl EmbeddedGetPayload {
    fn into_request(self) -> Result<GetRequest, String> {
        let collection_id = CollectionUuid::from_str(&self.collection_id)
            .map_err(|e| format!("invalid collection_id: {e}"))?;
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = self
            .database_name
            .unwrap_or_else(|| DEFAULT_DATABASE.to_string());
        let r#where = parse_where_fields(self.r#where, self.where_document)?;
        let include = match self.include {
            Some(include_values) => {
                IncludeList::try_from(include_values).map_err(|e| e.to_string())?
            }
            None => IncludeList::default_get(),
        };
        GetRequest::try_new(
            tenant_id,
            database_name,
            collection_id,
            self.ids,
            r#where,
            self.limit,
            self.offset.unwrap_or(0),
            include,
        )
        .map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedUpdatePayload {
    collection_id: String,
    ids: Vec<String>,
    #[serde(default)]
    embeddings: Option<Vec<Vec<f32>>>,
    #[serde(default)]
    documents: Option<Vec<Option<String>>>,
    #[serde(default)]
    uris: Option<Vec<Option<String>>>,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
}

impl EmbeddedUpdatePayload {
    fn into_request(self) -> Result<UpdateCollectionRecordsRequest, String> {
        let collection_id = CollectionUuid::from_str(&self.collection_id)
            .map_err(|e| format!("invalid collection_id: {e}"))?;
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = self
            .database_name
            .unwrap_or_else(|| DEFAULT_DATABASE.to_string());
        let embeddings = self
            .embeddings
            .map(|rows| rows.into_iter().map(Some).collect::<Vec<_>>());

        UpdateCollectionRecordsRequest::try_new(
            tenant_id,
            database_name,
            collection_id,
            self.ids,
            embeddings,
            self.documents,
            self.uris,
            None,
        )
        .map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedUpsertPayload {
    collection_id: String,
    ids: Vec<String>,
    embeddings: Vec<Vec<f32>>,
    #[serde(default)]
    documents: Option<Vec<Option<String>>>,
    #[serde(default)]
    uris: Option<Vec<Option<String>>>,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
}

impl EmbeddedUpsertPayload {
    fn into_request(self) -> Result<UpsertCollectionRecordsRequest, String> {
        let collection_id = CollectionUuid::from_str(&self.collection_id)
            .map_err(|e| format!("invalid collection_id: {e}"))?;
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = self
            .database_name
            .unwrap_or_else(|| DEFAULT_DATABASE.to_string());

        UpsertCollectionRecordsRequest::try_new(
            tenant_id,
            database_name,
            collection_id,
            self.ids,
            self.embeddings,
            self.documents,
            self.uris,
            None,
        )
        .map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedDeleteRecordsPayload {
    collection_id: String,
    #[serde(default)]
    ids: Option<Vec<String>>,
    #[serde(default)]
    r#where: Option<Value>,
    #[serde(default)]
    where_document: Option<Value>,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
}

impl EmbeddedDeleteRecordsPayload {
    fn into_request(self) -> Result<DeleteCollectionRecordsRequest, String> {
        let collection_id = CollectionUuid::from_str(&self.collection_id)
            .map_err(|e| format!("invalid collection_id: {e}"))?;
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = self
            .database_name
            .unwrap_or_else(|| DEFAULT_DATABASE.to_string());
        let r#where = parse_where_fields(self.r#where, self.where_document)?;

        DeleteCollectionRecordsRequest::try_new(
            tenant_id,
            database_name,
            collection_id,
            self.ids,
            r#where,
        )
        .map_err(|e| e.to_string())
    }
}

fn parse_where_fields(
    r#where: Option<Value>,
    where_document: Option<Value>,
) -> Result<Option<Where>, String> {
    RawWhereFields::new(
        r#where.unwrap_or(Value::Null),
        where_document.unwrap_or(Value::Null),
    )
    .parse()
    .map_err(|e| e.to_string())
}

unsafe fn c_ptr_to_string(value: *const c_char, arg_name: &str) -> Result<String, String> {
    if value.is_null() {
        return Err(format!("{arg_name} is null"));
    }

    CStr::from_ptr(value)
        .to_str()
        .map(|s| s.to_string())
        .map_err(|_| format!("{arg_name} is not valid UTF-8"))
}

unsafe fn parse_json_request<T: DeserializeOwned>(
    request_json: *const c_char,
    arg_name: &str,
) -> Result<T, String> {
    let request_str = c_ptr_to_string(request_json, arg_name)?;
    serde_json::from_str::<T>(&request_str).map_err(|e| format!("invalid request JSON: {e}"))
}

fn json_to_c_string_ptr<T: serde::Serialize>(value: &T) -> *mut c_char {
    let json = match serde_json::to_string(value) {
        Ok(json) => json,
        Err(e) => {
            set_last_error(&format!("failed to serialize response: {e}"));
            return ptr::null_mut();
        }
    };

    match CString::new(json) {
        Ok(cstr) => cstr.into_raw(),
        Err(_) => {
            set_last_error("serialized response contains null byte");
            ptr::null_mut()
        }
    }
}

fn parse_config_from_path(path: &str) -> Result<FrontendServerConfig, String> {
    if !std::path::Path::new(path).exists() {
        return Err(format!("Config file not found: {path}"));
    }
    let f = figment::Figment::from(Yaml::file(path))
        .merge(Env::prefixed("CHROMA_").map(|k| k.as_str().replace("__", ".").into()));
    f.extract().map_err(|e| format!("Config parse error: {e}"))
}

fn parse_config_from_string(yaml_str: &str) -> Result<FrontendServerConfig, String> {
    let f = figment::Figment::from(Yaml::string(yaml_str))
        .merge(Env::prefixed("CHROMA_").map(|k| k.as_str().replace("__", ".").into()));
    f.extract().map_err(|e| format!("Config parse error: {e}"))
}

fn resolve_storage_paths(config: &FrontendServerConfig) -> Result<FrontendServerConfig, String> {
    let mut resolved = config.clone();

    if let (Some(sql_cfg), Some(local_segman_cfg)) = (
        resolved.frontend.sqlitedb.as_mut(),
        resolved.frontend.segment_manager.as_mut(),
    ) {
        let persist_root = std::path::Path::new(&resolved.persist_path);
        let sqlite_url = persist_root.join(&resolved.sqlite_filename);

        let persist_path = persist_root
            .to_str()
            .ok_or_else(|| "persist_path must be valid UTF-8".to_string())?
            .to_string();
        let sqlite_path = sqlite_url
            .to_str()
            .ok_or_else(|| "sqlite_filename must produce a valid UTF-8 path".to_string())?
            .to_string();

        local_segman_cfg.persist_path.get_or_insert(persist_path);
        sql_cfg.url.get_or_insert(sqlite_path);
    }

    Ok(resolved)
}

/// Start a Chroma server from a config file path.
/// Returns a handle on success, NULL on failure.
/// Use `chroma_get_last_error` to get error details.
///
/// # Safety
/// `config_path` must be a valid null-terminated C string or NULL.
#[no_mangle]
pub unsafe extern "C" fn chroma_server_start(config_path: *const c_char) -> *mut c_void {
    let path = match c_ptr_to_string(config_path, "config_path") {
        Ok(path) => path,
        Err(msg) => {
            set_last_error(&msg);
            return ptr::null_mut();
        }
    };

    let config = match parse_config_from_path(&path) {
        Ok(c) => c,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    start_server_with_config(config)
}

/// Start a Chroma server from a YAML config string.
/// Returns a handle on success, NULL on failure.
/// Use `chroma_get_last_error` to get error details.
///
/// # Safety
/// `config_yaml` must be a valid null-terminated C string or NULL.
#[no_mangle]
pub unsafe extern "C" fn chroma_server_start_from_string(
    config_yaml: *const c_char,
) -> *mut c_void {
    let yaml = match c_ptr_to_string(config_yaml, "config_yaml") {
        Ok(yaml) => yaml,
        Err(msg) => {
            set_last_error(&msg);
            return ptr::null_mut();
        }
    };

    let config = match parse_config_from_string(&yaml) {
        Ok(c) => c,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    start_server_with_config(config)
}

fn start_server_with_config(config: FrontendServerConfig) -> *mut c_void {
    let runtime = match Runtime::new() {
        Ok(r) => r,
        Err(e) => {
            set_last_error(&format!("Failed to create tokio runtime: {e}"));
            return ptr::null_mut();
        }
    };

    let resolved_config = match resolve_storage_paths(&config) {
        Ok(config) => config,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let (shutdown_tx, shutdown_rx) = oneshot::channel::<()>();
    let port = resolved_config.port;
    let listen_address = match CString::new(resolved_config.listen_address.clone()) {
        Ok(s) => s,
        Err(_) => {
            set_last_error("listen_address contains null byte");
            return ptr::null_mut();
        }
    };

    // Use unit type () which implements both AuthenticateAndAuthorize and QuotaEnforcer.
    let auth: Arc<dyn chroma_frontend::auth::AuthenticateAndAuthorize> = Arc::new(());
    let quota_enforcer: Arc<dyn chroma_frontend::quota::QuotaEnforcer> = Arc::new(());

    runtime.spawn(async move {
        let system = System::new();
        let registry = Registry::new();

        tokio::select! {
            biased;
            _ = shutdown_rx => {
                // Shutdown requested.
            }
            _ = frontend_service_entrypoint_with_config_system_registry(
                auth,
                quota_enforcer,
                system,
                registry,
                &resolved_config,
            ) => {
                // Server exited.
            }
        }
    });

    let handle = Box::new(ServerHandle {
        _runtime: runtime,
        shutdown_tx: Some(shutdown_tx),
        port,
        listen_address,
    });

    Box::into_raw(handle) as *mut c_void
}

/// Get the port the server is configured to listen on.
/// Returns the port from config, or -1 on invalid handle.
///
/// # Safety
/// `handle` must be a valid handle from `chroma_server_start*` or NULL.
#[no_mangle]
pub unsafe extern "C" fn chroma_server_port(handle: *mut c_void) -> i32 {
    if handle.is_null() {
        return -1;
    }
    let server = &*(handle as *const ServerHandle);
    server.port as i32
}

/// Get the listen address the server is configured with.
/// Returns NULL on invalid handle.
/// The returned string is valid until the handle is freed.
///
/// # Safety
/// `handle` must be a valid handle from `chroma_server_start*` or NULL.
#[no_mangle]
pub unsafe extern "C" fn chroma_server_address(handle: *mut c_void) -> *const c_char {
    if handle.is_null() {
        return ptr::null();
    }
    let server = &*(handle as *const ServerHandle);
    server.listen_address.as_ptr()
}

/// Stop the server gracefully.
/// Returns SUCCESS on success, error code on failure.
///
/// # Safety
/// `handle` must be a valid handle from `chroma_server_start*` or NULL.
#[no_mangle]
pub unsafe extern "C" fn chroma_server_stop(handle: *mut c_void) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }

    let server = &mut *(handle as *mut ServerHandle);

    if let Some(tx) = server.shutdown_tx.take() {
        let _ = tx.send(());
        SUCCESS
    } else {
        set_last_error("server already stopped");
        ERROR_ALREADY_STOPPED
    }
}

/// Free the server handle.
/// Must be called after the server is stopped.
///
/// # Safety
/// `handle` must be a valid handle from `chroma_server_start*` or NULL.
/// Must not be called more than once for the same handle.
#[no_mangle]
pub unsafe extern "C" fn chroma_server_free(handle: *mut c_void) {
    if !handle.is_null() {
        let _ = Box::from_raw(handle as *mut ServerHandle);
    }
}

/// Start embedded (in-process) mode from a config file path.
/// Returns a handle on success, NULL on failure.
///
/// # Safety
/// `config_path` must be a valid null-terminated C string or NULL.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_start(config_path: *const c_char) -> *mut c_void {
    let path = match c_ptr_to_string(config_path, "config_path") {
        Ok(path) => path,
        Err(msg) => {
            set_last_error(&msg);
            return ptr::null_mut();
        }
    };

    let config = match parse_config_from_path(&path) {
        Ok(c) => c,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    start_embedded_with_config(config)
}

/// Start embedded (in-process) mode from a YAML config string.
/// Returns a handle on success, NULL on failure.
///
/// # Safety
/// `config_yaml` must be a valid null-terminated C string or NULL.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_start_from_string(
    config_yaml: *const c_char,
) -> *mut c_void {
    let yaml = match c_ptr_to_string(config_yaml, "config_yaml") {
        Ok(yaml) => yaml,
        Err(msg) => {
            set_last_error(&msg);
            return ptr::null_mut();
        }
    };

    let config = match parse_config_from_string(&yaml) {
        Ok(c) => c,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    start_embedded_with_config(config)
}

fn start_embedded_with_config(config: FrontendServerConfig) -> *mut c_void {
    let runtime = match Runtime::new() {
        Ok(r) => r,
        Err(e) => {
            set_last_error(&format!("failed to create tokio runtime: {e}"));
            return ptr::null_mut();
        }
    };

    let resolved_config = match resolve_storage_paths(&config) {
        Ok(config) => config,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let system = System::new();
    let registry = Registry::new();

    let frontend = match runtime.block_on(async {
        Frontend::try_from_config(
            &(resolved_config.frontend.clone(), system.clone()),
            &registry,
        )
        .await
    }) {
        Ok(frontend) => frontend,
        Err(e) => {
            set_last_error(&format!("failed to initialize embedded frontend: {e}"));
            return ptr::null_mut();
        }
    };

    let handle = Box::new(EmbeddedHandle {
        runtime,
        frontend: Mutex::new(frontend),
        _system: system,
        _registry: registry,
    });

    Box::into_raw(handle) as *mut c_void
}

/// Free the embedded handle and all associated resources.
///
/// # Safety
/// `handle` must be a valid handle from `chroma_embedded_start*` or NULL.
/// Must not be called more than once for the same handle.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_free(handle: *mut c_void) {
    if !handle.is_null() {
        let _ = Box::from_raw(handle as *mut EmbeddedHandle);
    }
}

/// Get an in-process heartbeat value as unix nanoseconds.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `out_heartbeat` must point to writable memory.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_heartbeat(
    handle: *mut c_void,
    out_heartbeat: *mut u64,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }
    if out_heartbeat.is_null() {
        set_last_error("out_heartbeat is null");
        return ERROR_NULL_INPUT;
    }

    let embedded = &*(handle as *const EmbeddedHandle);
    let frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    let heartbeat = match embedded
        .runtime
        .block_on(async { frontend.heartbeat().await })
    {
        Ok(heartbeat) => heartbeat,
        Err(e) => {
            set_last_error(&format!("heartbeat failed: {e}"));
            return ERROR_OPERATION_FAILED;
        }
    };

    *out_heartbeat = heartbeat.nanosecond_heartbeat as u64;
    SUCCESS
}

/// Get max batch size from the embedded frontend.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `out_max_batch_size` must point to writable memory.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_get_max_batch_size(
    handle: *mut c_void,
    out_max_batch_size: *mut u32,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }
    if out_max_batch_size.is_null() {
        set_last_error("out_max_batch_size is null");
        return ERROR_NULL_INPUT;
    }

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    *out_max_batch_size = frontend.get_max_batch_size();
    SUCCESS
}

/// Create a tenant in embedded mode.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_create_tenant(
    handle: *mut c_void,
    request_json: *const c_char,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }

    let payload: EmbeddedCreateTenantPayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ERROR_CONFIG_PARSE;
            }
        };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    match embedded
        .runtime
        .block_on(async { frontend.create_tenant(request).await })
    {
        Ok(_) => SUCCESS,
        Err(e) => {
            set_last_error(&format!("create tenant failed: {e}"));
            ERROR_OPERATION_FAILED
        }
    }
}

/// Get a tenant in embedded mode.
/// Returns a JSON-serialized tenant object on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_get_tenant(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    if handle.is_null() {
        set_last_error("handle is null");
        return ptr::null_mut();
    }

    let payload: EmbeddedGetTenantPayload = match parse_json_request(request_json, "request_json") {
        Ok(payload) => payload,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ptr::null_mut();
        }
    };

    let tenant = match embedded
        .runtime
        .block_on(async { frontend.get_tenant(request).await })
    {
        Ok(tenant) => tenant,
        Err(e) => {
            set_last_error(&format!("get tenant failed: {e}"));
            return ptr::null_mut();
        }
    };

    json_to_c_string_ptr(&tenant)
}

/// Update a tenant in embedded mode.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_update_tenant(
    handle: *mut c_void,
    request_json: *const c_char,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }

    let payload: EmbeddedUpdateTenantPayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ERROR_CONFIG_PARSE;
            }
        };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    match embedded
        .runtime
        .block_on(async { frontend.update_tenant(request).await })
    {
        Ok(_) => SUCCESS,
        Err(e) => {
            set_last_error(&format!("update tenant failed: {e}"));
            ERROR_OPERATION_FAILED
        }
    }
}

/// Create a database in embedded mode.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_create_database(
    handle: *mut c_void,
    request_json: *const c_char,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }

    let payload: EmbeddedCreateDatabasePayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ERROR_CONFIG_PARSE;
            }
        };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    match embedded
        .runtime
        .block_on(async { frontend.create_database(request).await })
    {
        Ok(_) => SUCCESS,
        Err(e) => {
            set_last_error(&format!("create database failed: {e}"));
            ERROR_OPERATION_FAILED
        }
    }
}

/// List databases in embedded mode.
/// Returns a JSON-serialized list of databases on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_list_databases(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    if handle.is_null() {
        set_last_error("handle is null");
        return ptr::null_mut();
    }

    let payload: EmbeddedListDatabasesPayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ptr::null_mut();
            }
        };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ptr::null_mut();
        }
    };

    let databases = match embedded
        .runtime
        .block_on(async { frontend.list_databases(request).await })
    {
        Ok(databases) => databases,
        Err(e) => {
            set_last_error(&format!("list databases failed: {e}"));
            return ptr::null_mut();
        }
    };

    json_to_c_string_ptr(&databases)
}

/// Get a database in embedded mode.
/// Returns a JSON-serialized database object on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_get_database(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    if handle.is_null() {
        set_last_error("handle is null");
        return ptr::null_mut();
    }

    let payload: EmbeddedGetDatabasePayload = match parse_json_request(request_json, "request_json")
    {
        Ok(payload) => payload,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ptr::null_mut();
        }
    };

    let database = match embedded
        .runtime
        .block_on(async { frontend.get_database(request).await })
    {
        Ok(database) => database,
        Err(e) => {
            set_last_error(&format!("get database failed: {e}"));
            return ptr::null_mut();
        }
    };

    json_to_c_string_ptr(&database)
}

/// Delete a database in embedded mode.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_delete_database(
    handle: *mut c_void,
    request_json: *const c_char,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }

    let payload: EmbeddedDeleteDatabasePayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ERROR_CONFIG_PARSE;
            }
        };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    match embedded
        .runtime
        .block_on(async { frontend.delete_database(request).await })
    {
        Ok(_) => SUCCESS,
        Err(e) => {
            set_last_error(&format!("delete database failed: {e}"));
            ERROR_OPERATION_FAILED
        }
    }
}

/// List collections in embedded mode.
/// Returns a JSON-serialized list of collections on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_list_collections(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    if handle.is_null() {
        set_last_error("handle is null");
        return ptr::null_mut();
    }

    let payload: EmbeddedListCollectionsPayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ptr::null_mut();
            }
        };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ptr::null_mut();
        }
    };

    let collections = match embedded
        .runtime
        .block_on(async { frontend.list_collections(request).await })
    {
        Ok(collections) => collections,
        Err(e) => {
            set_last_error(&format!("list collections failed: {e}"));
            return ptr::null_mut();
        }
    };

    json_to_c_string_ptr(&collections)
}

/// Get a collection in embedded mode.
/// Returns a JSON-serialized collection object on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_get_collection(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    if handle.is_null() {
        set_last_error("handle is null");
        return ptr::null_mut();
    }

    let payload: EmbeddedGetCollectionPayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ptr::null_mut();
            }
        };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ptr::null_mut();
        }
    };

    let collection = match embedded
        .runtime
        .block_on(async { frontend.get_collection(request).await })
    {
        Ok(collection) => collection,
        Err(e) => {
            set_last_error(&format!("get collection failed: {e}"));
            return ptr::null_mut();
        }
    };

    json_to_c_string_ptr(&collection)
}

/// Count collections in embedded mode.
/// Returns SUCCESS on success and writes the count to `out_count`.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
/// `out_count` must point to writable memory.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_count_collections(
    handle: *mut c_void,
    request_json: *const c_char,
    out_count: *mut u32,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }
    if out_count.is_null() {
        set_last_error("out_count is null");
        return ERROR_NULL_INPUT;
    }

    let payload: EmbeddedCountCollectionsPayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ERROR_CONFIG_PARSE;
            }
        };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    let count = match embedded
        .runtime
        .block_on(async { frontend.count_collections(request).await })
    {
        Ok(count) => count,
        Err(e) => {
            set_last_error(&format!("count collections failed: {e}"));
            return ERROR_OPERATION_FAILED;
        }
    };

    *out_count = count;
    SUCCESS
}

/// Update a collection in embedded mode.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_update_collection(
    handle: *mut c_void,
    request_json: *const c_char,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }

    let payload: EmbeddedUpdateCollectionPayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ERROR_CONFIG_PARSE;
            }
        };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    match embedded
        .runtime
        .block_on(async { frontend.update_collection(request).await })
    {
        Ok(_) => SUCCESS,
        Err(e) => {
            set_last_error(&format!("update collection failed: {e}"));
            ERROR_OPERATION_FAILED
        }
    }
}

/// Delete a collection in embedded mode.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_delete_collection(
    handle: *mut c_void,
    request_json: *const c_char,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }

    let payload: EmbeddedDeleteCollectionPayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ERROR_CONFIG_PARSE;
            }
        };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    match embedded
        .runtime
        .block_on(async { frontend.delete_collection(request).await })
    {
        Ok(_) => SUCCESS,
        Err(e) => {
            set_last_error(&format!("delete collection failed: {e}"));
            ERROR_OPERATION_FAILED
        }
    }
}

/// Fork a collection in embedded mode.
/// Returns a JSON-serialized collection object on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_fork_collection(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    if handle.is_null() {
        set_last_error("handle is null");
        return ptr::null_mut();
    }

    let payload: EmbeddedForkCollectionPayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ptr::null_mut();
            }
        };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ptr::null_mut();
        }
    };

    let collection = match embedded
        .runtime
        .block_on(async { frontend.fork_collection(request).await })
    {
        Ok(collection) => collection,
        Err(e) => {
            set_last_error(&format!("fork collection failed: {e}"));
            return ptr::null_mut();
        }
    };

    json_to_c_string_ptr(&collection)
}

/// Count records in a collection in embedded mode.
/// Returns SUCCESS on success and writes the count to `out_count`.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
/// `out_count` must point to writable memory.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_count(
    handle: *mut c_void,
    request_json: *const c_char,
    out_count: *mut u32,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }
    if out_count.is_null() {
        set_last_error("out_count is null");
        return ERROR_NULL_INPUT;
    }

    let payload: EmbeddedCountPayload = match parse_json_request(request_json, "request_json") {
        Ok(payload) => payload,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    let count = match embedded
        .runtime
        .block_on(async { frontend.count(request).await })
    {
        Ok(count) => count,
        Err(e) => {
            set_last_error(&format!("count records failed: {e}"));
            return ERROR_OPERATION_FAILED;
        }
    };

    *out_count = count;
    SUCCESS
}

/// Get records from a collection in embedded mode.
/// Returns a JSON-serialized get response on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_get(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    if handle.is_null() {
        set_last_error("handle is null");
        return ptr::null_mut();
    }

    let payload: EmbeddedGetPayload = match parse_json_request(request_json, "request_json") {
        Ok(payload) => payload,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ptr::null_mut();
        }
    };

    let response = match embedded
        .runtime
        .block_on(async { frontend.get(request).await })
    {
        Ok(response) => response,
        Err(e) => {
            set_last_error(&format!("get records failed: {e}"));
            return ptr::null_mut();
        }
    };

    json_to_c_string_ptr(&response)
}

/// Update records in a collection in embedded mode.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_update(
    handle: *mut c_void,
    request_json: *const c_char,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }

    let payload: EmbeddedUpdatePayload = match parse_json_request(request_json, "request_json") {
        Ok(payload) => payload,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    match embedded
        .runtime
        .block_on(async { frontend.update(request).await })
    {
        Ok(_) => SUCCESS,
        Err(e) => {
            set_last_error(&format!("update records failed: {e}"));
            ERROR_OPERATION_FAILED
        }
    }
}

/// Upsert records in a collection in embedded mode.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_upsert(
    handle: *mut c_void,
    request_json: *const c_char,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }

    let payload: EmbeddedUpsertPayload = match parse_json_request(request_json, "request_json") {
        Ok(payload) => payload,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    match embedded
        .runtime
        .block_on(async { frontend.upsert(request).await })
    {
        Ok(_) => SUCCESS,
        Err(e) => {
            set_last_error(&format!("upsert records failed: {e}"));
            ERROR_OPERATION_FAILED
        }
    }
}

/// Delete records in a collection in embedded mode.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_delete_records(
    handle: *mut c_void,
    request_json: *const c_char,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }

    let payload: EmbeddedDeleteRecordsPayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ERROR_CONFIG_PARSE;
            }
        };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    match embedded
        .runtime
        .block_on(async { frontend.delete(request).await })
    {
        Ok(_) => SUCCESS,
        Err(e) => {
            set_last_error(&format!("delete records failed: {e}"));
            ERROR_OPERATION_FAILED
        }
    }
}

/// Create a collection in embedded mode.
/// Returns a JSON-serialized collection object on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_create_collection(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    if handle.is_null() {
        set_last_error("handle is null");
        return ptr::null_mut();
    }

    let payload: EmbeddedCreateCollectionPayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ptr::null_mut();
            }
        };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ptr::null_mut();
        }
    };

    let collection = match embedded
        .runtime
        .block_on(async { frontend.create_collection(request).await })
    {
        Ok(collection) => collection,
        Err(e) => {
            set_last_error(&format!("create collection failed: {e}"));
            return ptr::null_mut();
        }
    };

    json_to_c_string_ptr(&collection)
}

/// Add records to a collection in embedded mode.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_add(
    handle: *mut c_void,
    request_json: *const c_char,
) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }

    let payload: EmbeddedAddPayload = match parse_json_request(request_json, "request_json") {
        Ok(payload) => payload,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ERROR_CONFIG_PARSE;
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    match embedded
        .runtime
        .block_on(async { frontend.add(request).await })
    {
        Ok(_) => SUCCESS,
        Err(e) => {
            set_last_error(&format!("add failed: {e}"));
            ERROR_OPERATION_FAILED
        }
    }
}

/// Query a collection in embedded mode.
/// Returns a JSON-serialized query response on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_query(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    if handle.is_null() {
        set_last_error("handle is null");
        return ptr::null_mut();
    }

    let payload: EmbeddedQueryPayload = match parse_json_request(request_json, "request_json") {
        Ok(payload) => payload,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let request = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ptr::null_mut();
        }
    };

    let response = match embedded
        .runtime
        .block_on(async { frontend.query(request).await })
    {
        Ok(response) => response,
        Err(e) => {
            set_last_error(&format!("query failed: {e}"));
            return ptr::null_mut();
        }
    };

    json_to_c_string_ptr(&response)
}

/// Get indexing status for a collection in embedded mode.
/// Returns a JSON-serialized indexing status response on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_indexing_status(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    if handle.is_null() {
        set_last_error("handle is null");
        return ptr::null_mut();
    }

    let payload: EmbeddedIndexingStatusPayload =
        match parse_json_request(request_json, "request_json") {
            Ok(payload) => payload,
            Err(e) => {
                set_last_error(&e);
                return ptr::null_mut();
            }
        };

    let collection_id = match payload.into_request() {
        Ok(request) => request,
        Err(e) => {
            set_last_error(&e);
            return ptr::null_mut();
        }
    };

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ptr::null_mut();
        }
    };

    let response = match embedded
        .runtime
        .block_on(async { frontend.indexing_status(collection_id).await })
    {
        Ok(response) => response,
        Err(e) => {
            set_last_error(&format!("indexing status failed: {e}"));
            return ptr::null_mut();
        }
    };

    json_to_c_string_ptr(&response)
}

/// Get healthcheck status in embedded mode.
/// Returns a JSON-serialized healthcheck response on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_healthcheck(handle: *mut c_void) -> *mut c_char {
    if handle.is_null() {
        set_last_error("handle is null");
        return ptr::null_mut();
    }

    let embedded = &*(handle as *const EmbeddedHandle);
    let frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ptr::null_mut();
        }
    };

    let response = embedded
        .runtime
        .block_on(async { frontend.healthcheck().await });
    json_to_c_string_ptr(&response)
}

/// Reset local state in embedded mode.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_reset(handle: *mut c_void) -> i32 {
    if handle.is_null() {
        set_last_error("handle is null");
        return ERROR_INVALID_HANDLE;
    }

    let embedded = &*(handle as *const EmbeddedHandle);
    let mut frontend = match embedded.frontend.lock() {
        Ok(frontend) => frontend,
        Err(_) => {
            set_last_error("embedded frontend lock poisoned");
            return ERROR_OPERATION_FAILED;
        }
    };

    match embedded.runtime.block_on(async { frontend.reset().await }) {
        Ok(_) => SUCCESS,
        Err(e) => {
            set_last_error(&format!("reset failed: {e}"));
            ERROR_OPERATION_FAILED
        }
    }
}

/// Free a heap-allocated C string returned by this library.
///
/// # Safety
/// `s` must be a pointer previously returned by this library via `CString::into_raw`.
#[no_mangle]
pub unsafe extern "C" fn chroma_string_free(s: *mut c_char) {
    if !s.is_null() {
        let _ = CString::from_raw(s);
    }
}

/// Get the last error message.
/// Returns NULL if no error. The returned string is valid until the next FFI call.
///
/// # Safety
/// Must be called from the same thread that made the failing call.
#[no_mangle]
pub unsafe extern "C" fn chroma_get_last_error() -> *const c_char {
    LAST_ERROR.with(|e| match e.borrow().as_ref() {
        Some(s) => s.as_ptr(),
        None => ptr::null(),
    })
}

/// Get the version string.
///
/// # Safety
/// Returns a pointer to a static string, always safe to call.
#[no_mangle]
pub unsafe extern "C" fn chroma_version() -> *const c_char {
    static VERSION: &[u8] = b"0.2.0\0";
    VERSION.as_ptr() as *const c_char
}

#[cfg(test)]
mod tests {
    use super::*;
    use proptest::prelude::*;
    use serde_json::json;

    #[test]
    fn test_parse_config_from_string() {
        let yaml = r#"
port: 8000
listen_address: "127.0.0.1"
persist_path: "./test_chroma"
allow_reset: true
"#;
        let config = parse_config_from_string(yaml);
        assert!(config.is_ok());
        let config = config.unwrap();
        assert_eq!(config.port, 8000);
        assert_eq!(config.listen_address, "127.0.0.1");
    }

    #[test]
    fn test_parse_where_fields_none_when_empty() {
        let parsed = parse_where_fields(None, None).expect("parse should succeed");
        assert!(parsed.is_none());
    }

    #[test]
    fn test_delete_records_payload_allows_empty_filters_per_chroma_types() {
        let payload = EmbeddedDeleteRecordsPayload {
            collection_id: "00000000-0000-0000-0000-000000000001".to_string(),
            ids: None,
            r#where: None,
            where_document: None,
            tenant_id: None,
            database_name: None,
        };
        assert!(payload.into_request().is_ok());
    }

    proptest! {
        #[test]
        fn prop_create_tenant_name_validation(name in "[a-z]{0,8}") {
            let result = EmbeddedCreateTenantPayload { name: name.clone() }.into_request();
            if name.len() < 3 {
                prop_assert!(result.is_err());
            } else {
                prop_assert!(result.is_ok());
            }
        }

        #[test]
        fn prop_delete_records_rejects_invalid_where_document(
            n in any::<i64>(),
        ) {
            let payload = EmbeddedDeleteRecordsPayload {
                collection_id: "00000000-0000-0000-0000-000000000001".to_string(),
                ids: None,
                r#where: None,
                where_document: Some(json!({"$contains": n})),
                tenant_id: None,
                database_name: None,
            };

            let result = payload.into_request();
            prop_assert!(result.is_err());
        }

        #[test]
        fn prop_query_default_n_results(
            embeddings in proptest::collection::vec(
                proptest::collection::vec(-1.0f32..1.0f32, 1..8),
                1..4
            ),
        ) {
            let payload = EmbeddedQueryPayload {
                collection_id: "00000000-0000-0000-0000-000000000001".to_string(),
                query_embeddings: embeddings,
                n_results: None,
                ids: None,
                r#where: None,
                where_document: None,
                include: None,
                tenant_id: None,
                database_name: None,
            };

            let request = payload.into_request().expect("query payload should parse");
            prop_assert_eq!(request.n_results, DEFAULT_QUERY_RESULTS);
        }
    }
}
