use std::any::Any;
use std::collections::HashMap;
use std::ffi::{c_char, c_void, CStr, CString};
use std::fs;
use std::io::Write;
use std::panic::{self, AssertUnwindSafe};
use std::path::{Path, PathBuf};
use std::ptr;
use std::str::FromStr;
use std::sync::{Arc, Mutex};
use std::time::{Instant, SystemTime, UNIX_EPOCH};

use chroma_config::registry::Registry;
use chroma_config::Configurable;
use chroma_frontend::config::FrontendServerConfig;
use chroma_frontend::frontend_service_entrypoint_with_config_system_registry;
use chroma_frontend::Frontend;
use chroma_index::{HnswIndex, HnswIndexConfig, IndexConfig, IndexUuid};
use chroma_log::sqlite_log::{
    legacy_embeddings_queue_config_default_kind, LegacyEmbeddingsQueueConfig,
};
use chroma_log::{BackfillMessage, LocalCompactionManager, Log, PurgeLogsMessage};
use chroma_segment::local_segment_manager::LocalSegmentManager;
use chroma_sqlite::db::SqliteDb;
use chroma_sysdb::SysDb;
use chroma_system::{ComponentHandle, System};
use chroma_types::{
    AddCollectionRecordsRequest, CollectionConfiguration, CollectionMetadataUpdate, CollectionUuid,
    CountCollectionsRequest, CountRequest, CreateCollectionRequest, CreateDatabaseRequest,
    CreateTenantRequest, DatabaseName, DeleteCollectionRecordsRequest, DeleteCollectionRequest,
    DeleteDatabaseRequest, ForkCollectionRequest, GetCollectionRequest, GetDatabaseRequest,
    GetRequest, GetTenantRequest, IncludeList, ListCollectionsRequest, ListDatabasesRequest,
    Metadata, QueryRequest, RawWhereFields, Schema, UpdateCollectionRecordsRequest,
    UpdateCollectionRequest, UpdateMetadata, UpdateTenantRequest, UpsertCollectionRecordsRequest,
    Where,
};
use figment::providers::{Env, Format, Yaml};
use serde::de::DeserializeOwned;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use serde_pickle::{DeOptions, SerOptions};
use sqlx::Row;
use tokio::runtime::Runtime;
use tokio::sync::oneshot;
use tokio::time::{timeout, Duration};

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
const HNSW_METADATA_FILENAME: &str = "index_metadata.pickle";

fn parse_database_name(database_name: String) -> Result<DatabaseName, String> {
    DatabaseName::new(database_name)
        .ok_or_else(|| "database_name must be at least 3 characters".to_string())
}

fn resolve_database_name(database_name: Option<String>) -> String {
    database_name.unwrap_or_else(|| DEFAULT_DATABASE.to_string())
}

fn resolve_database_name_typed(database_name: Option<String>) -> Result<DatabaseName, String> {
    parse_database_name(resolve_database_name(database_name))
}

static LAST_ERROR: Mutex<Option<String>> = Mutex::new(None);

fn set_last_error(msg: &str) {
    let sanitized = msg.replace('\0', "\\0");
    match LAST_ERROR.lock() {
        Ok(mut slot) => {
            *slot = Some(sanitized);
        }
        Err(poisoned) => {
            // Preserve error reporting even after an earlier panic poisoned the mutex.
            let mut slot = poisoned.into_inner();
            *slot = Some(sanitized);
        }
    }
}

fn last_error_cstring() -> Option<CString> {
    let slot = match LAST_ERROR.lock() {
        Ok(slot) => slot,
        Err(poisoned) => poisoned.into_inner(),
    };
    let msg = slot.as_ref()?;
    CString::new(msg.as_str()).ok()
}

fn last_error_message() -> Option<String> {
    let slot = match LAST_ERROR.lock() {
        Ok(slot) => slot,
        Err(poisoned) => poisoned.into_inner(),
    };
    slot.clone()
}

fn panic_payload_message(payload: Box<dyn Any + Send>) -> String {
    if let Some(message) = payload.downcast_ref::<String>() {
        return message.clone();
    }
    if let Some(message) = payload.downcast_ref::<&str>() {
        return (*message).to_string();
    }
    "unknown panic payload".to_string()
}

fn with_ffi_panic_boundary<T, F>(default: T, f: F) -> T
where
    F: FnOnce() -> T,
{
    match panic::catch_unwind(AssertUnwindSafe(f)) {
        Ok(value) => value,
        Err(payload) => {
            let panic_message = panic_payload_message(payload);
            set_last_error(&format!("panic across FFI boundary: {panic_message}"));
            default
        }
    }
}

macro_rules! ffi_guard_ptr_mut {
    ($body:block) => {
        with_ffi_panic_boundary(ptr::null_mut(), || $body)
    };
}

macro_rules! ffi_guard_ptr_const {
    ($body:block) => {
        with_ffi_panic_boundary(ptr::null(), || $body)
    };
}

macro_rules! ffi_guard_code {
    ($body:block) => {
        with_ffi_panic_boundary(ERROR_OPERATION_FAILED, || $body)
    };
}

macro_rules! ffi_guard_minus_one {
    ($body:block) => {
        with_ffi_panic_boundary(-1, || $body)
    };
}

macro_rules! ffi_guard_unit {
    ($body:block) => {
        with_ffi_panic_boundary((), || $body)
    };
}

struct ServerHandle {
    _runtime: Runtime, // kept alive to maintain the tokio runtime
    shutdown_tx: Option<oneshot::Sender<()>>,
    port: u16,
    listen_address: CString,
    persist_path: CString,
}

struct EmbeddedHandle {
    runtime: Runtime,
    frontend: Mutex<Frontend>,
    _system: System,
    registry: Registry,
    persist_path: CString,
}

#[derive(Debug, Deserialize)]
struct EmbeddedCreateCollectionPayload {
    name: String,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
    #[serde(default)]
    metadata: Option<Metadata>,
    #[serde(default, alias = "configuration_json")]
    configuration: Option<Value>,
    #[serde(default)]
    schema: Option<Value>,
    #[serde(default)]
    get_or_create: bool,
}

