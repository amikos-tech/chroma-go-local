package chroma

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

const (
	// Default tenant and database used by Chroma local mode.
	DefaultTenantID    = "default_tenant"
	DefaultDatabase    = "default_database"
	DefaultEmbeddedDir = "./chroma"
)

// Embedded represents an in-process Chroma frontend (no HTTP server).
type Embedded struct {
	handle uintptr
}

// StartEmbeddedConfig contains configuration options for starting embedded mode.
type StartEmbeddedConfig struct {
	ConfigPath   string // Path to YAML config file
	ConfigString string // YAML config string (used if ConfigPath is empty)
}

// EmbeddedConfig holds simple embedded configuration defaults.
type EmbeddedConfig struct {
	PersistPath    string
	SQLiteFilename string
	AllowReset     bool

	rawYAML string
}

// EmbeddedOption configures EmbeddedConfig.
type EmbeddedOption func(*EmbeddedConfig)

// DefaultEmbeddedConfig returns a default embedded config.
func DefaultEmbeddedConfig() *EmbeddedConfig {
	return &EmbeddedConfig{
		PersistPath:    DefaultEmbeddedDir,
		SQLiteFilename: "chroma.sqlite3",
		AllowReset:     false,
	}
}

// WithEmbeddedPersistPath sets the embedded persistence directory.
func WithEmbeddedPersistPath(path string) EmbeddedOption {
	return func(c *EmbeddedConfig) {
		c.PersistPath = path
	}
}

// WithEmbeddedSQLiteFilename sets the SQLite filename.
func WithEmbeddedSQLiteFilename(filename string) EmbeddedOption {
	return func(c *EmbeddedConfig) {
		c.SQLiteFilename = filename
	}
}

// WithEmbeddedAllowReset enables reset in embedded mode.
func WithEmbeddedAllowReset(allow bool) EmbeddedOption {
	return func(c *EmbeddedConfig) {
		c.AllowReset = allow
	}
}

// WithEmbeddedRawYAML uses a raw YAML config (overrides other options).
func WithEmbeddedRawYAML(yaml string) EmbeddedOption {
	return func(c *EmbeddedConfig) {
		c.rawYAML = yaml
	}
}

func (c *EmbeddedConfig) toYAML() string {
	if c.rawYAML != "" {
		return c.rawYAML
	}

	var b strings.Builder
	fmt.Fprintf(&b, "persist_path: %q\n", c.PersistPath)
	fmt.Fprintf(&b, "sqlite_filename: %q\n", c.SQLiteFilename)
	fmt.Fprintf(&b, "allow_reset: %t\n", c.AllowReset)
	return b.String()
}

// EmbeddedCreateCollectionRequest creates a collection in embedded mode.
type EmbeddedCreateCollectionRequest struct {
	Name         string `json:"name"`
	TenantID     string `json:"tenant_id,omitempty"`
	DatabaseName string `json:"database_name,omitempty"`
	GetOrCreate  bool   `json:"get_or_create,omitempty"`
}

// EmbeddedCollection is a compact view of a created collection.
type EmbeddedCollection struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Tenant   string `json:"tenant"`
	Database string `json:"database"`
}

// EmbeddedDatabase is a compact view of a database.
type EmbeddedDatabase struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Tenant string `json:"tenant"`
}

// EmbeddedTenant is a compact view of a tenant.
type EmbeddedTenant struct {
	Name         string  `json:"name"`
	ResourceName *string `json:"resource_name,omitempty"`
}

// EmbeddedCreateTenantRequest creates a tenant in embedded mode.
type EmbeddedCreateTenantRequest struct {
	Name string `json:"name"`
}

// EmbeddedGetTenantRequest gets a tenant by name.
type EmbeddedGetTenantRequest struct {
	Name string `json:"name"`
}

// EmbeddedUpdateTenantRequest updates tenant properties.
type EmbeddedUpdateTenantRequest struct {
	TenantID     string `json:"tenant_id"`
	ResourceName string `json:"resource_name"`
}

