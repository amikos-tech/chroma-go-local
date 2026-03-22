package chroma_test

import (
	"net"
	"testing"
	"time"

	chroma "github.com/amikos-tech/chroma-go-local"
	"github.com/stretchr/testify/require"
)

// -- Compile-time API surface gate --
// Every exported type, function, constant, and variable must appear below.
// If a symbol is removed or renamed, this file fails to compile.

// chroma.go types
var _ chroma.Server
var _ chroma.StartServerConfig

// config.go types
var _ chroma.ServerConfig
var _ chroma.ServerOption

// embedded.go types
var _ chroma.Embedded
var _ chroma.StartEmbeddedConfig
var _ chroma.EmbeddedConfig
var _ chroma.EmbeddedOption
var _ chroma.EmbeddedCreateCollectionRequest
var _ chroma.EmbeddedCollection
var _ chroma.EmbeddedDatabase
var _ chroma.EmbeddedTenant
var _ chroma.EmbeddedCreateTenantRequest
var _ chroma.EmbeddedGetTenantRequest
var _ chroma.EmbeddedUpdateTenantRequest
var _ chroma.EmbeddedCreateDatabaseRequest
var _ chroma.EmbeddedListDatabasesRequest
var _ chroma.EmbeddedGetDatabaseRequest
var _ chroma.EmbeddedDeleteDatabaseRequest
var _ chroma.EmbeddedListCollectionsRequest
var _ chroma.EmbeddedGetCollectionRequest
var _ chroma.EmbeddedCountCollectionsRequest
var _ chroma.EmbeddedUpdateCollectionRequest
var _ chroma.EmbeddedDeleteCollectionRequest
var _ chroma.EmbeddedForkCollectionRequest
var _ chroma.EmbeddedAddRequest
var _ chroma.EmbeddedQueryRequest
var _ chroma.EmbeddedQueryResponse
var _ chroma.EmbeddedCountRecordsRequest
var _ chroma.EmbeddedGetRecordsRequest
var _ chroma.EmbeddedGetRecordsResponse
var _ chroma.EmbeddedUpdateRecordsRequest
var _ chroma.EmbeddedUpsertRecordsRequest
var _ chroma.EmbeddedDeleteRecordsRequest
var _ chroma.EmbeddedDeleteRecordsResponse
var _ chroma.EmbeddedIndexingStatusRequest
var _ chroma.EmbeddedIndexingStatusResponse
var _ chroma.EmbeddedHealthCheckResponse

// backup.go types
var _ chroma.BackupMode
var _ chroma.BackupOptions
var _ chroma.ServerBackupOptions
var _ chroma.EmbeddedBackupOptions
var _ chroma.BackupOption
var _ chroma.BackupFileMetadata
var _ chroma.BackupManifest

// rebuild.go types
var _ chroma.RebuildCollectionResult
var _ chroma.RebuildCollectionOption

// compaction.go types
var _ chroma.CompactCollectionRequest
var _ chroma.CompactAllRequest
var _ chroma.CompactionCollectionResult
var _ chroma.CompactionResult

// wal_prune.go types
var _ chroma.WALPruneCollectionResult
var _ chroma.WALPruneResult
var _ chroma.WALPruneOption

// chroma.go functions
var _ = chroma.Init
var _ = chroma.StartServer
var _ = chroma.Version
var _ = chroma.VersionWithError

// config.go functions
var _ = chroma.DefaultServerConfig
var _ = chroma.NewServer
var _ = chroma.WithPort
var _ = chroma.WithListenAddress
var _ = chroma.WithMaxPayloadSize
var _ = chroma.WithCORSAllowOrigins
var _ = chroma.WithPersistPath
var _ = chroma.WithSQLiteFilename
var _ = chroma.WithAllowReset
var _ = chroma.WithOpenTelemetry
var _ = chroma.WithRawYAML

// embedded.go functions
var _ = chroma.DefaultEmbeddedConfig
var _ = chroma.NewEmbedded
var _ = chroma.StartEmbedded
var _ = chroma.WithEmbeddedPersistPath
var _ = chroma.WithEmbeddedSQLiteFilename
var _ = chroma.WithEmbeddedAllowReset
var _ = chroma.WithEmbeddedRawYAML