impl EmbeddedCreateCollectionPayload {
    fn into_request(self) -> Result<CreateCollectionRequest, String> {
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = resolve_database_name_typed(self.database_name)?;
        let configuration = self
            .configuration
            .map(|raw_configuration| {
                let configuration: CollectionConfiguration =
                    serde_json::from_value(raw_configuration)
                        .map_err(|e| format!("invalid configuration payload: {e}"))?;
                configuration
                    .try_into()
                    .map_err(|e| format!("invalid configuration value: {e}"))
            })
            .transpose()?;
        let schema = self
            .schema
            .map(|raw_schema| -> Result<Schema, String> {
                let schema: Schema = serde_json::from_value(raw_schema)
                    .map_err(|e| format!("invalid schema payload: {e}"))?;
                Ok(schema)
            })
            .transpose()?;

        CreateCollectionRequest::try_new(
            tenant_id,
            database_name,
            self.name,
            self.metadata,
            configuration,
            schema,
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
    metadatas: Option<Vec<Option<Metadata>>>,
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
        let database_name = resolve_database_name(self.database_name);

        AddCollectionRecordsRequest::try_new(
            tenant_id,
            database_name,
            collection_id,
            self.ids,
            self.embeddings,
            self.documents,
            self.uris,
            self.metadatas,
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
        let database_name = resolve_database_name(self.database_name);
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
    #[serde(default)]
    database_name: Option<String>,
}

impl EmbeddedIndexingStatusPayload {
    fn into_request(self) -> Result<(DatabaseName, CollectionUuid), String> {
        let database_name = resolve_database_name_typed(self.database_name)?;
        let collection_id = CollectionUuid::from_str(&self.collection_id)
            .map_err(|e| format!("invalid collection_id: {e}"))?;
        Ok((database_name, collection_id))
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedCompactCollectionPayload {
    name: String,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
}

impl EmbeddedCompactCollectionPayload {
    fn into_request(self) -> Result<GetCollectionRequest, String> {
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = resolve_database_name_typed(self.database_name)?;
        GetCollectionRequest::try_new(tenant_id, database_name, self.name)
            .map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedCompactAllPayload {
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
}

#[derive(Debug)]
struct CompactionTarget {
    collection_id: CollectionUuid,
    name: String,
    tenant_id: String,
    database_name: DatabaseName,
    database_name_raw: String,
}

#[derive(Debug, Serialize)]
struct EmbeddedCompactionCollectionResult {
    collection_id: String,
    name: String,
    tenant_id: String,
    database_name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pending_ops_before: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pending_ops_after: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pending_ops_before_error: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pending_ops_after_error: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}

#[derive(Debug, Serialize)]
struct EmbeddedCompactionResponse {
    collection_count: u32,
    duration_ms: u64,
    pending_ops_before_total: u64,
    pending_ops_after_total: u64,
    collections: Vec<EmbeddedCompactionCollectionResult>,
}

#[derive(Debug, Clone, Copy)]
enum CompactionFailureMode {
    FailFast,
    ContinueOnError,
}

#[derive(Debug, Clone, Deserialize, Default)]
struct WalPrunePolicyPayload {
    #[serde(default)]
    max_age_seconds: Option<u64>,
    #[serde(default)]
    max_bytes: Option<u64>,
    #[serde(default)]
    watermark_high_bytes: Option<u64>,
    #[serde(default)]
    watermark_low_bytes: Option<u64>,
}

impl WalPrunePolicyPayload {
    fn has_policy(&self) -> bool {
        self.max_age_seconds.is_some()
            || self.max_bytes.is_some()
            || (self.watermark_high_bytes.is_some() && self.watermark_low_bytes.is_some())
    }

    fn validate(&self) -> Result<(), String> {
        if let Some(max_age_seconds) = self.max_age_seconds {
            if max_age_seconds == 0 {
                return Err("max_age_seconds must be greater than 0".to_string());
            }
        }
        if self.watermark_high_bytes.is_some() != self.watermark_low_bytes.is_some() {
            return Err("wal prune watermark requires both high and low bytes".to_string());
        }
        if let (Some(high), Some(low)) = (self.watermark_high_bytes, self.watermark_low_bytes) {
            if low > high {
                return Err(
                    "wal prune watermark low bytes must be less than or equal to high bytes"
                        .to_string(),
                );
            }
        }
        Ok(())
    }
}

#[derive(Debug, Clone)]
struct WalPruneExecutionOptions {
    dry_run: bool,
    vacuum: bool,
    policies: WalPrunePolicyPayload,
}

impl WalPruneExecutionOptions {
    fn validate(&self) -> Result<(), String> {
        self.policies.validate()?;
        if !self.dry_run && !self.policies.has_policy() {
            return Err(
                "at least one WAL prune policy is required unless dry-run is enabled".to_string(),
            );
        }
        Ok(())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedPruneWalCollectionPayload {
    name: String,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
    #[serde(default)]
    dry_run: Option<bool>,
    #[serde(default)]
    vacuum: Option<bool>,
    #[serde(flatten)]
    policies: WalPrunePolicyPayload,
}

struct WalPruneCollectionRequestParts {
    request: GetCollectionRequest,
    options: WalPruneExecutionOptions,
}

impl EmbeddedPruneWalCollectionPayload {
    fn into_request(self) -> Result<WalPruneCollectionRequestParts, String> {
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = resolve_database_name_typed(self.database_name)?;
        let request = GetCollectionRequest::try_new(tenant_id, database_name, self.name)
            .map_err(|e| e.to_string())?;
        let options = WalPruneExecutionOptions {
            dry_run: self.dry_run.unwrap_or(false),
            vacuum: self.vacuum.unwrap_or(false),
            policies: self.policies,
        };
        options.validate()?;
        Ok(WalPruneCollectionRequestParts { request, options })
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedPruneWalAllPayload {
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
    #[serde(default)]
    dry_run: Option<bool>,
    #[serde(default)]
    vacuum: Option<bool>,
    #[serde(flatten)]
    policies: WalPrunePolicyPayload,
}

impl EmbeddedPruneWalAllPayload {
    fn into_options(self) -> Result<WalPruneExecutionOptions, String> {
        if let Some(database_name) = self.database_name.as_ref() {
            parse_database_name(database_name.clone())?;
        }
        if let Some(tenant_id) = self.tenant_id.as_ref() {
            if tenant_id.trim().len() < 3 {
                return Err("tenant_id must be at least 3 characters".to_string());
            }
        }
        let options = WalPruneExecutionOptions {
            dry_run: self.dry_run.unwrap_or(false),
            vacuum: self.vacuum.unwrap_or(false),
            policies: self.policies,
        };
        options.validate()?;
        Ok(options)
    }
}

#[derive(Debug)]
struct WalPruneTarget {
    collection_id: CollectionUuid,
    name: String,
    tenant_id: String,
    database_name_raw: String,
}

#[derive(Debug)]
struct WalPruneCandidateRow {
    seq_id: i64,
    created_at_secs: i64,
    estimated_bytes: u64,
}

#[derive(Debug, Serialize)]
struct EmbeddedWalPruneCollectionResult {
    collection_id: String,
    name: String,
    tenant_id: String,
    database_name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    safe_seq_cutoff: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    candidate_seq_min: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    candidate_seq_max: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pruned_seq_min: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pruned_seq_max: Option<u64>,
    candidate_count: u64,
    candidate_bytes: u64,
    pruned_count: u64,
    pruned_bytes: u64,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}

#[derive(Debug, Serialize)]
struct EmbeddedWalPruneResponse {
    collection_count: u32,
    duration_ms: u64,
    dry_run: bool,
    vacuum_requested: bool,
    vacuum_executed: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    warning: Option<String>,
    candidate_count_total: u64,
    candidate_bytes_total: u64,
    pruned_count_total: u64,
    pruned_bytes_total: u64,
    collections: Vec<EmbeddedWalPruneCollectionResult>,
}

#[derive(Debug, Clone, Copy)]
enum WalPruneFailureMode {
    FailFast,
    ContinueOnError,
}

#[derive(Debug, Deserialize)]
struct EmbeddedRebuildCollectionPayload {
    name: String,
    #[serde(default)]
    tenant_id: Option<String>,
    #[serde(default)]
    database_name: Option<String>,
    #[serde(default)]
    precheck: Option<bool>,
    #[serde(default)]
    keep_backup: Option<bool>,
}

struct RebuildCollectionRequestParts {
    request: GetCollectionRequest,
    precheck: bool,
    keep_backup: bool,
}

impl EmbeddedRebuildCollectionPayload {
    fn into_request(self) -> Result<RebuildCollectionRequestParts, String> {
        let tenant_id = self.tenant_id.unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = resolve_database_name_typed(self.database_name)?;
        let request = GetCollectionRequest::try_new(tenant_id, database_name, self.name)
            .map_err(|e| e.to_string())?;
        Ok(RebuildCollectionRequestParts {
            request,
            precheck: self.precheck.unwrap_or(false),
            keep_backup: self.keep_backup.unwrap_or(true),
        })
    }
}

// Mirrors Chroma 1.5.2 PersistentData pickle layout used for index_metadata.pickle
// (chromadb/segment/impl/vector/local_persistent_hnsw.py). Keep this in sync.
#[derive(Debug, Deserialize, Serialize, Default)]
struct HnswIdMap {
    dimensionality: Option<usize>,
    total_elements_added: u32,
    #[serde(default)]
    max_seq_id: Option<u64>,
    id_to_label: HashMap<String, u32>,
    label_to_id: HashMap<u32, String>,
    id_to_seq_id: HashMap<String, u32>,
}

impl HnswIdMap {
    fn validate(&self) -> Result<(), String> {
        if self.id_to_label.len() != self.label_to_id.len() {
            return Err(format!(
                "id_to_label/label_to_id length mismatch: {} != {}",
                self.id_to_label.len(),
                self.label_to_id.len()
            ));
        }

        for (id, label) in &self.id_to_label {
            match self.label_to_id.get(label) {
                Some(mapped_id) if mapped_id == id => {}
                Some(mapped_id) => {
                    return Err(format!(
                        "non-bijective mapping for id {id}: label {label} maps to {mapped_id}"
                    ));
                }
                None => {
                    return Err(format!("missing reverse mapping for id {id} label {label}"));
                }
            }
        }

        for (label, id) in &self.label_to_id {
            match self.id_to_label.get(id) {
                Some(mapped_label) if mapped_label == label => {}
                Some(mapped_label) => {
                    return Err(format!(
                        "non-bijective mapping for label {label}: id {id} maps to {mapped_label}"
                    ));
                }
                None => {
                    return Err(format!("missing forward mapping for label {label} id {id}"));
                }
            }
        }

        for id in self.id_to_seq_id.keys() {
            if !self.id_to_label.contains_key(id) {
                return Err(format!("seq-id mapping references unknown id {id}"));
            }
        }

        if let Some(max_seq_id) = self.max_seq_id {
            if let Some(observed_max_seq_id) = self.id_to_seq_id.values().copied().max() {
                let observed_max_seq_id = u64::from(observed_max_seq_id);
                if max_seq_id < observed_max_seq_id {
                    return Err(format!(
                        "max_seq_id {max_seq_id} is smaller than observed seq id {observed_max_seq_id}"
                    ));
                }
            }
        }

        Ok(())
    }

    fn insert_record(
        &mut self,
        user_id: String,
        label: u32,
        seq_id: Option<u32>,
    ) -> Result<(), String> {
        if self.id_to_label.contains_key(&user_id) {
            return Err(format!("duplicate id mapping for {user_id}"));
        }
        if self.label_to_id.contains_key(&label) {
            return Err(format!("duplicate label mapping for {label}"));
        }

        self.id_to_label.insert(user_id.clone(), label);
        self.label_to_id.insert(label, user_id.clone());
        if let Some(seq_id) = seq_id {
            self.id_to_seq_id.insert(user_id, seq_id);
        }
        self.total_elements_added = label;
        Ok(())
    }
}

#[derive(Debug, Serialize)]
struct EmbeddedRebuildCollectionResponse {
    collection_id: String,
    name: String,
    tenant_id: String,
    database_name: String,
    precheck: bool,
    would_rebuild: bool,
    rebuilt: bool,
    records_scanned: u64,
    vectors_reindexed: u64,
    duration_ms: u64,
    backup_path: String,
    warnings: Vec<String>,
}

struct RebuildCollectionIdentity {
    collection_id: String,
    name: String,
    tenant_id: String,
    database_name: String,
}

impl EmbeddedRebuildCollectionResponse {
    fn skipped(
        identity: &RebuildCollectionIdentity,
        precheck: bool,
        warnings: Vec<String>,
    ) -> Self {
        Self {
            collection_id: identity.collection_id.clone(),
            name: identity.name.clone(),
            tenant_id: identity.tenant_id.clone(),
            database_name: identity.database_name.clone(),
            precheck,
            would_rebuild: false,
            rebuilt: false,
            records_scanned: 0,
            vectors_reindexed: 0,
            duration_ms: 0,
            backup_path: String::new(),
            warnings,
        }
    }

    fn precheck(
        identity: &RebuildCollectionIdentity,
        records_scanned: u64,
        warnings: Vec<String>,
    ) -> Self {
        Self {
            collection_id: identity.collection_id.clone(),
            name: identity.name.clone(),
            tenant_id: identity.tenant_id.clone(),
            database_name: identity.database_name.clone(),
            precheck: true,
            would_rebuild: true,
            rebuilt: false,
            records_scanned,
            vectors_reindexed: 0,
            duration_ms: 0,
            backup_path: String::new(),
            warnings,
        }
    }

    fn rebuilt(
        identity: &RebuildCollectionIdentity,
        records_scanned: u64,
        vectors_reindexed: u64,
        duration_ms: u64,
        backup_path: String,
        warnings: Vec<String>,
    ) -> Self {
        Self {
            collection_id: identity.collection_id.clone(),
            name: identity.name.clone(),
            tenant_id: identity.tenant_id.clone(),
            database_name: identity.database_name.clone(),
            precheck: false,
            would_rebuild: true,
            rebuilt: true,
            records_scanned,
            vectors_reindexed,
            duration_ms,
            backup_path,
            warnings,
        }
    }
}

fn load_hnsw_id_map(path: &Path) -> Result<HnswIdMap, String> {
    let file = fs::File::open(path)
        .map_err(|e| format!("failed to open hnsw metadata {}: {e}", path.display()))?;
    let id_map: HnswIdMap = serde_pickle::from_reader(file, DeOptions::new())
        .map_err(|e| format!("failed to decode hnsw metadata {}: {e}", path.display()))?;
    id_map
        .validate()
        .map_err(|e| format!("invalid hnsw metadata {}: {e}", path.display()))?;
    Ok(id_map)
}

fn write_hnsw_id_map(path: &Path, id_map: &HnswIdMap) -> Result<(), String> {
    let file = fs::File::create(path)
        .map_err(|e| format!("failed to create hnsw metadata {}: {e}", path.display()))?;
    let mut writer = std::io::BufWriter::new(file);
    serde_pickle::to_writer(&mut writer, id_map, SerOptions::new())
        .map_err(|e| format!("failed to encode hnsw metadata {}: {e}", path.display()))?;
    writer
        .flush()
        .map_err(|e| format!("failed to flush hnsw metadata {}: {e}", path.display()))
}

fn unique_path_suffix() -> String {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_nanos();
    format!("{now}_{}", std::process::id())
}

fn build_temp_rebuild_dir(source_dir: &Path) -> Result<PathBuf, String> {
    let parent = source_dir
        .parent()
        .ok_or_else(|| format!("invalid source index path {}", source_dir.display()))?;
    let stem = source_dir
        .file_name()
        .ok_or_else(|| format!("invalid source index path {}", source_dir.display()))?
        .to_string_lossy();
    Ok(parent.join(format!("{stem}.rebuild.{}", unique_path_suffix())))
}

struct TempRebuildDirGuard {
    path: PathBuf,
    active: bool,
}

impl TempRebuildDirGuard {
    fn new(path: PathBuf) -> Self {
        Self { path, active: true }
    }

    fn disarm(&mut self) {
        self.active = false;
    }

    fn remove_now(&mut self) -> Result<(), String> {
        if !self.active {
            return Ok(());
        }

        match fs::remove_dir_all(&self.path) {
            Ok(_) => {
                self.active = false;
                Ok(())
            }
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
                self.active = false;
                Ok(())
            }
            Err(e) => Err(format!(
                "failed to remove temporary rebuild directory {}: {e}",
                self.path.display()
            )),
        }
    }
}

impl Drop for TempRebuildDirGuard {
    fn drop(&mut self) {
        if self.active {
            // Drop is a best-effort safety net for panic/early-return paths. We intentionally
            // ignore cleanup errors here because Drop cannot return a Result; explicit error
            // paths call remove_now() to surface failures.
            let _ = fs::remove_dir_all(&self.path);
        }
    }
}

trait SwapFsOps {
    fn rename(&self, from: &Path, to: &Path) -> std::io::Result<()>;
    fn remove_dir_all(&self, path: &Path) -> std::io::Result<()>;
}

struct StdSwapFsOps;

const WINDOWS_FS_RETRY_DELAYS_MS: [u64; 6] = [10, 25, 50, 100, 200, 400];

fn is_windows_retryable_fs_error(err: &std::io::Error) -> bool {
    #[cfg(windows)]
    {
        // ERROR_ACCESS_DENIED (5), ERROR_SHARING_VIOLATION (32), ERROR_LOCK_VIOLATION (33)
        matches!(err.raw_os_error(), Some(5 | 32 | 33))
            || err.kind() == std::io::ErrorKind::PermissionDenied
    }
    #[cfg(not(windows))]
    {
        let _ = err;
        false
    }
}

fn run_with_windows_fs_retry<F>(mut op: F) -> std::io::Result<()>
where
    F: FnMut() -> std::io::Result<()>,
{
    let mut attempt = 0usize;
    loop {
        match op() {
            Ok(()) => return Ok(()),
            Err(err) => {
                if !is_windows_retryable_fs_error(&err)
                    || attempt >= WINDOWS_FS_RETRY_DELAYS_MS.len()
                {
                    return Err(err);
                }
                let delay_ms = WINDOWS_FS_RETRY_DELAYS_MS[attempt];
                attempt += 1;
                std::thread::sleep(std::time::Duration::from_millis(delay_ms));
            }
        }
    }
}

impl SwapFsOps for StdSwapFsOps {
    fn rename(&self, from: &Path, to: &Path) -> std::io::Result<()> {
        run_with_windows_fs_retry(|| fs::rename(from, to))
    }

    fn remove_dir_all(&self, path: &Path) -> std::io::Result<()> {
        run_with_windows_fs_retry(|| fs::remove_dir_all(path))
    }
}

fn swap_rebuilt_index_dir(
    source_dir: &Path,
    rebuilt_dir: &Path,
    keep_backup: bool,
) -> Result<(String, Vec<String>), String> {
    let fs_ops = StdSwapFsOps;
    swap_rebuilt_index_dir_with_ops(
        source_dir,
        rebuilt_dir,
        keep_backup,
        unique_path_suffix(),
        &fs_ops,
    )
}

fn rebuild_required_capacity(record_count: usize, resize_factor: f64) -> usize {
    let live_records = record_count.max(1);
    let scaled_capacity = ((live_records as f64) * resize_factor).ceil() as usize;
    scaled_capacity.max(record_count.saturating_add(1)).max(100)
}

fn swap_rebuilt_index_dir_with_ops(
    source_dir: &Path,
    rebuilt_dir: &Path,
    keep_backup: bool,
    suffix: String,
    fs_ops: &dyn SwapFsOps,
) -> Result<(String, Vec<String>), String> {
    let parent = source_dir
        .parent()
        .ok_or_else(|| format!("invalid source index path {}", source_dir.display()))?;
    let name = source_dir
        .file_name()
        .ok_or_else(|| format!("invalid source index path {}", source_dir.display()))?
        .to_string_lossy();

    let moved_source = if keep_backup {
        parent.join(format!("{name}_backup_{suffix}"))
    } else {
        parent.join(format!("{name}_rollback_{suffix}"))
    };

    fs_ops.rename(source_dir, &moved_source).map_err(|e| {
        format!(
            "failed to stage existing index {} -> {}: {e}",
            source_dir.display(),
            moved_source.display()
        )
    })?;

    if let Err(swap_err) = fs_ops.rename(rebuilt_dir, source_dir) {
        let rollback_result = fs_ops.rename(&moved_source, source_dir);
        return Err(format_swap_activation_error(
            rebuilt_dir,
            source_dir,
            &swap_err,
            rollback_result.err().as_ref(),
        ));
    }

    if keep_backup {
        return Ok((moved_source.to_string_lossy().to_string(), Vec::new()));
    }

    let mut warnings = Vec::new();
    if let Err(remove_err) = fs_ops.remove_dir_all(&moved_source) {
        warnings.push(format!(
            "failed to remove rollback directory {}: {remove_err}",
            moved_source.display()
        ));
    }
    Ok((String::new(), warnings))
}

fn format_swap_activation_error(
    rebuilt_dir: &Path,
    source_dir: &Path,
    swap_err: &std::io::Error,
    rollback_err: Option<&std::io::Error>,
) -> String {
    match rollback_err {
        Some(rollback_err) => format!(
            "failed to activate rebuilt index {} -> {}: {swap_err}; rollback failed: {rollback_err}",
            rebuilt_dir.display(),
            source_dir.display()
        ),
        None => format!(
            "failed to activate rebuilt index {} -> {}: {swap_err}; rollback succeeded",
            rebuilt_dir.display(),
            source_dir.display()
        ),
    }
}

// NOTE: This relies on current upstream error string content from chroma-index/hnswlib.
// Keep as a best-effort fallback until typed error variants are exposed.
fn is_hnsw_label_not_found(error_message: &str) -> bool {
    error_message.contains("Label not found")
}

async fn run_rebuild_collection(
    frontend: &mut Frontend,
    registry: &Registry,
    persist_root: &Path,
    request: GetCollectionRequest,
    precheck: bool,
    keep_backup: bool,
) -> Result<EmbeddedRebuildCollectionResponse, String> {
    let collection = frontend
        .get_collection(request)
        .await
        .map_err(|e| format!("get collection failed: {e}"))?;
    let response_identity = RebuildCollectionIdentity {
        collection_id: collection.collection_id.to_string(),
        name: collection.name.clone(),
        tenant_id: collection.tenant.clone(),
        database_name: collection.database.clone(),
    };

    let mut sysdb = registry
        .get::<SysDb>()
        .map_err(|e| format!("sysdb unavailable: {e}"))?;
    let local_segment_manager = registry
        .get::<LocalSegmentManager>()
        .map_err(|e| format!("local segment manager unavailable: {e}"))?;

    let mut collection_and_segments = sysdb
        .get_collection_with_segments(None, collection.collection_id)
        .await
        .map_err(|e| format!("get collection with segments failed: {e}"))?;

    if collection_and_segments.collection.schema.is_none() {
        collection_and_segments.collection.schema = Some(
            Schema::try_from(&collection_and_segments.collection.config)
                .map_err(|e| format!("failed to reconcile collection schema: {e}"))?,
        );
    }

    let hnsw_config = collection_and_segments
        .collection
        .schema
        .as_ref()
        .map(|schema| {
            schema.get_internal_hnsw_config_with_legacy_fallback(
                &collection_and_segments.vector_segment,
            )
        })
        .transpose()
        .map_err(|e| format!("failed to load hnsw configuration: {e}"))?
        .flatten()
        .ok_or_else(|| "collection is missing hnsw configuration".to_string())?;

    let source_dir = persist_root.join(collection_and_segments.vector_segment.id.to_string());
    let metadata_path = source_dir.join(HNSW_METADATA_FILENAME);

    let mut warnings = Vec::new();
    if !source_dir.exists() {
        warnings.push(format!(
            "vector index directory {} does not exist",
            source_dir.display()
        ));
        return Ok(EmbeddedRebuildCollectionResponse::skipped(
            &response_identity,
            precheck,
            warnings,
        ));
    }

    if !metadata_path.exists() {
        warnings.push(format!(
            "hnsw metadata file {} does not exist",
            metadata_path.display()
        ));
        return Ok(EmbeddedRebuildCollectionResponse::skipped(
            &response_identity,
            precheck,
            warnings,
        ));
    }

    let source_id_map = load_hnsw_id_map(&metadata_path)?;
    if source_id_map.id_to_label.is_empty() {
        warnings.push("index metadata contains no records; rebuild skipped".to_string());
        return Ok(EmbeddedRebuildCollectionResponse::skipped(
            &response_identity,
            precheck,
            warnings,
        ));
    }

    let dimensionality = source_id_map
        .dimensionality
        .or(collection_and_segments
            .collection
            .dimension
            .map(|d| d as usize))
        .ok_or_else(|| "collection dimension is unknown; cannot rebuild".to_string())?;
    let source_dir_str = source_dir.to_str().ok_or_else(|| {
        format!(
            "source index path is not valid UTF-8: {}",
            source_dir.display()
        )
    })?;

    let index_config = IndexConfig::new(dimensionality as i32, hnsw_config.space.clone().into());
    let source_index = HnswIndex::load(
        source_dir_str,
        &index_config,
        hnsw_config.ef_search,
        IndexUuid(collection_and_segments.vector_segment.id.0),
    )
    .map_err(|e| format!("failed to load source hnsw index: {e}"))?;

    let mut rebuild_records: Vec<(String, u32, Option<u32>)> = source_id_map
        .id_to_label
        .iter()
        .map(|(user_id, label)| {
            (
                user_id.clone(),
                *label,
                source_id_map.id_to_seq_id.get(user_id).copied(),
            )
        })
        .collect();
    rebuild_records.sort_by_key(|(_, label, _)| *label);
    let records_scanned = rebuild_records.len() as u64;

    if precheck {
        return Ok(EmbeddedRebuildCollectionResponse::precheck(
            &response_identity,
            records_scanned,
            warnings,
        ));
    }

    let started_at = Instant::now();
    let rebuilt_dir = build_temp_rebuild_dir(&source_dir)?;
    fs::create_dir(&rebuilt_dir).map_err(|e| {
        format!(
            "failed to create temporary rebuild directory {}: {e}",
            rebuilt_dir.display()
        )
    })?;
    let mut rebuilt_dir_guard = TempRebuildDirGuard::new(rebuilt_dir.clone());

    let required_capacity =
        rebuild_required_capacity(rebuild_records.len(), hnsw_config.resize_factor);

    let mut target_config = HnswIndexConfig::new_persistent(
        hnsw_config.max_neighbors,
        hnsw_config.ef_construction,
        hnsw_config.ef_search,
        &rebuilt_dir,
    )
    .map_err(|e| format!("failed to create hnsw target configuration: {e}"))?;
    target_config.max_elements = required_capacity;

    let mut rebuilt_index = HnswIndex::init(
        &index_config,
        Some(&target_config),
        IndexUuid(collection_and_segments.vector_segment.id.0),
    )
    .map_err(|e| format!("failed to initialize rebuilt hnsw index: {e}"))?;
    if rebuilt_index.capacity() < required_capacity {
        let next_pow2 = required_capacity.next_power_of_two();
        rebuilt_index
            .resize(next_pow2)
            .map_err(|e| format!("failed to resize rebuilt hnsw index: {e}"))?;
    }

    let mut rebuilt_id_map = HnswIdMap {
        dimensionality: Some(dimensionality),
        total_elements_added: 0,
        max_seq_id: source_id_map.max_seq_id,
        id_to_label: HashMap::new(),
        label_to_id: HashMap::new(),
        id_to_seq_id: HashMap::new(),
    };

    let mut vectors_reindexed: u64 = 0;
    for (user_id, source_label, seq_id) in rebuild_records {
        let embedding = match source_index.get(source_label as usize) {
            Ok(embedding) => embedding,
            Err(e) => {
                let msg = e.to_string();
                if is_hnsw_label_not_found(&msg) {
                    return Err(format!(
                        "source index is inconsistent: missing embedding for label {source_label} referenced by id {user_id}"
                    ));
                }
                return Err(format!(
                    "failed to read source embedding for label {source_label}: {e}"
                ));
            }
        };
        let Some(embedding) = embedding else {
            return Err(format!(
                "source index is inconsistent: missing embedding for label {source_label} referenced by id {user_id}"
            ));
        };

        let next_label = rebuilt_id_map
            .total_elements_added
            .checked_add(1)
            .ok_or_else(|| "label counter overflow while rebuilding index".to_string())?;
        rebuilt_index
            .add(next_label as usize, &embedding)
            .map_err(|e| format!("failed to add embedding for label {next_label}: {e}"))?;

        rebuilt_id_map
            .insert_record(user_id.clone(), next_label, seq_id)
            .map_err(|e| format!("failed to update rebuilt metadata for id {user_id}: {e}"))?;
        vectors_reindexed = vectors_reindexed.saturating_add(1);
    }

    rebuilt_index
        .save()
        .map_err(|e| format!("failed to persist rebuilt hnsw index: {e}"))?;
    write_hnsw_id_map(&rebuilt_dir.join(HNSW_METADATA_FILENAME), &rebuilt_id_map)?;

    // Ensure all HNSW file handles are closed before attempting directory swap on Windows.
    drop(source_index);
    drop(rebuilt_index);

    local_segment_manager
        .reset()
        .await
        .map_err(|e| format!("failed to reset local segment manager before swap: {e}"))?;

    let swap_result = swap_rebuilt_index_dir(&source_dir, &rebuilt_dir, keep_backup);
    let (backup_path, mut swap_warnings) = match swap_result {
        Ok(result) => result,
        Err(mut e) => {
            if let Err(reset_err) = local_segment_manager.reset().await {
                e = format!(
                    "{e}; failed to reset local segment manager after swap failure: {reset_err}"
                );
            }
            if let Err(cleanup_err) = rebuilt_dir_guard.remove_now() {
                e = format!("{e}; {cleanup_err}");
            }
            return Err(e);
        }
    };
    rebuilt_dir_guard.disarm();
    warnings.append(&mut swap_warnings);

    local_segment_manager
        .reset()
        .await
        .map_err(|e| format!("failed to reset local segment manager after swap: {e}"))?;

    let duration_ms = started_at.elapsed().as_millis().min(u128::from(u64::MAX)) as u64;
    Ok(EmbeddedRebuildCollectionResponse::rebuilt(
        &response_identity,
        records_scanned,
        vectors_reindexed,
        duration_ms,
        backup_path,
        warnings,
    ))
}

fn compaction_target_from_parts(
    collection_id: CollectionUuid,
    name: String,
    tenant_id: String,
    database_name_raw: String,
) -> Result<CompactionTarget, String> {
    let database_name = parse_database_name(database_name_raw.clone())?;
    Ok(CompactionTarget {
        collection_id,
        name,
        tenant_id,
        database_name,
        database_name_raw,
    })
}

async fn run_explicit_compaction(
    frontend: &mut Frontend,
    registry: &Registry,
    targets: Vec<CompactionTarget>,
    failure_mode: CompactionFailureMode,
) -> Result<EmbeddedCompactionResponse, String> {
    if targets.is_empty() {
        return Ok(EmbeddedCompactionResponse {
            collection_count: 0,
            duration_ms: 0,
            pending_ops_before_total: 0,
            pending_ops_after_total: 0,
            collections: Vec::new(),
        });
    }

    let compactor_handle = registry
        .get::<ComponentHandle<LocalCompactionManager>>()
        .map_err(|e| format!("local compaction manager unavailable: {e}"))?;

    let started_at = Instant::now();
    let mut pending_ops_before_total: u64 = 0;
    let mut pending_ops_after_total: u64 = 0;
    let mut collection_results = Vec::with_capacity(targets.len());

    for target in targets {
        let CompactionTarget {
            collection_id,
            name,
            tenant_id,
            database_name,
            database_name_raw,
        } = target;

        let (pending_ops_before, pending_ops_before_error) = match frontend
            .indexing_status(database_name.clone(), collection_id)
            .await
        {
            Ok(status) => (Some(status.num_unindexed_ops), None),
            Err(e) => {
                let warning = format!(
                    "indexing_status failed before compaction for {} ({}): {e}",
                    collection_id, name
                );
                eprintln!("warning: {warning}");
                (None, Some(warning))
            }
        };
        if let Some(value) = pending_ops_before {
            pending_ops_before_total = pending_ops_before_total.saturating_add(value);
        }

        let compaction_error = {
            let compaction_result: Result<(), String> = async {
                compactor_handle
                    .request(BackfillMessage { collection_id }, None)
                    .await
                    .map_err(|e| {
                        format!(
                            "failed to send backfill message for collection {}: {e}",
                            collection_id
                        )
                    })?
                    .map_err(|e| {
                        format!("backfill failed for collection {}: {e}", collection_id)
                    })?;

                compactor_handle
                    .request(PurgeLogsMessage { collection_id }, None)
                    .await
                    .map_err(|e| {
                        format!(
                            "failed to send purge message for collection {}: {e}",
                            collection_id
                        )
                    })?
                    .map_err(|e| format!("purge failed for collection {}: {e}", collection_id))?;
                Ok(())
            }
            .await;

            compaction_result.err()
        };

        if let Some(error) = compaction_error {
            match failure_mode {
                CompactionFailureMode::FailFast => return Err(error),
                CompactionFailureMode::ContinueOnError => {
                    collection_results.push(EmbeddedCompactionCollectionResult {
                        collection_id: collection_id.to_string(),
                        name,
                        tenant_id,
                        database_name: database_name_raw,
                        pending_ops_before,
                        pending_ops_after: None,
                        pending_ops_before_error,
                        pending_ops_after_error: None,
                        error: Some(error),
                    });
                    continue;
                }
            }
        }

        let (pending_ops_after, pending_ops_after_error) = match frontend
            .indexing_status(database_name.clone(), collection_id)
            .await
        {
            Ok(status) => (Some(status.num_unindexed_ops), None),
            Err(e) => {
                let warning = format!(
                    "indexing_status failed after compaction for {} ({}): {e}",
                    collection_id, name
                );
                eprintln!("warning: {warning}");
                (None, Some(warning))
            }
        };
        if let Some(value) = pending_ops_after {
            pending_ops_after_total = pending_ops_after_total.saturating_add(value);
        }

        collection_results.push(EmbeddedCompactionCollectionResult {
            collection_id: collection_id.to_string(),
            name,
            tenant_id,
            database_name: database_name_raw,
            pending_ops_before,
            pending_ops_after,
            pending_ops_before_error,
            pending_ops_after_error,
            error: None,
        });
    }

    let duration_ms = started_at.elapsed().as_millis().min(u128::from(u64::MAX)) as u64;
    let collection_count = u32::try_from(collection_results.len()).unwrap_or(u32::MAX);
    Ok(EmbeddedCompactionResponse {
        collection_count,
        duration_ms,
        pending_ops_before_total,
        pending_ops_after_total,
        collections: collection_results,
    })
}

async fn get_collection_ids_to_migrate(sqlite: &SqliteDb) -> Result<Vec<CollectionUuid>, String> {
    let rows = sqlx::query(
        r#"
        SELECT collection
        FROM segments
        WHERE id NOT IN (SELECT segment_id FROM max_seq_id)
          AND type = 'urn:chroma:segment/vector/hnsw-local-persisted'
        "#,
    )
    .fetch_all(sqlite.get_conn())
    .await
    .map_err(|e| format!("failed to list segments missing max_seq_id: {e}"))?;

    rows.into_iter()
        .map(|row| {
            let raw: String = row
                .try_get(0)
                .map_err(|e| format!("failed to decode missing max_seq_id row: {e}"))?;
            CollectionUuid::from_str(&raw)
                .map_err(|e| format!("invalid collection id in segments table: {e}"))
        })
        .collect()
}

async fn trigger_vector_segments_max_seq_id_migration(
    sqlite: &SqliteDb,
    sysdb: &mut SysDb,
    segment_manager: &LocalSegmentManager,
) -> Result<(), String> {
    let collection_ids = get_collection_ids_to_migrate(sqlite).await?;

    for collection_id in collection_ids {
        let collection = sysdb
            .get_collection_with_segments(None, collection_id)
            .await
            .map_err(|e| format!("failed to load collection for max_seq_id migration: {e}"))?;

        // If collection is uninitialized, there is no sequence id to materialize.
        let dim = match collection.collection.dimension {
            Some(dim) => dim,
            None => continue,
        };

        segment_manager
            .get_hnsw_writer(
                &collection.collection,
                &collection.vector_segment,
                dim as usize,
            )
            .await
            .map_err(|e| format!("failed to initialize hnsw writer for migration: {e}"))?;
    }

    Ok(())
}

async fn configure_sqlite_log_auto_purge(log: &Log) -> Result<(), String> {
    match log {
        Log::Sqlite(sqlite_log) => {
            let config = LegacyEmbeddingsQueueConfig {
                automatically_purge: true,
                kind: legacy_embeddings_queue_config_default_kind(),
            };
            sqlite_log
                .update_legacy_embeddings_queue_config(config)
                .await
                .map_err(|e| format!("failed to configure sqlite log auto purge: {e}"))?;
            Ok(())
        }
        _ => Err("expected a sqlite log for wal prune".to_string()),
    }
}

async fn query_collection_safe_seq_cutoff(
    sqlite: &SqliteDb,
    collection_id: CollectionUuid,
) -> Result<Option<i64>, String> {
    let row = sqlx::query(
        r#"
        SELECT MIN(COALESCE(CAST(max_seq_id.seq_id AS INTEGER), -1)) AS min_seq
        FROM segments
        LEFT JOIN max_seq_id ON segments.id = max_seq_id.segment_id
        WHERE segments.collection = ?
        "#,
    )
    .bind(collection_id.to_string())
    .fetch_one(sqlite.get_conn())
    .await
    .map_err(|e| format!("failed to query safe seq cutoff: {e}"))?;

    let min_seq: Option<i64> = row
        .try_get("min_seq")
        .map_err(|e| format!("failed to decode safe seq cutoff: {e}"))?;
    match min_seq {
        Some(value) if value > 0 => Ok(Some(value)),
        _ => Ok(None),
    }
}

async fn query_wal_candidate_rows(
    sqlite: &SqliteDb,
    collection_id: CollectionUuid,
    safe_seq_cutoff: i64,
) -> Result<Vec<WalPruneCandidateRow>, String> {
    if safe_seq_cutoff <= 0 {
        return Ok(Vec::new());
    }

    let topic_suffix = format!("%/{}", collection_id);
    let rows = sqlx::query(
        r#"
        SELECT
            seq_id,
            COALESCE(CAST(strftime('%s', created_at) AS INTEGER), 0) AS created_at_secs,
            (
                LENGTH(COALESCE(id, '')) +
                LENGTH(COALESCE(metadata, '')) +
                LENGTH(COALESCE(encoding, '')) +
                LENGTH(COALESCE(vector, x'')) +
                32
            ) AS estimated_bytes
        FROM embeddings_queue
        WHERE topic LIKE ?
          AND seq_id < ?
        ORDER BY seq_id ASC
        "#,
    )
    .bind(topic_suffix)
    .bind(safe_seq_cutoff)
    .fetch_all(sqlite.get_conn())
    .await
    .map_err(|e| format!("failed to query wal candidates: {e}"))?;

    rows.into_iter()
        .map(|row| {
            let seq_id: i64 = row
                .try_get("seq_id")
                .map_err(|e| format!("failed to decode candidate seq_id: {e}"))?;
            let created_at_secs: i64 = row
                .try_get("created_at_secs")
                .map_err(|e| format!("failed to decode candidate created_at_secs: {e}"))?;
            let estimated_bytes_i64: i64 = row
                .try_get("estimated_bytes")
                .map_err(|e| format!("failed to decode candidate estimated_bytes: {e}"))?;
            Ok(WalPruneCandidateRow {
                seq_id,
                created_at_secs,
                estimated_bytes: u64::try_from(estimated_bytes_i64.max(0)).unwrap_or(u64::MAX),
            })
        })
        .collect()
}

fn wal_candidate_total_bytes(rows: &[WalPruneCandidateRow]) -> u64 {
    rows.iter()
        .fold(0u64, |acc, row| acc.saturating_add(row.estimated_bytes))
}

fn wal_prune_prefix_for_max_age(
    rows: &[WalPruneCandidateRow],
    max_age_seconds: u64,
    now_secs: i64,
) -> usize {
    let max_age_i64 = i64::try_from(max_age_seconds).unwrap_or(i64::MAX);
    let cutoff = now_secs.saturating_sub(max_age_i64);
    let mut prefix = 0usize;
    for row in rows {
        if row.created_at_secs < cutoff {
            prefix += 1;
            continue;
        }
        break;
    }
    prefix
}

fn wal_prune_prefix_for_max_bytes(rows: &[WalPruneCandidateRow], max_bytes: u64) -> usize {
    let total_bytes = wal_candidate_total_bytes(rows);
    if total_bytes <= max_bytes {
        return 0;
    }

    let mut removed = 0u64;
    let mut prefix = 0usize;
    for row in rows {
        removed = removed.saturating_add(row.estimated_bytes);
        prefix += 1;
        if total_bytes.saturating_sub(removed) <= max_bytes {
            break;
        }
    }
    prefix
}

fn wal_prune_prefix_for_watermark(
    rows: &[WalPruneCandidateRow],
    high_bytes: u64,
    low_bytes: u64,
) -> usize {
    let total_bytes = wal_candidate_total_bytes(rows);
    if total_bytes <= high_bytes {
        return 0;
    }

    let mut removed = 0u64;
    let mut prefix = 0usize;
    for row in rows {
        removed = removed.saturating_add(row.estimated_bytes);
        prefix += 1;
        if total_bytes.saturating_sub(removed) <= low_bytes {
            break;
        }
    }
    prefix
}

fn wal_prune_prefix_count(
    rows: &[WalPruneCandidateRow],
    policies: &WalPrunePolicyPayload,
    now_secs: i64,
) -> usize {
    if rows.is_empty() || !policies.has_policy() {
        return 0;
    }

    let mut prefix = rows.len();
    if let Some(max_age_seconds) = policies.max_age_seconds {
        prefix = prefix.min(wal_prune_prefix_for_max_age(
            rows,
            max_age_seconds,
            now_secs,
        ));
    }
    if let Some(max_bytes) = policies.max_bytes {
        prefix = prefix.min(wal_prune_prefix_for_max_bytes(rows, max_bytes));
    }
    if let (Some(high_bytes), Some(low_bytes)) =
        (policies.watermark_high_bytes, policies.watermark_low_bytes)
    {
        prefix = prefix.min(wal_prune_prefix_for_watermark(rows, high_bytes, low_bytes));
    }
    prefix
}

fn seq_opt_u64(value: Option<i64>, field: &str) -> Result<Option<u64>, String> {
    match value {
        Some(v) => u64::try_from(v)
            .map(Some)
            .map_err(|_| format!("negative sequence id for {field}: {v}")),
        None => Ok(None),
    }
}

async fn run_explicit_wal_prune(
    _frontend: &mut Frontend,
    registry: &Registry,
    targets: Vec<WalPruneTarget>,
    options: WalPruneExecutionOptions,
    failure_mode: WalPruneFailureMode,
) -> Result<EmbeddedWalPruneResponse, String> {
    if targets.is_empty() {
        return Ok(EmbeddedWalPruneResponse {
            collection_count: 0,
            duration_ms: 0,
            dry_run: options.dry_run,
            vacuum_requested: options.vacuum,
            vacuum_executed: false,
            warning: None,
            candidate_count_total: 0,
            candidate_bytes_total: 0,
            pruned_count_total: 0,
            pruned_bytes_total: 0,
            collections: Vec::new(),
        });
    }

    let sqlite = registry
        .get::<SqliteDb>()
        .map_err(|e| format!("sqlite db unavailable: {e}"))?;
    let segment_manager = registry
        .get::<LocalSegmentManager>()
        .map_err(|e| format!("local segment manager unavailable: {e}"))?;
    let mut sysdb = registry
        .get::<SysDb>()
        .map_err(|e| format!("sysdb unavailable: {e}"))?;
    let mut log = registry
        .get::<Log>()
        .map_err(|e| format!("log unavailable: {e}"))?;

    trigger_vector_segments_max_seq_id_migration(&sqlite, &mut sysdb, &segment_manager).await?;
    if !options.dry_run {
        configure_sqlite_log_auto_purge(&log).await?;
    }

    let started_at = Instant::now();
    let now_secs = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|e| format!("system clock is before UNIX epoch: {e}"))?
        .as_secs();
    let now_secs_i64 = i64::try_from(now_secs).unwrap_or(i64::MAX);

    let mut candidate_count_total = 0u64;
    let mut candidate_bytes_total = 0u64;
    let mut pruned_count_total = 0u64;
    let mut pruned_bytes_total = 0u64;
    let mut warning: Option<String> = None;
    let mut collection_results: Vec<EmbeddedWalPruneCollectionResult> =
        Vec::with_capacity(targets.len());

    for target in targets {
        let WalPruneTarget {
            collection_id,
            name,
            tenant_id,
            database_name_raw,
        } = target;

        let safe_seq_cutoff = match query_collection_safe_seq_cutoff(&sqlite, collection_id).await {
            Ok(value) => value,
            Err(error) => match failure_mode {
                WalPruneFailureMode::FailFast => return Err(error),
                WalPruneFailureMode::ContinueOnError => {
                    collection_results.push(EmbeddedWalPruneCollectionResult {
                        collection_id: collection_id.to_string(),
                        name: name.clone(),
                        tenant_id: tenant_id.clone(),
                        database_name: database_name_raw.clone(),
                        safe_seq_cutoff: None,
                        candidate_seq_min: None,
                        candidate_seq_max: None,
                        pruned_seq_min: None,
                        pruned_seq_max: None,
                        candidate_count: 0,
                        candidate_bytes: 0,
                        pruned_count: 0,
                        pruned_bytes: 0,
                        error: Some(error),
                    });
                    continue;
                }
            },
        };

        let candidate_rows = match safe_seq_cutoff {
            Some(cutoff) => match query_wal_candidate_rows(&sqlite, collection_id, cutoff).await {
                Ok(rows) => rows,
                Err(error) => match failure_mode {
                    WalPruneFailureMode::FailFast => return Err(error),
                    WalPruneFailureMode::ContinueOnError => {
                        collection_results.push(EmbeddedWalPruneCollectionResult {
                            collection_id: collection_id.to_string(),
                            name: name.clone(),
                            tenant_id: tenant_id.clone(),
                            database_name: database_name_raw.clone(),
                            safe_seq_cutoff: seq_opt_u64(safe_seq_cutoff, "safe_seq_cutoff")?,
                            candidate_seq_min: None,
                            candidate_seq_max: None,
                            pruned_seq_min: None,
                            pruned_seq_max: None,
                            candidate_count: 0,
                            candidate_bytes: 0,
                            pruned_count: 0,
                            pruned_bytes: 0,
                            error: Some(error),
                        });
                        continue;
                    }
                },
            },
            None => Vec::new(),
        };
        if let Some(negative_row) = candidate_rows.iter().find(|row| row.seq_id < 0) {
            let error = format!(
                "negative wal seq_id encountered for collection {}: {}",
                collection_id, negative_row.seq_id
            );
            match failure_mode {
                WalPruneFailureMode::FailFast => return Err(error),
                WalPruneFailureMode::ContinueOnError => {
                    collection_results.push(EmbeddedWalPruneCollectionResult {
                        collection_id: collection_id.to_string(),
                        name: name.clone(),
                        tenant_id: tenant_id.clone(),
                        database_name: database_name_raw.clone(),
                        safe_seq_cutoff: seq_opt_u64(safe_seq_cutoff, "safe_seq_cutoff")?,
                        candidate_seq_min: None,
                        candidate_seq_max: None,
                        pruned_seq_min: None,
                        pruned_seq_max: None,
                        candidate_count: 0,
                        candidate_bytes: 0,
                        pruned_count: 0,
                        pruned_bytes: 0,
                        error: Some(error),
                    });
                    continue;
                }
            }
        }

        let candidate_count = u64::try_from(candidate_rows.len()).unwrap_or(u64::MAX);
        let candidate_bytes = wal_candidate_total_bytes(&candidate_rows);
        let prune_prefix = wal_prune_prefix_count(&candidate_rows, &options.policies, now_secs_i64);
        let selected_rows = &candidate_rows[..prune_prefix];
        let pruned_count = u64::try_from(selected_rows.len()).unwrap_or(u64::MAX);
        let pruned_bytes = wal_candidate_total_bytes(selected_rows);

        if !options.dry_run && prune_prefix > 0 {
            let last_selected_seq = selected_rows
                .last()
                .map(|row| row.seq_id)
                .ok_or_else(|| "internal wal prune error: selected rows empty".to_string())?;
            let purge_exclusive = u64::try_from(last_selected_seq)
                .map_err(|_| format!("invalid WAL seq_id for purge: {last_selected_seq}"))?
                .checked_add(1)
                .ok_or_else(|| "purge sequence id overflow".to_string())?;

            if let Err(error) = log.purge_logs(collection_id, purge_exclusive).await {
                let message = format!("purge failed for collection {}: {error}", collection_id);
                match failure_mode {
                    WalPruneFailureMode::FailFast => return Err(message),
                    WalPruneFailureMode::ContinueOnError => {
                        collection_results.push(EmbeddedWalPruneCollectionResult {
                            collection_id: collection_id.to_string(),
                            name,
                            tenant_id,
                            database_name: database_name_raw,
                            safe_seq_cutoff: seq_opt_u64(safe_seq_cutoff, "safe_seq_cutoff")?,
                            candidate_seq_min: seq_opt_u64(
                                candidate_rows.first().map(|r| r.seq_id),
                                "candidate_seq_min",
                            )?,
                            candidate_seq_max: seq_opt_u64(
                                candidate_rows.last().map(|r| r.seq_id),
                                "candidate_seq_max",
                            )?,
                            pruned_seq_min: seq_opt_u64(
                                selected_rows.first().map(|r| r.seq_id),
                                "pruned_seq_min",
                            )?,
                            pruned_seq_max: seq_opt_u64(
                                selected_rows.last().map(|r| r.seq_id),
                                "pruned_seq_max",
                            )?,
                            candidate_count,
                            candidate_bytes,
                            pruned_count: 0,
                            pruned_bytes: 0,
                            error: Some(message),
                        });
                        candidate_count_total =
                            candidate_count_total.saturating_add(candidate_count);
                        candidate_bytes_total =
                            candidate_bytes_total.saturating_add(candidate_bytes);
                        continue;
                    }
                }
            }
        }

        candidate_count_total = candidate_count_total.saturating_add(candidate_count);
        candidate_bytes_total = candidate_bytes_total.saturating_add(candidate_bytes);
        pruned_count_total = pruned_count_total.saturating_add(pruned_count);
        pruned_bytes_total = pruned_bytes_total.saturating_add(pruned_bytes);

        collection_results.push(EmbeddedWalPruneCollectionResult {
            collection_id: collection_id.to_string(),
            name,
            tenant_id,
            database_name: database_name_raw,
            safe_seq_cutoff: seq_opt_u64(safe_seq_cutoff, "safe_seq_cutoff")?,
            candidate_seq_min: seq_opt_u64(
                candidate_rows.first().map(|r| r.seq_id),
                "candidate_seq_min",
            )?,
            candidate_seq_max: seq_opt_u64(
                candidate_rows.last().map(|r| r.seq_id),
                "candidate_seq_max",
            )?,
            pruned_seq_min: seq_opt_u64(selected_rows.first().map(|r| r.seq_id), "pruned_seq_min")?,
            pruned_seq_max: seq_opt_u64(selected_rows.last().map(|r| r.seq_id), "pruned_seq_max")?,
            candidate_count,
            candidate_bytes,
            pruned_count,
            pruned_bytes,
            error: None,
        });
    }

    let mut vacuum_executed = false;
    if !options.dry_run && options.vacuum && pruned_count_total > 0 {
        match sqlx::query("PRAGMA busy_timeout = 5000")
            .execute(sqlite.get_conn())
            .await
        {
            Ok(_) => match sqlx::query("VACUUM").execute(sqlite.get_conn()).await {
                Ok(_) => {
                    vacuum_executed = true;
                }
                Err(e) => {
                    warning = Some(format!(
                        "wal prune completed, but sqlite VACUUM failed: {e}"
                    ));
                }
            },
            Err(e) => {
                warning = Some(format!(
                    "wal prune completed, but sqlite VACUUM was skipped because busy_timeout configuration failed: {e}"
                ));
            }
        }
    }

    let duration_ms = started_at.elapsed().as_millis().min(u128::from(u64::MAX)) as u64;
    let collection_count = u32::try_from(collection_results.len()).unwrap_or(u32::MAX);
    Ok(EmbeddedWalPruneResponse {
        collection_count,
        duration_ms,
        dry_run: options.dry_run,
        vacuum_requested: options.vacuum,
        vacuum_executed,
        warning,
        candidate_count_total,
        candidate_bytes_total,
        pruned_count_total,
        pruned_bytes_total,
        collections: collection_results,
    })
}

async fn list_wal_prune_targets(
    frontend: &mut Frontend,
    tenant_id: String,
    database_name: Option<String>,
) -> Result<Vec<WalPruneTarget>, String> {
    let mut database_names: Vec<String> = Vec::new();
    if let Some(database_name) = database_name {
        database_names.push(database_name);
    } else {
        let page_size: u32 = 100;
        let mut offset: u32 = 0;
        loop {
            let request = ListDatabasesRequest::try_new(tenant_id.clone(), Some(page_size), offset)
                .map_err(|e| e.to_string())?;
            let databases = frontend
                .list_databases(request)
                .await
                .map_err(|e| format!("list databases failed: {e}"))?;
            if databases.is_empty() {
                break;
            }

            let count = u32::try_from(databases.len()).unwrap_or(u32::MAX);
            database_names.extend(databases.into_iter().map(|database| database.name));
            if count < page_size {
                break;
            }
            offset = offset.saturating_add(count);
        }
    }

    let mut targets = Vec::new();
    for database_name in database_names {
        let parsed_database_name = parse_database_name(database_name.clone())?;
        let page_size: u32 = 100;
        let mut offset: u32 = 0;
        loop {
            let request = ListCollectionsRequest::try_new(
                tenant_id.clone(),
                parsed_database_name.clone(),
                Some(page_size),
                offset,
            )
            .map_err(|e| e.to_string())?;
            let collections = frontend.list_collections(request).await.map_err(|e| {
                format!(
                    "list collections failed for database {}: {e}",
                    database_name
                )
            })?;
            if collections.is_empty() {
                break;
            }

            let count = u32::try_from(collections.len()).unwrap_or(u32::MAX);
            targets.extend(collections.into_iter().map(|collection| WalPruneTarget {
                collection_id: collection.collection_id,
                name: collection.name,
                tenant_id: collection.tenant,
                database_name_raw: collection.database,
            }));
            if count < page_size {
                break;
            }
            offset = offset.saturating_add(count);
        }
    }

    Ok(targets)
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
        let database_name = parse_database_name(self.name)?;
        CreateDatabaseRequest::try_new(tenant_id, database_name).map_err(|e| e.to_string())
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
        let database_name = parse_database_name(self.name)?;
        GetDatabaseRequest::try_new(tenant_id, database_name).map_err(|e| e.to_string())
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
        let database_name = resolve_database_name_typed(self.database_name)?;
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
        let database_name = resolve_database_name_typed(self.database_name)?;
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
        let database_name = resolve_database_name_typed(self.database_name)?;
        CountCollectionsRequest::try_new(tenant_id, database_name).map_err(|e| e.to_string())
    }
}

#[derive(Debug, Deserialize)]
struct EmbeddedUpdateCollectionPayload {
    #[serde(default)]
    database_name: Option<String>,
    collection_id: String,
    #[serde(default)]
    new_name: Option<String>,
    #[serde(default)]
    new_metadata: Option<UpdateMetadata>,
}

impl EmbeddedUpdateCollectionPayload {
    fn into_request(self) -> Result<UpdateCollectionRequest, String> {
        let database_name = self.database_name.map(parse_database_name).transpose()?;
        let collection_id = CollectionUuid::from_str(&self.collection_id)
            .map_err(|e| format!("invalid collection_id: {e}"))?;
        if self.new_name.is_none() && self.new_metadata.is_none() {
            return Err("at least one of new_name or new_metadata is required".to_string());
        }
        // TODO: Reject null-valued metadata entries at the shim boundary for defense in depth
        // across non-Go bindings. The Go API validates and rejects these before FFI.
        let new_metadata = self
            .new_metadata
            .map(CollectionMetadataUpdate::UpdateMetadata);
        UpdateCollectionRequest::try_new(
            database_name,
            collection_id,
            self.new_name,
            new_metadata,
            None,
        )
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
        let database_name = resolve_database_name(self.database_name);
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
        let database_name = resolve_database_name(self.database_name);
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
        let database_name = resolve_database_name(self.database_name);
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
        let database_name = resolve_database_name(self.database_name);
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
    metadatas: Option<Vec<Option<UpdateMetadata>>>,
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
        let database_name = resolve_database_name(self.database_name);
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
            self.metadatas,
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
    metadatas: Option<Vec<Option<UpdateMetadata>>>,
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
        let database_name = resolve_database_name(self.database_name);

        UpsertCollectionRecordsRequest::try_new(
            tenant_id,
            database_name,
            collection_id,
            self.ids,
            self.embeddings,
            self.documents,
            self.uris,
            self.metadatas,
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
        let database_name = resolve_database_name(self.database_name);
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

fn json_to_c_string_ptr_with_context<T: serde::Serialize>(value: &T, context: &str) -> *mut c_char {
    let ptr = json_to_c_string_ptr(value);
    if ptr.is_null() {
        if let Some(last_error) = last_error_message() {
            set_last_error(&format!("{context}: {last_error}"));
        } else {
            set_last_error(context);
        }
    }
    ptr
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
    ffi_guard_ptr_mut!({
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
    })
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
    ffi_guard_ptr_mut!({
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
    })
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
    let persist_path = match CString::new(resolved_config.persist_path.clone()) {
        Ok(s) => s,
        Err(_) => {
            set_last_error("persist_path contains null byte");
            return ptr::null_mut();
        }
    };

    // Use unit type () which implements both AuthenticateAndAuthorize and QuotaEnforcer.
    let auth: Arc<dyn chroma_frontend::auth::AuthenticateAndAuthorize> = Arc::new(());
    let quota_enforcer: Arc<dyn chroma_frontend::quota::QuotaEnforcer> = Arc::new(());

    let mut server_task = runtime.spawn(async move {
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

    match runtime.block_on(async { timeout(Duration::from_millis(250), &mut server_task).await }) {
        Ok(Ok(())) => {
            set_last_error("server exited during startup");
            return ptr::null_mut();
        }
        Ok(Err(join_err)) => {
            set_last_error(&format!("server panicked during startup: {join_err}"));
            return ptr::null_mut();
        }
        Err(_) => {
            // Timed out waiting for early task termination; assume startup is healthy.
        }
    }

    let handle = Box::new(ServerHandle {
        _runtime: runtime,
        shutdown_tx: Some(shutdown_tx),
        port,
        listen_address,
        persist_path,
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
    ffi_guard_minus_one!({
        if handle.is_null() {
            return -1;
        }
        let server = &*(handle as *const ServerHandle);
        server.port as i32
    })
}

/// Get the listen address the server is configured with.
/// Returns NULL on invalid handle.
/// The returned string is valid until the handle is freed.
///
/// # Safety
/// `handle` must be a valid handle from `chroma_server_start*` or NULL.
#[no_mangle]
pub unsafe extern "C" fn chroma_server_address(handle: *mut c_void) -> *const c_char {
    ffi_guard_ptr_const!({
        if handle.is_null() {
            return ptr::null();
        }
        let server = &*(handle as *const ServerHandle);
        server.listen_address.as_ptr()
    })
}

/// Get the effective persist path used by the server runtime.
/// Returns NULL on invalid handle.
/// The returned string is valid until the handle is freed.
///
/// # Safety
/// `handle` must be a valid handle from `chroma_server_start*` or NULL.
#[no_mangle]
pub unsafe extern "C" fn chroma_server_persist_path(handle: *mut c_void) -> *const c_char {
    ffi_guard_ptr_const!({
        if handle.is_null() {
            return ptr::null();
        }
        let server = &*(handle as *const ServerHandle);
        server.persist_path.as_ptr()
    })
}

/// Stop the server gracefully.
/// Returns SUCCESS on success, error code on failure.
///
/// # Safety
/// `handle` must be a valid handle from `chroma_server_start*` or NULL.
#[no_mangle]
pub unsafe extern "C" fn chroma_server_stop(handle: *mut c_void) -> i32 {
    ffi_guard_code!({
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
    })
}

/// Free the server handle.
/// Must be called after the server is stopped.
///
/// # Safety
/// `handle` must be a valid handle from `chroma_server_start*` or NULL.
/// Must not be called more than once for the same handle.
#[no_mangle]
pub unsafe extern "C" fn chroma_server_free(handle: *mut c_void) {
    ffi_guard_unit!({
        if !handle.is_null() {
            let _ = Box::from_raw(handle as *mut ServerHandle);
        }
    })
}

/// Start embedded (in-process) mode from a config file path.
/// Returns a handle on success, NULL on failure.
///
/// # Safety
/// `config_path` must be a valid null-terminated C string or NULL.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_start(config_path: *const c_char) -> *mut c_void {
    ffi_guard_ptr_mut!({
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
    })
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
    ffi_guard_ptr_mut!({
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
    })
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
    let persist_path = match CString::new(resolved_config.persist_path.clone()) {
        Ok(s) => s,
        Err(_) => {
            set_last_error("persist_path contains null byte");
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
        registry,
        persist_path,
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
    ffi_guard_unit!({
        if !handle.is_null() {
            let _ = Box::from_raw(handle as *mut EmbeddedHandle);
        }
    })
}

/// Get the effective persist path used by embedded runtime.
/// Returns NULL on invalid handle.
/// The returned string is valid until the handle is freed.
///
/// # Safety
/// `handle` must be a valid handle from `chroma_embedded_start*` or NULL.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_persist_path(handle: *mut c_void) -> *const c_char {
    ffi_guard_ptr_const!({
        if handle.is_null() {
            return ptr::null();
        }
        let embedded = &*(handle as *const EmbeddedHandle);
        embedded.persist_path.as_ptr()
    })
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
    ffi_guard_code!({
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
    })
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
    ffi_guard_code!({
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
    })
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
    ffi_guard_code!({
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
    })
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
    ffi_guard_ptr_mut!({
        if handle.is_null() {
            set_last_error("handle is null");
            return ptr::null_mut();
        }

        let payload: EmbeddedGetTenantPayload =
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
            Err(e) => {
                set_last_error(&format!("embedded frontend lock poisoned: {e}"));
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
    })
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
    ffi_guard_code!({
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
    })
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
    ffi_guard_code!({
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
    })
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
    ffi_guard_ptr_mut!({
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
    })
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
    ffi_guard_ptr_mut!({
        if handle.is_null() {
            set_last_error("handle is null");
            return ptr::null_mut();
        }

        let payload: EmbeddedGetDatabasePayload =
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
    })
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
    ffi_guard_code!({
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
    })
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
    ffi_guard_ptr_mut!({
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
    })
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
    ffi_guard_ptr_mut!({
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
    })
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
    ffi_guard_code!({
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
    })
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
    ffi_guard_code!({
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
    })
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
    ffi_guard_code!({
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
    })
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
    ffi_guard_ptr_mut!({
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
    })
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
    ffi_guard_code!({
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
    })
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
    ffi_guard_ptr_mut!({
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
    })
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
    ffi_guard_code!({
        if handle.is_null() {
            set_last_error("handle is null");
            return ERROR_INVALID_HANDLE;
        }

        let payload: EmbeddedUpdatePayload = match parse_json_request(request_json, "request_json")
        {
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
    })
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
    ffi_guard_code!({
        if handle.is_null() {
            set_last_error("handle is null");
            return ERROR_INVALID_HANDLE;
        }

        let payload: EmbeddedUpsertPayload = match parse_json_request(request_json, "request_json")
        {
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
    })
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
    ffi_guard_code!({
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
    })
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
    ffi_guard_ptr_mut!({
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
    })
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
    ffi_guard_code!({
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
    })
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
    ffi_guard_ptr_mut!({
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
    })
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
    ffi_guard_ptr_mut!({
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

        let (database_name, collection_id) = match payload.into_request() {
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
            .block_on(async { frontend.indexing_status(database_name, collection_id).await })
        {
            Ok(response) => response,
            Err(e) => {
                set_last_error(&format!("indexing status failed: {e}"));
                return ptr::null_mut();
            }
        };

        json_to_c_string_ptr(&response)
    })
}

/// Get healthcheck status in embedded mode.
/// Returns a JSON-serialized healthcheck response on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_healthcheck(handle: *mut c_void) -> *mut c_char {
    ffi_guard_ptr_mut!({
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
    })
}

/// Rebuild vector index artifacts for a single collection in embedded mode.
/// Returns a JSON-serialized rebuild response on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
/// Returned pointer must be freed with `chroma_string_free`.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_rebuild_collection(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    ffi_guard_ptr_mut!({
        if handle.is_null() {
            set_last_error("handle is null");
            return ptr::null_mut();
        }

        let payload: EmbeddedRebuildCollectionPayload =
            match parse_json_request(request_json, "request_json") {
                Ok(payload) => payload,
                Err(e) => {
                    set_last_error(&e);
                    return ptr::null_mut();
                }
            };
        let parts = match payload.into_request() {
            Ok(parts) => parts,
            Err(e) => {
                set_last_error(&e);
                return ptr::null_mut();
            }
        };

        let embedded = &*(handle as *const EmbeddedHandle);
        let mut frontend = match embedded.frontend.lock() {
            Ok(frontend) => frontend,
            Err(e) => {
                set_last_error(&format!("embedded frontend lock poisoned: {e}"));
                return ptr::null_mut();
            }
        };

        let persist_path = match embedded.persist_path.as_c_str().to_str() {
            Ok(path) => PathBuf::from(path),
            Err(e) => {
                set_last_error(&format!("persist path is not valid UTF-8: {e}"));
                return ptr::null_mut();
            }
        };

        let response = match embedded.runtime.block_on(async {
            run_rebuild_collection(
                &mut frontend,
                &embedded.registry,
                &persist_path,
                parts.request,
                parts.precheck,
                parts.keep_backup,
            )
            .await
        }) {
            Ok(response) => response,
            Err(e) => {
                set_last_error(&format!("rebuild collection failed: {e}"));
                return ptr::null_mut();
            }
        };

        json_to_c_string_ptr_with_context(
            &response,
            "rebuild collection completed but failed to serialize response",
        )
    })
}

/// Run explicit compaction for a single collection in embedded mode.
/// Returns a JSON-serialized compaction response on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
/// Returned pointer must be freed with `chroma_string_free`.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_compact_collection(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    ffi_guard_ptr_mut!({
        if handle.is_null() {
            set_last_error("handle is null");
            return ptr::null_mut();
        }

        let payload: EmbeddedCompactCollectionPayload =
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

        let response = match embedded.runtime.block_on(async {
            let collection = frontend
                .get_collection(request)
                .await
                .map_err(|e| format!("get collection failed: {e}"))?;
            let target = compaction_target_from_parts(
                collection.collection_id,
                collection.name,
                collection.tenant,
                collection.database,
            )?;
            run_explicit_compaction(
                &mut frontend,
                &embedded.registry,
                vec![target],
                CompactionFailureMode::FailFast,
            )
            .await
        }) {
            Ok(response) => response,
            Err(e) => {
                set_last_error(&format!("compact collection failed: {e}"));
                return ptr::null_mut();
            }
        };

        json_to_c_string_ptr_with_context(
            &response,
            "compact collection completed but failed to serialize response",
        )
    })
}

/// Run explicit compaction for all collections in embedded mode.
/// Returns a JSON-serialized compaction response on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
/// Returned pointer must be freed with `chroma_string_free`.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_compact_all(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    ffi_guard_ptr_mut!({
        if handle.is_null() {
            set_last_error("handle is null");
            return ptr::null_mut();
        }

        let payload: EmbeddedCompactAllPayload =
            match parse_json_request(request_json, "request_json") {
                Ok(payload) => payload,
                Err(e) => {
                    set_last_error(&e);
                    return ptr::null_mut();
                }
            };
        let tenant_id = payload
            .tenant_id
            .unwrap_or_else(|| DEFAULT_TENANT.to_string());

        let embedded = &*(handle as *const EmbeddedHandle);
        let mut frontend = match embedded.frontend.lock() {
            Ok(frontend) => frontend,
            Err(_) => {
                set_last_error("embedded frontend lock poisoned");
                return ptr::null_mut();
            }
        };

        let response = match embedded.runtime.block_on(async {
            let mut database_names: Vec<String> = Vec::new();
            if let Some(database_name) = payload.database_name {
                database_names.push(database_name);
            } else {
                let page_size: u32 = 100;
                let mut offset: u32 = 0;
                loop {
                    let request =
                        ListDatabasesRequest::try_new(tenant_id.clone(), Some(page_size), offset)
                            .map_err(|e| e.to_string())?;
                    let databases = frontend
                        .list_databases(request)
                        .await
                        .map_err(|e| format!("list databases failed: {e}"))?;
                    if databases.is_empty() {
                        break;
                    }

                    let count = u32::try_from(databases.len()).unwrap_or(u32::MAX);
                    database_names.extend(databases.into_iter().map(|database| database.name));
                    if count < page_size {
                        break;
                    }
                    offset = offset.saturating_add(count);
                }
            }

            let mut targets = Vec::new();
            for database_name in database_names {
                let parsed_database_name = parse_database_name(database_name.clone())?;
                let page_size: u32 = 100;
                let mut offset: u32 = 0;
                loop {
                    let request = ListCollectionsRequest::try_new(
                        tenant_id.clone(),
                        parsed_database_name.clone(),
                        Some(page_size),
                        offset,
                    )
                    .map_err(|e| e.to_string())?;
                    let collections = frontend.list_collections(request).await.map_err(|e| {
                        format!(
                            "list collections failed for database {}: {e}",
                            database_name
                        )
                    })?;
                    if collections.is_empty() {
                        break;
                    }

                    let count = u32::try_from(collections.len()).unwrap_or(u32::MAX);
                    for collection in collections {
                        targets.push(compaction_target_from_parts(
                            collection.collection_id,
                            collection.name,
                            collection.tenant,
                            collection.database,
                        )?);
                    }
                    if count < page_size {
                        break;
                    }
                    offset = offset.saturating_add(count);
                }
            }

            run_explicit_compaction(
                &mut frontend,
                &embedded.registry,
                targets,
                CompactionFailureMode::ContinueOnError,
            )
            .await
        }) {
            Ok(response) => response,
            Err(e) => {
                set_last_error(&format!("compact all failed: {e}"));
                return ptr::null_mut();
            }
        };

        json_to_c_string_ptr_with_context(
            &response,
            "compact all completed but failed to serialize response",
        )
    })
}

/// Run explicit WAL prune for one collection in embedded mode.
/// Returns a JSON-serialized WAL prune response on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
/// Returned pointer must be freed with `chroma_string_free`.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_prune_wal_collection(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    ffi_guard_ptr_mut!({
        if handle.is_null() {
            set_last_error("handle is null");
            return ptr::null_mut();
        }

        let payload: EmbeddedPruneWalCollectionPayload =
            match parse_json_request(request_json, "request_json") {
                Ok(payload) => payload,
                Err(e) => {
                    set_last_error(&e);
                    return ptr::null_mut();
                }
            };
        let parts = match payload.into_request() {
            Ok(parts) => parts,
            Err(e) => {
                set_last_error(&e);
                return ptr::null_mut();
            }
        };

        let embedded = &*(handle as *const EmbeddedHandle);
        let mut frontend = match embedded.frontend.lock() {
            Ok(frontend) => frontend,
            Err(e) => {
                set_last_error(&format!("embedded frontend lock poisoned: {e}"));
                return ptr::null_mut();
            }
        };

        let response = match embedded.runtime.block_on(async {
            let collection = frontend
                .get_collection(parts.request)
                .await
                .map_err(|e| format!("get collection failed: {e}"))?;
            let target = WalPruneTarget {
                collection_id: collection.collection_id,
                name: collection.name,
                tenant_id: collection.tenant,
                database_name_raw: collection.database,
            };
            run_explicit_wal_prune(
                &mut frontend,
                &embedded.registry,
                vec![target],
                parts.options,
                WalPruneFailureMode::FailFast,
            )
            .await
        }) {
            Ok(response) => response,
            Err(e) => {
                set_last_error(&format!("prune wal collection failed: {e}"));
                return ptr::null_mut();
            }
        };

        json_to_c_string_ptr_with_context(
            &response,
            "prune wal collection completed but failed to serialize response",
        )
    })
}

/// Run explicit WAL prune for all collections in embedded mode.
/// Returns a JSON-serialized WAL prune response on success, NULL on failure.
///
/// # Safety
/// `handle` must be a valid embedded handle.
/// `request_json` must be a valid null-terminated JSON string.
/// Returned pointer must be freed with `chroma_string_free`.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_prune_wal_all(
    handle: *mut c_void,
    request_json: *const c_char,
) -> *mut c_char {
    ffi_guard_ptr_mut!({
        if handle.is_null() {
            set_last_error("handle is null");
            return ptr::null_mut();
        }

        let payload: EmbeddedPruneWalAllPayload =
            match parse_json_request(request_json, "request_json") {
                Ok(payload) => payload,
                Err(e) => {
                    set_last_error(&e);
                    return ptr::null_mut();
                }
            };
        let tenant_id = payload
            .tenant_id
            .clone()
            .unwrap_or_else(|| DEFAULT_TENANT.to_string());
        let database_name = payload.database_name.clone();
        let options = match payload.into_options() {
            Ok(options) => options,
            Err(e) => {
                set_last_error(&e);
                return ptr::null_mut();
            }
        };

        let embedded = &*(handle as *const EmbeddedHandle);
        let mut frontend = match embedded.frontend.lock() {
            Ok(frontend) => frontend,
            Err(e) => {
                set_last_error(&format!("embedded frontend lock poisoned: {e}"));
                return ptr::null_mut();
            }
        };

        let response = match embedded.runtime.block_on(async {
            let targets = list_wal_prune_targets(&mut frontend, tenant_id, database_name).await?;
            run_explicit_wal_prune(
                &mut frontend,
                &embedded.registry,
                targets,
                options,
                WalPruneFailureMode::ContinueOnError,
            )
            .await
        }) {
            Ok(response) => response,
            Err(e) => {
                set_last_error(&format!("prune wal all failed: {e}"));
                return ptr::null_mut();
            }
        };

        json_to_c_string_ptr_with_context(
            &response,
            "prune wal all completed but failed to serialize response",
        )
    })
}

/// Reset local state in embedded mode.
/// Returns SUCCESS on success.
///
/// # Safety
/// `handle` must be a valid embedded handle.
#[no_mangle]
pub unsafe extern "C" fn chroma_embedded_reset(handle: *mut c_void) -> i32 {
    ffi_guard_code!({
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
    })
}

/// Free a heap-allocated C string returned by this library.
///
/// # Safety
/// `s` must be a pointer previously returned by this library via `CString::into_raw`.
#[no_mangle]
pub unsafe extern "C" fn chroma_string_free(s: *mut c_char) {
    ffi_guard_unit!({
        if !s.is_null() {
            let _ = CString::from_raw(s);
        }
    })
}

/// Get the last error message.
/// Returns NULL if no error.
///
/// # Safety
/// The returned string must be freed with `chroma_string_free`.
#[no_mangle]
pub unsafe extern "C" fn chroma_get_last_error() -> *const c_char {
    ffi_guard_ptr_const!({
        match last_error_cstring() {
            Some(s) => s.into_raw() as *const c_char,
            None => ptr::null(),
        }
    })
}

/// Get the version string.
///
/// # Safety
/// Returns a pointer to a static string, always safe to call.
#[no_mangle]
pub unsafe extern "C" fn chroma_version() -> *const c_char {
    ffi_guard_ptr_const!({
        static VERSION: &[u8] = b"0.2.0\0";
        VERSION.as_ptr() as *const c_char
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use proptest::prelude::*;
    use serde_json::json;
    use tempfile::tempdir;

    struct ScriptedSwapFsOps {
        rename_results: std::sync::Mutex<Vec<std::io::Result<()>>>,
        remove_results: std::sync::Mutex<Vec<std::io::Result<()>>>,
        rename_calls: std::sync::Mutex<Vec<(PathBuf, PathBuf)>>,
    }

    impl ScriptedSwapFsOps {
        fn new(
            rename_results: Vec<std::io::Result<()>>,
            remove_results: Vec<std::io::Result<()>>,
        ) -> Self {
            Self {
                rename_results: std::sync::Mutex::new(rename_results),
                remove_results: std::sync::Mutex::new(remove_results),
                rename_calls: std::sync::Mutex::new(Vec::new()),
            }
        }
    }

    impl SwapFsOps for ScriptedSwapFsOps {
        fn rename(&self, from: &Path, to: &Path) -> std::io::Result<()> {
            self.rename_calls
                .lock()
                .expect("rename_calls lock should be available")
                .push((from.to_path_buf(), to.to_path_buf()));

            let mut results = self
                .rename_results
                .lock()
                .expect("rename_results lock should be available");
            if results.is_empty() {
                return Ok(());
            }
            results.remove(0)
        }

        fn remove_dir_all(&self, _path: &Path) -> std::io::Result<()> {
            let mut results = self
                .remove_results
                .lock()
                .expect("remove_results lock should be available");
            if results.is_empty() {
                return Ok(());
            }
            results.remove(0)
        }
    }

    fn last_error_string() -> Option<String> {
        LAST_ERROR
            .lock()
            .ok()
            .and_then(|slot| slot.as_ref().cloned())
    }

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

    #[test]
    fn test_add_payload_accepts_metadata_arrays() {
        let payload: EmbeddedAddPayload = serde_json::from_value(json!({
            "collection_id": "00000000-0000-0000-0000-000000000001",
            "ids": ["doc-1"],
            "embeddings": [[0.1, 0.2, 0.3]],
            "metadatas": [{
                "tags": ["alpha", "beta"],
                "scores": [1.1, 2.2],
                "flags": [true, false],
                "levels": [1, 2]
            }]
        }))
        .expect("payload should deserialize");

        let request = payload.into_request().expect("request should build");
        assert!(request.metadatas.is_some());
    }

    #[test]
    fn test_create_collection_payload_accepts_metadata_and_configuration() {
        let payload: EmbeddedCreateCollectionPayload = serde_json::from_value(json!({
            "name": "test_collection",
            "metadata": {
                "owner": "qa",
                "priority": 3
            },
            "configuration": {
                "hnsw": {
                    "space": "cosine",
                    "ef_construction": 200
                }
            },
            "get_or_create": true
        }))
        .expect("payload should deserialize");

        let request = payload.into_request().expect("request should build");
        assert!(request.metadata.is_some());
        assert!(request.configuration.is_some());
        assert!(request.get_or_create);
    }

    #[test]
    fn test_create_collection_payload_accepts_configuration_json_alias() {
        let payload: EmbeddedCreateCollectionPayload = serde_json::from_value(json!({
            "name": "test_collection_alias",
            "configuration_json": {
                "hnsw": {
                    "space": "ip"
                }
            }
        }))
        .expect("payload should deserialize");

        let request = payload.into_request().expect("request should build");
        assert!(request.configuration.is_some());
    }

    #[test]
    fn test_create_collection_payload_rejects_invalid_configuration() {
        let payload: EmbeddedCreateCollectionPayload = serde_json::from_value(json!({
            "name": "test_collection_invalid_config",
            "configuration": {
                "hnsw": {
                    "space": "invalid_space"
                }
            }
        }))
        .expect("payload should deserialize");

        let err = payload
            .into_request()
            .expect_err("payload should fail when configuration payload is invalid");
        assert!(err.contains("invalid configuration payload"));
    }

    #[test]
    fn test_create_collection_payload_accepts_schema() {
        let schema = serde_json::to_value(Schema::default()).expect("schema should serialize");
        let payload: EmbeddedCreateCollectionPayload = serde_json::from_value(json!({
            "name": "test_collection_schema",
            "schema": schema
        }))
        .expect("payload should deserialize");

        let request = payload.into_request().expect("request should build");
        assert!(request.schema.is_some());
    }

    #[test]
    fn test_create_collection_payload_rejects_invalid_schema() {
        let payload: EmbeddedCreateCollectionPayload = serde_json::from_value(json!({
            "name": "test_collection_invalid_schema",
            "schema": {
                "defaults": "invalid"
            }
        }))
        .expect("payload should deserialize");

        let err = payload
            .into_request()
            .expect_err("payload should fail when schema payload is invalid");
        assert!(err.contains("invalid schema payload"));
    }

    #[test]
    fn test_update_collection_payload_accepts_new_metadata_without_name() {
        // This validates raw shim payload parsing only.
        // The Go API rejects null-valued entries in new_metadata before reaching FFI.
        let payload: EmbeddedUpdateCollectionPayload = serde_json::from_value(json!({
            "collection_id": "00000000-0000-0000-0000-000000000001",
            "new_metadata": {
                "owner": "qa",
                "version": 2,
                "deprecated": null
            }
        }))
        .expect("payload should deserialize");

        let request = payload.into_request().expect("request should build");
        assert!(request.new_name.is_none());
        assert!(matches!(
            request.new_metadata,
            Some(CollectionMetadataUpdate::UpdateMetadata(_))
        ));
    }

    #[test]
    fn test_update_collection_payload_accepts_name_and_new_metadata() {
        let payload: EmbeddedUpdateCollectionPayload = serde_json::from_value(json!({
            "collection_id": "00000000-0000-0000-0000-000000000001",
            "new_name": "test_collection_v2",
            "new_metadata": {
                "owner": "qa"
            }
        }))
        .expect("payload should deserialize");

        let request = payload.into_request().expect("request should build");
        assert_eq!(request.new_name.as_deref(), Some("test_collection_v2"));
        match request.new_metadata {
            Some(CollectionMetadataUpdate::UpdateMetadata(metadata)) => {
                assert!(metadata.contains_key("owner"));
            }
            _ => panic!("expected update metadata variant"),
        }
    }

    #[test]
    fn test_update_collection_payload_requires_name_or_metadata() {
        let payload: EmbeddedUpdateCollectionPayload = serde_json::from_value(json!({
            "collection_id": "00000000-0000-0000-0000-000000000001"
        }))
        .expect("payload should deserialize");

        let err = payload
            .into_request()
            .expect_err("payload should fail without update fields");
        assert!(err.contains("at least one of new_name or new_metadata is required"));
    }

    #[test]
    fn test_update_collection_payload_rejects_invalid_new_metadata() {
        let payload: EmbeddedUpdateCollectionPayload = serde_json::from_value(json!({
            "collection_id": "00000000-0000-0000-0000-000000000001",
            "new_metadata": {
                "$reserved": "bad"
            }
        }))
        .expect("payload should deserialize");

        let err = payload
            .into_request()
            .expect_err("payload should fail for invalid metadata key");
        assert!(err.to_lowercase().contains("metadata"));
    }

    #[test]
    fn test_update_payload_accepts_metadata_arrays() {
        let payload: EmbeddedUpdatePayload = serde_json::from_value(json!({
            "collection_id": "00000000-0000-0000-0000-000000000001",
            "ids": ["doc-1"],
            "metadatas": [{
                "tags": ["updated", "stable"],
                "scores": [3.3, 4.4]
            }]
        }))
        .expect("payload should deserialize");

        let request = payload.into_request().expect("request should build");
        assert!(request.metadatas.is_some());
    }

    #[test]
    fn test_rebuild_payload_defaults() {
        let payload: EmbeddedRebuildCollectionPayload = serde_json::from_value(json!({
            "name": "docs"
        }))
        .expect("payload should deserialize");

        let parts = payload.into_request().expect("request should build");
        assert_eq!(parts.request.collection_name, "docs");
        assert_eq!(parts.request.tenant_id, DEFAULT_TENANT);
        assert_eq!(parts.request.database_name.as_ref(), DEFAULT_DATABASE);
        assert!(!parts.precheck);
        assert!(parts.keep_backup);
    }

    #[test]
    fn test_rebuild_payload_precheck_and_keep_backup_false() {
        let payload: EmbeddedRebuildCollectionPayload = serde_json::from_value(json!({
            "name": "docs",
            "precheck": true,
            "keep_backup": false,
            "database_name": "prod"
        }))
        .expect("payload should deserialize");

        let parts = payload.into_request().expect("request should build");
        assert!(parts.precheck);
        assert!(!parts.keep_backup);
    }

    #[test]
    fn test_rebuild_payload_rejects_short_database_name() {
        let payload: EmbeddedRebuildCollectionPayload = serde_json::from_value(json!({
            "name": "docs",
            "database_name": "ab"
        }))
        .expect("payload should deserialize");

        match payload.into_request() {
            Ok(_) => panic!("payload should fail for invalid database"),
            Err(err) => assert!(err.contains("database_name must be at least 3 characters")),
        }
    }

    #[test]
    fn test_wal_prune_collection_payload_allows_dry_run_without_policy() {
        let payload: EmbeddedPruneWalCollectionPayload = serde_json::from_value(json!({
            "name": "docs",
            "dry_run": true
        }))
        .expect("payload should deserialize");

        let parts = payload.into_request().expect("request should build");
        assert!(parts.options.dry_run);
        assert!(!parts.options.policies.has_policy());
    }

    #[test]
    fn test_wal_prune_collection_payload_requires_policy_when_mutating() {
        let payload: EmbeddedPruneWalCollectionPayload = serde_json::from_value(json!({
            "name": "docs"
        }))
        .expect("payload should deserialize");

        match payload.into_request() {
            Ok(_) => panic!("request should fail without policy"),
            Err(err) => assert!(err.contains("at least one WAL prune policy is required")),
        }
    }

    #[test]
    fn test_wal_prune_payload_rejects_invalid_watermark() {
        let payload: EmbeddedPruneWalCollectionPayload = serde_json::from_value(json!({
            "name": "docs",
            "max_bytes": 1024,
            "watermark_high_bytes": 100,
            "watermark_low_bytes": 200
        }))
        .expect("payload should deserialize");

        match payload.into_request() {
            Ok(_) => panic!("request should fail for invalid watermark"),
            Err(err) => assert!(err.contains("wal prune watermark")),
        }
    }

    #[test]
    fn test_wal_prune_all_payload_rejects_short_tenant() {
        let payload: EmbeddedPruneWalAllPayload = serde_json::from_value(json!({
            "tenant_id": "ab",
            "dry_run": true
        }))
        .expect("payload should deserialize");

        let err = payload
            .into_options()
            .expect_err("request should fail for short tenant");
        assert!(err.contains("tenant_id must be at least 3 characters"));
    }

    #[test]
    fn test_wal_prune_prefix_count_uses_and_semantics() {
        let rows = vec![
            WalPruneCandidateRow {
                seq_id: 1,
                created_at_secs: 100,
                estimated_bytes: 50,
            },
            WalPruneCandidateRow {
                seq_id: 2,
                created_at_secs: 200,
                estimated_bytes: 50,
            },
            WalPruneCandidateRow {
                seq_id: 3,
                created_at_secs: 300,
                estimated_bytes: 50,
            },
        ];
        let policies = WalPrunePolicyPayload {
            max_age_seconds: Some(150),
            max_bytes: Some(90),
            watermark_high_bytes: Some(120),
            watermark_low_bytes: Some(60),
        };

        // age prefix at now=300 is 1 row (created_at <= 150), max_bytes prefix is 2 rows,
        // watermark prefix is 2 rows; AND semantics keeps min(prefixes)=1.
        let prefix = wal_prune_prefix_count(&rows, &policies, 300);
        assert_eq!(prefix, 1);
    }

    #[test]
    fn test_hnsw_id_map_round_trip_validates() {
        let temp = tempdir().expect("tempdir should be created");
        let metadata_path = temp.path().join(HNSW_METADATA_FILENAME);

        let mut id_map = HnswIdMap {
            dimensionality: Some(3),
            total_elements_added: 0,
            max_seq_id: Some(22),
            id_to_label: HashMap::new(),
            label_to_id: HashMap::new(),
            id_to_seq_id: HashMap::new(),
        };
        id_map
            .insert_record("doc-1".to_string(), 1, Some(11))
            .expect("first insert should succeed");
        id_map
            .insert_record("doc-2".to_string(), 2, None)
            .expect("second insert should succeed");

        write_hnsw_id_map(&metadata_path, &id_map).expect("metadata write should succeed");
        let decoded = load_hnsw_id_map(&metadata_path).expect("metadata load should succeed");

        assert_eq!(decoded.dimensionality, Some(3));
        assert_eq!(decoded.total_elements_added, 2);
        assert_eq!(decoded.max_seq_id, Some(22));
        assert_eq!(decoded.id_to_label.get("doc-1"), Some(&1));
        assert_eq!(decoded.id_to_label.get("doc-2"), Some(&2));
        assert_eq!(
            decoded.label_to_id.get(&1).map(String::as_str),
            Some("doc-1")
        );
        assert_eq!(
            decoded.label_to_id.get(&2).map(String::as_str),
            Some("doc-2")
        );
        assert_eq!(decoded.id_to_seq_id.get("doc-1"), Some(&11));
    }

    #[test]
    fn test_load_hnsw_id_map_rejects_non_bijective_mapping() {
        let temp = tempdir().expect("tempdir should be created");
        let metadata_path = temp.path().join(HNSW_METADATA_FILENAME);

        let mut id_to_label = HashMap::new();
        id_to_label.insert("doc-1".to_string(), 1);
        let mut label_to_id = HashMap::new();
        label_to_id.insert(2, "doc-1".to_string());
        let invalid = HnswIdMap {
            dimensionality: Some(3),
            total_elements_added: 2,
            max_seq_id: None,
            id_to_label,
            label_to_id,
            id_to_seq_id: HashMap::new(),
        };

        write_hnsw_id_map(&metadata_path, &invalid).expect("metadata write should succeed");
        let err =
            load_hnsw_id_map(&metadata_path).expect_err("invalid metadata should be rejected");
        assert!(err.contains("invalid hnsw metadata"));
        assert!(err.contains("non-bijective") || err.contains("missing reverse mapping"));
    }

    #[test]
    fn test_load_hnsw_id_map_rejects_seq_id_for_unknown_id() {
        let temp = tempdir().expect("tempdir should be created");
        let metadata_path = temp.path().join(HNSW_METADATA_FILENAME);

        let mut id_to_label = HashMap::new();
        id_to_label.insert("doc-1".to_string(), 1);
        let mut label_to_id = HashMap::new();
        label_to_id.insert(1, "doc-1".to_string());
        let mut id_to_seq_id = HashMap::new();
        id_to_seq_id.insert("doc-2".to_string(), 22);
        let invalid = HnswIdMap {
            dimensionality: Some(3),
            total_elements_added: 1,
            max_seq_id: Some(22),
            id_to_label,
            label_to_id,
            id_to_seq_id,
        };

        write_hnsw_id_map(&metadata_path, &invalid).expect("metadata write should succeed");
        let err =
            load_hnsw_id_map(&metadata_path).expect_err("invalid metadata should be rejected");
        assert!(err.contains("invalid hnsw metadata"));
        assert!(err.contains("seq-id mapping references unknown id"));
    }

    #[test]
    fn test_load_hnsw_id_map_rejects_max_seq_id_below_observed() {
        let temp = tempdir().expect("tempdir should be created");
        let metadata_path = temp.path().join(HNSW_METADATA_FILENAME);

        let mut id_to_label = HashMap::new();
        id_to_label.insert("doc-1".to_string(), 1);
        let mut label_to_id = HashMap::new();
        label_to_id.insert(1, "doc-1".to_string());
        let mut id_to_seq_id = HashMap::new();
        id_to_seq_id.insert("doc-1".to_string(), 22);
        let invalid = HnswIdMap {
            dimensionality: Some(3),
            total_elements_added: 1,
            max_seq_id: Some(21),
            id_to_label,
            label_to_id,
            id_to_seq_id,
        };

        write_hnsw_id_map(&metadata_path, &invalid).expect("metadata write should succeed");
        let err =
            load_hnsw_id_map(&metadata_path).expect_err("invalid metadata should be rejected");
        assert!(err.contains("invalid hnsw metadata"));
        assert!(err.contains("max_seq_id"));
    }

    #[test]
    fn test_temp_rebuild_dir_guard_removes_directory_on_drop() {
        let temp = tempdir().expect("tempdir should be created");
        let rebuilt_path = temp.path().join("tmp.rebuild.guard");
        fs::create_dir_all(&rebuilt_path).expect("rebuilt path should be created");
        fs::write(rebuilt_path.join("segment.bin"), b"test")
            .expect("rebuilt artifact should be written");

        {
            let _guard = TempRebuildDirGuard::new(rebuilt_path.clone());
        }

        assert!(
            !rebuilt_path.exists(),
            "temporary rebuilt directory should be removed by drop guard"
        );
    }

    #[test]
    fn test_rebuild_required_capacity_uses_resize_factor_headroom() {
        let target = rebuild_required_capacity(10_000, 1.2);
        assert_eq!(target, 12_000);
    }

    #[test]
    fn test_rebuild_required_capacity_enforces_minimums() {
        let low_scale = rebuild_required_capacity(80, 0.5);
        assert_eq!(low_scale, 100);

        let needs_plus_one = rebuild_required_capacity(220, 1.0);
        assert_eq!(needs_plus_one, 221);
    }

    #[test]
    fn test_swap_rebuilt_index_dir_reports_rollback_success_context() {
        let temp = tempdir().expect("tempdir should be created");
        let source = temp.path().join("source");
        fs::create_dir(&source).expect("source dir should be created");

        let rebuilt_missing = temp.path().join("missing_rebuild");
        let err = swap_rebuilt_index_dir(&source, &rebuilt_missing, true)
            .expect_err("swap should fail when rebuilt path is missing");

        assert!(err.contains("failed to activate rebuilt index"));
        assert!(err.contains("rollback succeeded"));
    }

    #[test]
    fn test_swap_rebuilt_index_dir_with_ops_rolls_back_on_activation_failure() {
        let source = Path::new("/tmp/source");
        let rebuilt = Path::new("/tmp/rebuilt");
        let ops = ScriptedSwapFsOps::new(
            vec![
                Ok(()),
                Err(std::io::Error::new(
                    std::io::ErrorKind::Other,
                    "activation failed",
                )),
                Ok(()),
            ],
            Vec::new(),
        );

        let err = swap_rebuilt_index_dir_with_ops(
            source,
            rebuilt,
            true,
            "fixed_suffix".to_string(),
            &ops,
        )
        .expect_err("swap should fail and rollback should be attempted");

        assert!(err.contains("rollback succeeded"));

        let calls = ops
            .rename_calls
            .lock()
            .expect("rename_calls lock should be available");
        assert_eq!(calls.len(), 3);
        assert_eq!(calls[0].0, source);
        assert!(calls[0]
            .1
            .to_string_lossy()
            .contains("source_backup_fixed_suffix"));
        assert_eq!(calls[1].0, rebuilt);
        assert_eq!(calls[1].1, source);
        assert_eq!(calls[2].0, calls[0].1);
        assert_eq!(calls[2].1, source);
    }

    #[test]
    fn test_swap_rebuilt_index_dir_with_ops_reports_rollback_failure() {
        let source = Path::new("/tmp/source");
        let rebuilt = Path::new("/tmp/rebuilt");
        let ops = ScriptedSwapFsOps::new(
            vec![
                Ok(()),
                Err(std::io::Error::new(
                    std::io::ErrorKind::Other,
                    "activation failed",
                )),
                Err(std::io::Error::new(
                    std::io::ErrorKind::PermissionDenied,
                    "rollback denied",
                )),
            ],
            Vec::new(),
        );

        let err = swap_rebuilt_index_dir_with_ops(
            source,
            rebuilt,
            false,
            "fixed_suffix".to_string(),
            &ops,
        )
        .expect_err("swap should fail and rollback failure should be reported");

        assert!(err.contains("rollback failed"));

        let calls = ops
            .rename_calls
            .lock()
            .expect("rename_calls lock should be available");
        assert_eq!(calls.len(), 3);
        assert!(calls[0]
            .1
            .to_string_lossy()
            .contains("source_rollback_fixed_suffix"));
    }

    #[test]
    fn test_format_swap_activation_error_includes_rollback_failure() {
        let source = Path::new("/tmp/source");
        let rebuilt = Path::new("/tmp/rebuilt");
        let swap_err = std::io::Error::new(std::io::ErrorKind::Other, "swap failed");
        let rollback_err =
            std::io::Error::new(std::io::ErrorKind::PermissionDenied, "rollback failed");

        let message = format_swap_activation_error(rebuilt, source, &swap_err, Some(&rollback_err));
        assert!(message.contains("failed to activate rebuilt index"));
        assert!(message.contains("rollback failed"));
        assert!(message.contains("swap failed"));
    }

    #[test]
    fn test_ffi_panic_boundary_returns_default_and_sets_last_error() {
        let value = with_ffi_panic_boundary(123, || -> i32 {
            panic!("boom");
        });

        assert_eq!(value, 123);
        let err = last_error_string().expect("last error should be populated");
        assert!(err.contains("panic across FFI boundary"));
        assert!(err.contains("boom"));
    }

    #[test]
    fn test_ffi_panic_boundary_returns_success_value() {
        let value = with_ffi_panic_boundary(123, || 7);
        assert_eq!(value, 7);
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