// EmbeddedCreateDatabaseRequest creates a database in embedded mode.
type EmbeddedCreateDatabaseRequest struct {
	Name     string `json:"name"`
	TenantID string `json:"tenant_id,omitempty"`
}

// EmbeddedListDatabasesRequest lists databases in embedded mode.
type EmbeddedListDatabasesRequest struct {
	TenantID string `json:"tenant_id,omitempty"`
	Limit    uint32 `json:"limit,omitempty"`
	Offset   uint32 `json:"offset,omitempty"`
}

// EmbeddedGetDatabaseRequest gets a single database.
type EmbeddedGetDatabaseRequest struct {
	Name     string `json:"name"`
	TenantID string `json:"tenant_id,omitempty"`
}

// EmbeddedDeleteDatabaseRequest deletes a single database.
type EmbeddedDeleteDatabaseRequest struct {
	Name     string `json:"name"`
	TenantID string `json:"tenant_id,omitempty"`
}

// EmbeddedListCollectionsRequest lists collections for a database.
type EmbeddedListCollectionsRequest struct {
	TenantID     string `json:"tenant_id,omitempty"`
	DatabaseName string `json:"database_name,omitempty"`
	Limit        uint32 `json:"limit,omitempty"`
	Offset       uint32 `json:"offset,omitempty"`
}

// EmbeddedGetCollectionRequest gets a collection by name.
type EmbeddedGetCollectionRequest struct {
	Name         string `json:"name"`
	TenantID     string `json:"tenant_id,omitempty"`
	DatabaseName string `json:"database_name,omitempty"`
}

// EmbeddedCountCollectionsRequest counts collections for a database.
type EmbeddedCountCollectionsRequest struct {
	TenantID     string `json:"tenant_id,omitempty"`
	DatabaseName string `json:"database_name,omitempty"`
}

// EmbeddedUpdateCollectionRequest renames a collection.
type EmbeddedUpdateCollectionRequest struct {
	CollectionID string `json:"collection_id"`
	NewName      string `json:"new_name"`
}

// EmbeddedDeleteCollectionRequest deletes a collection by name.
type EmbeddedDeleteCollectionRequest struct {
	Name         string `json:"name"`
	TenantID     string `json:"tenant_id,omitempty"`
	DatabaseName string `json:"database_name,omitempty"`
}

// EmbeddedForkCollectionRequest forks a source collection into a new target collection.
type EmbeddedForkCollectionRequest struct {
	SourceCollectionID   string `json:"source_collection_id"`
	TargetCollectionName string `json:"target_collection_name"`
	TenantID             string `json:"tenant_id,omitempty"`
	DatabaseName         string `json:"database_name,omitempty"`
}

// EmbeddedAddRequest adds records to a collection.
type EmbeddedAddRequest struct {
	CollectionID string      `json:"collection_id"`
	IDs          []string    `json:"ids"`
	Embeddings   [][]float32 `json:"embeddings"`
	Documents    []string    `json:"documents,omitempty"`
	URIs         []string    `json:"uris,omitempty"`
	TenantID     string      `json:"tenant_id,omitempty"`
	DatabaseName string      `json:"database_name,omitempty"`
}

// EmbeddedQueryRequest queries vectors from a collection.
type EmbeddedQueryRequest struct {
	CollectionID    string         `json:"collection_id"`
	QueryEmbeddings [][]float32    `json:"query_embeddings"`
	NResults        uint32         `json:"n_results,omitempty"`
	IDs             []string       `json:"ids,omitempty"`
	Where           map[string]any `json:"where,omitempty"`
	WhereDocument   map[string]any `json:"where_document,omitempty"`
	Include         []string       `json:"include,omitempty"`
	TenantID        string         `json:"tenant_id,omitempty"`
	DatabaseName    string         `json:"database_name,omitempty"`
}

