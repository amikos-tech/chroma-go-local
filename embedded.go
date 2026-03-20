package chroma

import "github.com/amikos-tech/chroma-go-local/internal/runtime"

const (
	DefaultTenantID    = runtime.DefaultTenantID
	DefaultDatabase    = runtime.DefaultDatabase
	DefaultEmbeddedDir = runtime.DefaultEmbeddedDir
)

type Embedded = runtime.Embedded
type StartEmbeddedConfig = runtime.StartEmbeddedConfig
type EmbeddedConfig = runtime.EmbeddedConfig
type EmbeddedOption = runtime.EmbeddedOption
type EmbeddedCreateCollectionRequest = runtime.EmbeddedCreateCollectionRequest
type EmbeddedCollection = runtime.EmbeddedCollection
type EmbeddedDatabase = runtime.EmbeddedDatabase
type EmbeddedTenant = runtime.EmbeddedTenant
type EmbeddedCreateTenantRequest = runtime.EmbeddedCreateTenantRequest
type EmbeddedGetTenantRequest = runtime.EmbeddedGetTenantRequest
type EmbeddedUpdateTenantRequest = runtime.EmbeddedUpdateTenantRequest
type EmbeddedCreateDatabaseRequest = runtime.EmbeddedCreateDatabaseRequest
type EmbeddedListDatabasesRequest = runtime.EmbeddedListDatabasesRequest
type EmbeddedGetDatabaseRequest = runtime.EmbeddedGetDatabaseRequest
type EmbeddedDeleteDatabaseRequest = runtime.EmbeddedDeleteDatabaseRequest
type EmbeddedListCollectionsRequest = runtime.EmbeddedListCollectionsRequest
type EmbeddedGetCollectionRequest = runtime.EmbeddedGetCollectionRequest
type EmbeddedCountCollectionsRequest = runtime.EmbeddedCountCollectionsRequest
type EmbeddedUpdateCollectionRequest = runtime.EmbeddedUpdateCollectionRequest
type EmbeddedDeleteCollectionRequest = runtime.EmbeddedDeleteCollectionRequest
type EmbeddedForkCollectionRequest = runtime.EmbeddedForkCollectionRequest
type EmbeddedAddRequest = runtime.EmbeddedAddRequest
type EmbeddedQueryRequest = runtime.EmbeddedQueryRequest
type EmbeddedQueryResponse = runtime.EmbeddedQueryResponse
type EmbeddedCountRecordsRequest = runtime.EmbeddedCountRecordsRequest
type EmbeddedGetRecordsRequest = runtime.EmbeddedGetRecordsRequest
type EmbeddedGetRecordsResponse = runtime.EmbeddedGetRecordsResponse
type EmbeddedUpdateRecordsRequest = runtime.EmbeddedUpdateRecordsRequest
type EmbeddedUpsertRecordsRequest = runtime.EmbeddedUpsertRecordsRequest
type EmbeddedDeleteRecordsRequest = runtime.EmbeddedDeleteRecordsRequest
type EmbeddedDeleteRecordsResponse = runtime.EmbeddedDeleteRecordsResponse
type EmbeddedIndexingStatusRequest = runtime.EmbeddedIndexingStatusRequest
type EmbeddedIndexingStatusResponse = runtime.EmbeddedIndexingStatusResponse
type EmbeddedHealthCheckResponse = runtime.EmbeddedHealthCheckResponse

func DefaultEmbeddedConfig() *EmbeddedConfig {
	return runtime.DefaultEmbeddedConfig()
}

func NewEmbedded(opts ...EmbeddedOption) (*Embedded, error) {
	return runtime.NewEmbedded(opts...)
}

func StartEmbedded(config StartEmbeddedConfig) (*Embedded, error) {
	return runtime.StartEmbedded(config)
}

func WithEmbeddedPersistPath(path string) EmbeddedOption {
	return runtime.WithEmbeddedPersistPath(path)
}

func WithEmbeddedSQLiteFilename(filename string) EmbeddedOption {
	return runtime.WithEmbeddedSQLiteFilename(filename)
}

func WithEmbeddedAllowReset(allow bool) EmbeddedOption {
	return runtime.WithEmbeddedAllowReset(allow)
}

func WithEmbeddedRawYAML(yaml string) EmbeddedOption {
	return runtime.WithEmbeddedRawYAML(yaml)
}