// backup.go functions
var _ = chroma.WithDestination
var _ = chroma.WithIncludeMetadata
var _ = chroma.WithLeaveStopped
var _ = chroma.WithLeaveClosed

// rebuild.go functions
var _ = chroma.WithRebuildTenantID
var _ = chroma.WithRebuildDatabaseName
var _ = chroma.WithRebuildPrecheck
var _ = chroma.WithRebuildKeepBackup

// wal_prune.go functions
var _ = chroma.WithWALPruneTenantID
var _ = chroma.WithWALPruneDatabaseName
var _ = chroma.WithWALPruneDryRun
var _ = chroma.WithWALPruneVacuum
var _ = chroma.WithWALPruneMaxAge
var _ = chroma.WithWALPruneMaxBytes
var _ = chroma.WithWALPruneWatermark

// errors.go constants
var _ = chroma.Success
var _ = chroma.ErrNullInput
var _ = chroma.ErrInvalidUTF8
var _ = chroma.ErrConfigParse
var _ = chroma.ErrServerStart
var _ = chroma.ErrInvalidHandle
var _ = chroma.ErrRuntimeCreate
var _ = chroma.ErrAlreadyStopped
var _ = chroma.ErrOperation

// embedded.go constants
var _ = chroma.DefaultTenantID
var _ = chroma.DefaultDatabase
var _ = chroma.DefaultEmbeddedDir

// backup.go constants
var _ = chroma.BackupModeServer
var _ = chroma.BackupModeEmbedded

// errors.go sentinel errors
var _ = chroma.ErrNullPointer
var _ = chroma.ErrLibraryNotLoaded
var _ = chroma.ErrServerNotStarted
var _ = chroma.ErrServerAlreadyStop
var _ = chroma.ErrEmbeddedNotStarted

// -- Behavioral smoke tests --

func TestInit(t *testing.T) {
	err := chroma.Init("")
	require.NoError(t, err)
}

func TestVersion(t *testing.T) {
	err := chroma.Init("")
	require.NoError(t, err)
	v := chroma.Version()
	require.NotEmpty(t, v)
}

func TestVersionWithError(t *testing.T) {
	err := chroma.Init("")
	require.NoError(t, err)
	v, err := chroma.VersionWithError()
	require.NoError(t, err)
	require.NotEmpty(t, v)
}

func TestDefaultServerConfig(t *testing.T) {
	err := chroma.Init("")
	require.NoError(t, err)
	cfg := chroma.DefaultServerConfig()
	require.NotNil(t, cfg)
	require.Greater(t, cfg.Port, 0)
}

func TestNewServer(t *testing.T) {
	err := chroma.Init("")
	require.NoError(t, err)
	port := freePort(t)
	server, err := chroma.NewServer(chroma.WithPort(port))
	require.NoError(t, err)
	require.NotNil(t, server)
	defer func() { _ = server.Stop() }()
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

func TestDefaultEmbeddedConfig(t *testing.T) {
	err := chroma.Init("")
	require.NoError(t, err)
	cfg := chroma.DefaultEmbeddedConfig()
	require.NotNil(t, cfg)
}

func TestBackupOptionBuilders(t *testing.T) {
	require.NotNil(t, chroma.WithDestination("/tmp/test"))
	require.NotNil(t, chroma.WithIncludeMetadata())
	require.NotNil(t, chroma.WithLeaveStopped())
	require.NotNil(t, chroma.WithLeaveClosed())
}

func TestRebuildOptionBuilders(t *testing.T) {
	require.NotNil(t, chroma.WithRebuildTenantID("t"))
	require.NotNil(t, chroma.WithRebuildDatabaseName("d"))
	require.NotNil(t, chroma.WithRebuildPrecheck())
	require.NotNil(t, chroma.WithRebuildKeepBackup(true))
}

func TestWALPruneOptionBuilders(t *testing.T) {
	require.NotNil(t, chroma.WithWALPruneTenantID("t"))
	require.NotNil(t, chroma.WithWALPruneDatabaseName("d"))
	require.NotNil(t, chroma.WithWALPruneDryRun())
	require.NotNil(t, chroma.WithWALPruneVacuum())
	require.NotNil(t, chroma.WithWALPruneMaxAge(time.Hour))
	require.NotNil(t, chroma.WithWALPruneMaxBytes(1024))
	require.NotNil(t, chroma.WithWALPruneWatermark(2048, 1024))
}