// EmbeddedQueryResponse contains top match ids per query embedding.
type EmbeddedQueryResponse struct {
	IDs [][]string `json:"ids"`
}

// EmbeddedCountRecordsRequest counts records in a collection.
type EmbeddedCountRecordsRequest struct {
	CollectionID string `json:"collection_id"`
	TenantID     string `json:"tenant_id,omitempty"`
	DatabaseName string `json:"database_name,omitempty"`
}

// EmbeddedGetRecordsRequest fetches records by ids, filters, or pagination.
type EmbeddedGetRecordsRequest struct {
	CollectionID  string         `json:"collection_id"`
	IDs           []string       `json:"ids,omitempty"`
	Where         map[string]any `json:"where,omitempty"`
	WhereDocument map[string]any `json:"where_document,omitempty"`
	Limit         uint32         `json:"limit,omitempty"`
	Offset        uint32         `json:"offset,omitempty"`
	Include       []string       `json:"include,omitempty"`
	TenantID      string         `json:"tenant_id,omitempty"`
	DatabaseName  string         `json:"database_name,omitempty"`
}

// EmbeddedGetRecordsResponse contains fetched record fields.
type EmbeddedGetRecordsResponse struct {
	IDs        []string         `json:"ids"`
	Embeddings [][]float32      `json:"embeddings,omitempty"`
	Documents  []*string        `json:"documents,omitempty"`
	URIs       []*string        `json:"uris,omitempty"`
	Metadatas  []map[string]any `json:"metadatas,omitempty"`
	Include    []string         `json:"include,omitempty"`
}

// EmbeddedUpdateRecordsRequest updates existing records by id.
type EmbeddedUpdateRecordsRequest struct {
	CollectionID string      `json:"collection_id"`
	IDs          []string    `json:"ids"`
	Embeddings   [][]float32 `json:"embeddings,omitempty"`
	Documents    []string    `json:"documents,omitempty"`
	URIs         []string    `json:"uris,omitempty"`
	TenantID     string      `json:"tenant_id,omitempty"`
	DatabaseName string      `json:"database_name,omitempty"`
}

// EmbeddedUpsertRecordsRequest upserts records by id.
type EmbeddedUpsertRecordsRequest struct {
	CollectionID string      `json:"collection_id"`
	IDs          []string    `json:"ids"`
	Embeddings   [][]float32 `json:"embeddings"`
	Documents    []string    `json:"documents,omitempty"`
	URIs         []string    `json:"uris,omitempty"`
	TenantID     string      `json:"tenant_id,omitempty"`
	DatabaseName string      `json:"database_name,omitempty"`
}

// EmbeddedDeleteRecordsRequest deletes records by ids and/or filters.
type EmbeddedDeleteRecordsRequest struct {
	CollectionID  string         `json:"collection_id"`
	IDs           []string       `json:"ids,omitempty"`
	Where         map[string]any `json:"where,omitempty"`
	WhereDocument map[string]any `json:"where_document,omitempty"`
	TenantID      string         `json:"tenant_id,omitempty"`
	DatabaseName  string         `json:"database_name,omitempty"`
}

// EmbeddedIndexingStatusRequest gets indexing progress for a collection.
type EmbeddedIndexingStatusRequest struct {
	CollectionID string `json:"collection_id"`
}

// EmbeddedIndexingStatusResponse describes indexing progress in local mode.
type EmbeddedIndexingStatusResponse struct {
	OpIndexingProgress float32 `json:"op_indexing_progress"`
	NumUnindexedOps    uint64  `json:"num_unindexed_ops"`
	NumIndexedOps      uint64  `json:"num_indexed_ops"`
	TotalOps           uint64  `json:"total_ops"`
}

// EmbeddedHealthCheckResponse describes embedded readiness state.
type EmbeddedHealthCheckResponse struct {
	IsExecutorReady  bool `json:"is_executor_ready"`
	IsLogClientReady bool `json:"is_log_client_ready"`
}

// NewEmbedded starts embedded mode with builder options.
func NewEmbedded(opts ...EmbeddedOption) (*Embedded, error) {
	cfg := DefaultEmbeddedConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return StartEmbedded(StartEmbeddedConfig{ConfigString: cfg.toYAML()})
}

// StartEmbedded starts in-process embedded mode.
func StartEmbedded(config StartEmbeddedConfig) (*Embedded, error) {
	if libHandle == 0 {
		return nil, ErrLibraryNotLoaded
	}

	var handle uintptr
	switch {
	case config.ConfigPath != "":
		pathBytes := cStringFromGo(config.ConfigPath)
		handle = chromaEmbeddedStart(&pathBytes[0])
	case config.ConfigString != "":
		yamlBytes := cStringFromGo(config.ConfigString)
		handle = chromaEmbeddedStartFromString(&yamlBytes[0])
	default:
		return nil, errors.New("either ConfigPath or ConfigString must be provided")
	}

	if handle == 0 {
		return nil, errors.Wrap(ErrNullPointer, getLastError())
	}

	embedded := &Embedded{handle: handle}
	runtime.SetFinalizer(embedded, func(e *Embedded) {
		_ = e.Close()
	})
	return embedded, nil
}

// Heartbeat returns unix nanoseconds from in-process frontend heartbeat.
func (e *Embedded) Heartbeat() (uint64, error) {
	if e == nil || e.handle == 0 {
		return 0, ErrEmbeddedNotStarted
	}

	var heartbeat uint64
	rc := chromaEmbeddedHeartbeat(e.handle, &heartbeat)
	if rc != Success {
		return 0, errorFromCode(rc, getLastError())
	}
	return heartbeat, nil
}

// MaxBatchSize returns the configured max batch size.
func (e *Embedded) MaxBatchSize() (uint32, error) {
	if e == nil || e.handle == 0 {
		return 0, ErrEmbeddedNotStarted
	}

	var maxBatchSize uint32
	rc := chromaEmbeddedGetMaxBatchSize(e.handle, &maxBatchSize)
	if rc != Success {
		return 0, errorFromCode(rc, getLastError())
	}
	return maxBatchSize, nil
}

// CreateTenant creates a tenant.
func (e *Embedded) CreateTenant(request EmbeddedCreateTenantRequest) error {
	if e == nil || e.handle == 0 {
		return ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.Name) == "" {
		return errors.New("name is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return err
	}

	rc := chromaEmbeddedCreateTenant(e.handle, &requestBytes[0])
	if rc != Success {
		return errorFromCode(rc, getLastError())
	}
	return nil
}

// GetTenant gets a tenant by name.
func (e *Embedded) GetTenant(request EmbeddedGetTenantRequest) (*EmbeddedTenant, error) {
	if e == nil || e.handle == 0 {
		return nil, ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.Name) == "" {
		return nil, errors.New("name is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return nil, err
	}

	respPtr := chromaEmbeddedGetTenant(e.handle, &requestBytes[0])
	if respPtr == nil {
		return nil, errors.Wrap(ErrNullPointer, getLastError())
	}
	defer chromaStringFree(respPtr)

	var tenant EmbeddedTenant
	if err := json.Unmarshal([]byte(goStringFromPtr(respPtr)), &tenant); err != nil {
		return nil, errors.Wrap(err, "failed to decode get tenant response")
	}
	return &tenant, nil
}

// UpdateTenant updates tenant properties.
func (e *Embedded) UpdateTenant(request EmbeddedUpdateTenantRequest) error {
	if e == nil || e.handle == 0 {
		return ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.TenantID) == "" {
		return errors.New("tenant_id is required")
	}
	if strings.TrimSpace(request.ResourceName) == "" {
		return errors.New("resource_name is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return err
	}

	rc := chromaEmbeddedUpdateTenant(e.handle, &requestBytes[0])
	if rc != Success {
		return errorFromCode(rc, getLastError())
	}
	return nil
}

// Healthcheck returns local readiness of internal embedded components.
func (e *Embedded) Healthcheck() (*EmbeddedHealthCheckResponse, error) {
	if e == nil || e.handle == 0 {
		return nil, ErrEmbeddedNotStarted
	}

	respPtr := chromaEmbeddedHealthcheck(e.handle)
	if respPtr == nil {
		return nil, errors.Wrap(ErrNullPointer, getLastError())
	}
	defer chromaStringFree(respPtr)

	var response EmbeddedHealthCheckResponse
	if err := json.Unmarshal([]byte(goStringFromPtr(respPtr)), &response); err != nil {
		return nil, errors.Wrap(err, "failed to decode healthcheck response")
	}
	return &response, nil
}

// IndexingStatus reports indexing progress for a collection.
func (e *Embedded) IndexingStatus(request EmbeddedIndexingStatusRequest) (*EmbeddedIndexingStatusResponse, error) {
	if e == nil || e.handle == 0 {
		return nil, ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.CollectionID) == "" {
		return nil, errors.New("collection_id is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return nil, err
	}

	respPtr := chromaEmbeddedIndexingStatus(e.handle, &requestBytes[0])
	if respPtr == nil {
		return nil, errors.Wrap(ErrNullPointer, getLastError())
	}
	defer chromaStringFree(respPtr)

	var response EmbeddedIndexingStatusResponse
	if err := json.Unmarshal([]byte(goStringFromPtr(respPtr)), &response); err != nil {
		return nil, errors.Wrap(err, "failed to decode indexing status response")
	}
	return &response, nil
}

// Reset resets local state if allow_reset is enabled.
func (e *Embedded) Reset() error {
	if e == nil || e.handle == 0 {
		return ErrEmbeddedNotStarted
	}

	rc := chromaEmbeddedReset(e.handle)
	if rc != Success {
		return errorFromCode(rc, getLastError())
	}
	return nil
}

// CreateDatabase creates a database.
func (e *Embedded) CreateDatabase(request EmbeddedCreateDatabaseRequest) error {
	if e == nil || e.handle == 0 {
		return ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.Name) == "" {
		return errors.New("name is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return err
	}

	rc := chromaEmbeddedCreateDatabase(e.handle, &requestBytes[0])
	if rc != Success {
		return errorFromCode(rc, getLastError())
	}
	return nil
}

// ListDatabases lists databases.
func (e *Embedded) ListDatabases(request EmbeddedListDatabasesRequest) ([]EmbeddedDatabase, error) {
	if e == nil || e.handle == 0 {
		return nil, ErrEmbeddedNotStarted
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return nil, err
	}

	respPtr := chromaEmbeddedListDatabases(e.handle, &requestBytes[0])
	if respPtr == nil {
		return nil, errors.Wrap(ErrNullPointer, getLastError())
	}
	defer chromaStringFree(respPtr)

	var databases []EmbeddedDatabase
	if err := json.Unmarshal([]byte(goStringFromPtr(respPtr)), &databases); err != nil {
		return nil, errors.Wrap(err, "failed to decode list databases response")
	}
	return databases, nil
}

// GetDatabase gets a database by name.
func (e *Embedded) GetDatabase(request EmbeddedGetDatabaseRequest) (*EmbeddedDatabase, error) {
	if e == nil || e.handle == 0 {
		return nil, ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.Name) == "" {
		return nil, errors.New("name is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return nil, err
	}

	respPtr := chromaEmbeddedGetDatabase(e.handle, &requestBytes[0])
	if respPtr == nil {
		return nil, errors.Wrap(ErrNullPointer, getLastError())
	}
	defer chromaStringFree(respPtr)

	var database EmbeddedDatabase
	if err := json.Unmarshal([]byte(goStringFromPtr(respPtr)), &database); err != nil {
		return nil, errors.Wrap(err, "failed to decode get database response")
	}
	return &database, nil
}

// DeleteDatabase deletes a database by name.
func (e *Embedded) DeleteDatabase(request EmbeddedDeleteDatabaseRequest) error {
	if e == nil || e.handle == 0 {
		return ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.Name) == "" {
		return errors.New("name is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return err
	}

	rc := chromaEmbeddedDeleteDatabase(e.handle, &requestBytes[0])
	if rc != Success {
		return errorFromCode(rc, getLastError())
	}
	return nil
}

// ListCollections lists collections for a database.
func (e *Embedded) ListCollections(request EmbeddedListCollectionsRequest) ([]EmbeddedCollection, error) {
	if e == nil || e.handle == 0 {
		return nil, ErrEmbeddedNotStarted
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return nil, err
	}

	respPtr := chromaEmbeddedListCollections(e.handle, &requestBytes[0])
	if respPtr == nil {
		return nil, errors.Wrap(ErrNullPointer, getLastError())
	}
	defer chromaStringFree(respPtr)

	var collections []EmbeddedCollection
	if err := json.Unmarshal([]byte(goStringFromPtr(respPtr)), &collections); err != nil {
		return nil, errors.Wrap(err, "failed to decode list collections response")
	}
	return collections, nil
}

// GetCollection gets a collection by name.
func (e *Embedded) GetCollection(request EmbeddedGetCollectionRequest) (*EmbeddedCollection, error) {
	if e == nil || e.handle == 0 {
		return nil, ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.Name) == "" {
		return nil, errors.New("name is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return nil, err
	}

	respPtr := chromaEmbeddedGetCollection(e.handle, &requestBytes[0])
	if respPtr == nil {
		return nil, errors.Wrap(ErrNullPointer, getLastError())
	}
	defer chromaStringFree(respPtr)

	var collection EmbeddedCollection
	if err := json.Unmarshal([]byte(goStringFromPtr(respPtr)), &collection); err != nil {
		return nil, errors.Wrap(err, "failed to decode get collection response")
	}
	return &collection, nil
}

// CountCollections counts collections for a database.
func (e *Embedded) CountCollections(request EmbeddedCountCollectionsRequest) (uint32, error) {
	if e == nil || e.handle == 0 {
		return 0, ErrEmbeddedNotStarted
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return 0, err
	}

	var count uint32
	rc := chromaEmbeddedCountCollections(e.handle, &requestBytes[0], &count)
	if rc != Success {
		return 0, errorFromCode(rc, getLastError())
	}
	return count, nil
}

// UpdateCollection updates collection properties (currently supports rename).
func (e *Embedded) UpdateCollection(request EmbeddedUpdateCollectionRequest) error {
	if e == nil || e.handle == 0 {
		return ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.CollectionID) == "" {
		return errors.New("collection_id is required")
	}
	if strings.TrimSpace(request.NewName) == "" {
		return errors.New("new_name is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return err
	}

	rc := chromaEmbeddedUpdateCollection(e.handle, &requestBytes[0])
	if rc != Success {
		return errorFromCode(rc, getLastError())
	}
	return nil
}

// DeleteCollection deletes a collection by name.
func (e *Embedded) DeleteCollection(request EmbeddedDeleteCollectionRequest) error {
	if e == nil || e.handle == 0 {
		return ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.Name) == "" {
		return errors.New("name is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return err
	}

	rc := chromaEmbeddedDeleteCollection(e.handle, &requestBytes[0])
	if rc != Success {
		return errorFromCode(rc, getLastError())
	}
	return nil
}

// ForkCollection forks a source collection into a target collection.
func (e *Embedded) ForkCollection(request EmbeddedForkCollectionRequest) (*EmbeddedCollection, error) {
	if e == nil || e.handle == 0 {
		return nil, ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.SourceCollectionID) == "" {
		return nil, errors.New("source_collection_id is required")
	}
	if strings.TrimSpace(request.TargetCollectionName) == "" {
		return nil, errors.New("target_collection_name is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return nil, err
	}

	respPtr := chromaEmbeddedForkCollection(e.handle, &requestBytes[0])
	if respPtr == nil {
		return nil, errors.Wrap(ErrNullPointer, getLastError())
	}
	defer chromaStringFree(respPtr)

	var collection EmbeddedCollection
	if err := json.Unmarshal([]byte(goStringFromPtr(respPtr)), &collection); err != nil {
		return nil, errors.Wrap(err, "failed to decode fork collection response")
	}
	return &collection, nil
}

// CountRecords counts records in a collection.
func (e *Embedded) CountRecords(request EmbeddedCountRecordsRequest) (uint32, error) {
	if e == nil || e.handle == 0 {
		return 0, ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.CollectionID) == "" {
		return 0, errors.New("collection_id is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return 0, err
	}

	var count uint32
	rc := chromaEmbeddedCount(e.handle, &requestBytes[0], &count)
	if rc != Success {
		return 0, errorFromCode(rc, getLastError())
	}
	return count, nil
}

// GetRecords fetches records from a collection.
func (e *Embedded) GetRecords(request EmbeddedGetRecordsRequest) (*EmbeddedGetRecordsResponse, error) {
	if e == nil || e.handle == 0 {
		return nil, ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.CollectionID) == "" {
		return nil, errors.New("collection_id is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return nil, err
	}

	respPtr := chromaEmbeddedGet(e.handle, &requestBytes[0])
	if respPtr == nil {
		return nil, errors.Wrap(ErrNullPointer, getLastError())
	}
	defer chromaStringFree(respPtr)

	var response EmbeddedGetRecordsResponse
	if err := json.Unmarshal([]byte(goStringFromPtr(respPtr)), &response); err != nil {
		return nil, errors.Wrap(err, "failed to decode get records response")
	}
	return &response, nil
}

// UpdateRecords updates existing records by id.
func (e *Embedded) UpdateRecords(request EmbeddedUpdateRecordsRequest) error {
	if e == nil || e.handle == 0 {
		return ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.CollectionID) == "" {
		return errors.New("collection_id is required")
	}
	if len(request.IDs) == 0 {
		return errors.New("ids must not be empty")
	}
	if len(request.Embeddings) > 0 && len(request.Embeddings) != len(request.IDs) {
		return errors.New("embeddings must have same length as ids when provided")
	}
	if len(request.Documents) > 0 && len(request.Documents) != len(request.IDs) {
		return errors.New("documents must have same length as ids when provided")
	}
	if len(request.URIs) > 0 && len(request.URIs) != len(request.IDs) {
		return errors.New("uris must have same length as ids when provided")
	}
	if len(request.Embeddings) == 0 && len(request.Documents) == 0 && len(request.URIs) == 0 {
		return errors.New("at least one of embeddings, documents, or uris must be provided")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return err
	}

	rc := chromaEmbeddedUpdate(e.handle, &requestBytes[0])
	if rc != Success {
		return errorFromCode(rc, getLastError())
	}
	return nil
}

// UpsertRecords upserts records by id.
func (e *Embedded) UpsertRecords(request EmbeddedUpsertRecordsRequest) error {
	if e == nil || e.handle == 0 {
		return ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.CollectionID) == "" {
		return errors.New("collection_id is required")
	}
	if len(request.IDs) == 0 {
		return errors.New("ids must not be empty")
	}
	if len(request.Embeddings) == 0 {
		return errors.New("embeddings must not be empty")
	}
	if len(request.IDs) != len(request.Embeddings) {
		return errors.New("ids and embeddings must have same length")
	}
	if len(request.Documents) > 0 && len(request.Documents) != len(request.IDs) {
		return errors.New("documents must have same length as ids when provided")
	}
	if len(request.URIs) > 0 && len(request.URIs) != len(request.IDs) {
		return errors.New("uris must have same length as ids when provided")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return err
	}

	rc := chromaEmbeddedUpsert(e.handle, &requestBytes[0])
	if rc != Success {
		return errorFromCode(rc, getLastError())
	}
	return nil
}

// DeleteRecords deletes records by ids and/or filters.
func (e *Embedded) DeleteRecords(request EmbeddedDeleteRecordsRequest) error {
	if e == nil || e.handle == 0 {
		return ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.CollectionID) == "" {
		return errors.New("collection_id is required")
	}
	if len(request.IDs) == 0 && len(request.Where) == 0 && len(request.WhereDocument) == 0 {
		return errors.New("at least one of ids, where, or where_document must be provided")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return err
	}

	rc := chromaEmbeddedDeleteRecords(e.handle, &requestBytes[0])
	if rc != Success {
		return errorFromCode(rc, getLastError())
	}
	return nil
}

// CreateCollection creates a collection and returns a compact response object.
func (e *Embedded) CreateCollection(request EmbeddedCreateCollectionRequest) (*EmbeddedCollection, error) {
	if e == nil || e.handle == 0 {
		return nil, ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.Name) == "" {
		return nil, errors.New("name is required")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return nil, err
	}

	respPtr := chromaEmbeddedCreateCollection(e.handle, &requestBytes[0])
	if respPtr == nil {
		return nil, errors.Wrap(ErrNullPointer, getLastError())
	}
	defer chromaStringFree(respPtr)

	var collection EmbeddedCollection
	if err := json.Unmarshal([]byte(goStringFromPtr(respPtr)), &collection); err != nil {
		return nil, errors.Wrap(err, "failed to decode collection response")
	}
	return &collection, nil
}

// Add adds records into an existing collection.
func (e *Embedded) Add(request EmbeddedAddRequest) error {
	if e == nil || e.handle == 0 {
		return ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.CollectionID) == "" {
		return errors.New("collection_id is required")
	}
	if len(request.IDs) == 0 {
		return errors.New("ids must not be empty")
	}
	if len(request.Embeddings) == 0 {
		return errors.New("embeddings must not be empty")
	}
	if len(request.IDs) != len(request.Embeddings) {
		return errors.New("ids and embeddings must have same length")
	}
	if len(request.Documents) > 0 && len(request.Documents) != len(request.IDs) {
		return errors.New("documents must have same length as ids when provided")
	}
	if len(request.URIs) > 0 && len(request.URIs) != len(request.IDs) {
		return errors.New("uris must have same length as ids when provided")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return err
	}

	rc := chromaEmbeddedAdd(e.handle, &requestBytes[0])
	if rc != Success {
		return errorFromCode(rc, getLastError())
	}
	return nil
}

// Query runs nearest-neighbor search against a collection.
func (e *Embedded) Query(request EmbeddedQueryRequest) (*EmbeddedQueryResponse, error) {
	if e == nil || e.handle == 0 {
		return nil, ErrEmbeddedNotStarted
	}
	if strings.TrimSpace(request.CollectionID) == "" {
		return nil, errors.New("collection_id is required")
	}
	if len(request.QueryEmbeddings) == 0 {
		return nil, errors.New("query_embeddings must not be empty")
	}

	requestBytes, err := marshalRequestJSON(request)
	if err != nil {
		return nil, err
	}

	respPtr := chromaEmbeddedQuery(e.handle, &requestBytes[0])
	if respPtr == nil {
		return nil, errors.Wrap(ErrNullPointer, getLastError())
	}
	defer chromaStringFree(respPtr)

	var response EmbeddedQueryResponse
	if err := json.Unmarshal([]byte(goStringFromPtr(respPtr)), &response); err != nil {
		return nil, errors.Wrap(err, "failed to decode query response")
	}
	return &response, nil
}

// Close releases embedded mode resources.
func (e *Embedded) Close() error {
	if e == nil || e.handle == 0 {
		return nil
	}

	chromaEmbeddedFree(e.handle)
	e.handle = 0
	return nil
}

func marshalRequestJSON(v any) ([]byte, error) {
	payload, err := json.Marshal(v)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}
	return append(payload, 0), nil
}
