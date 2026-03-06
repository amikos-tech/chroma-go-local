package chroma

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEmbeddedPruneCollectionWALDryRunNoMutation(t *testing.T) {
	embedded, _ := startTestEmbedded(t)
	databaseName, collectionName, collectionID := seedWALPruneCollection(t, embedded)

	countBefore, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collectionID,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)

	result, err := embedded.PruneCollectionWAL(
		collectionName,
		WithWALPruneDatabaseName(databaseName),
		WithWALPruneDryRun(),
		WithWALPruneVacuum(),
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.DryRun)
	require.True(t, result.VacuumRequested)
	require.False(t, result.VacuumExecuted)
	require.Equal(t, uint32(1), result.CollectionCount)
	require.Len(t, result.Collections, 1)
	require.Equal(t, collectionName, result.Collections[0].Name)

	countAfter, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collectionID,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.Equal(t, countBefore, countAfter)

	query, err := embedded.Query(EmbeddedQueryRequest{
		CollectionID:    collectionID,
		DatabaseName:    databaseName,
		QueryEmbeddings: [][]float32{{0.1, 0.2, 0.3}},
		NResults:        1,
	})
	require.NoError(t, err)
	require.Len(t, query.IDs, 1)
	require.Len(t, query.IDs[0], 1)
}

func TestEmbeddedPruneCollectionWALExecutionPreservesQueries(t *testing.T) {
	embedded, _ := startTestEmbedded(t)
	databaseName, collectionName, collectionID := seedWALPruneCollection(t, embedded)

	result, err := embedded.PruneCollectionWAL(
		collectionName,
		WithWALPruneDatabaseName(databaseName),
		WithWALPruneMaxBytes(0),
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.DryRun)
	require.Equal(t, uint32(1), result.CollectionCount)
	require.Len(t, result.Collections, 1)
	require.Equal(t, collectionName, result.Collections[0].Name)

	count, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collectionID,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.Equal(t, uint32(2), count)

	query, err := embedded.Query(EmbeddedQueryRequest{
		CollectionID:    collectionID,
		DatabaseName:    databaseName,
		QueryEmbeddings: [][]float32{{0.1, 0.2, 0.3}},
		NResults:        1,
	})
	require.NoError(t, err)
	require.Len(t, query.IDs, 1)
	require.Len(t, query.IDs[0], 1)
}

func TestEmbeddedPruneCollectionWALExecutionPrunesAndVacuums(t *testing.T) {
	embedded, _ := startTestEmbedded(t)
	databaseName, collectionName, collectionID := seedWALPruneCollection(t, embedded)

	dryRun, err := embedded.PruneCollectionWAL(
		collectionName,
		WithWALPruneDatabaseName(databaseName),
		WithWALPruneDryRun(),
		WithWALPruneMaxBytes(0),
	)
	require.NoError(t, err)
	require.NotNil(t, dryRun)
	if dryRun.CandidateCountTotal == 0 {
		t.Skip("no WAL prune candidates available in this runtime state")
	}

	result, err := embedded.PruneCollectionWAL(
		collectionName,
		WithWALPruneDatabaseName(databaseName),
		WithWALPruneMaxBytes(0),
		WithWALPruneVacuum(),
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.DryRun)
	require.True(t, result.VacuumRequested)
	if !result.VacuumExecuted {
		require.NotEmpty(t, result.Warning, "vacuum failure should be surfaced as warning when prune succeeds")
	}
	require.Greater(t, result.PrunedCountTotal, uint64(0), "expected prune execution to delete WAL rows")
	require.Greater(t, result.Collections[0].PrunedCount, uint64(0))

	count, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collectionID,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.Equal(t, uint32(2), count)
}

func TestEmbeddedPruneCollectionWALMaxAgePolicy(t *testing.T) {
	embedded, _ := startTestEmbedded(t)
	databaseName, collectionName, _ := seedWALPruneCollection(t, embedded)

	time.Sleep(2100 * time.Millisecond)

	result, err := embedded.PruneCollectionWAL(
		collectionName,
		WithWALPruneDatabaseName(databaseName),
		WithWALPruneMaxAge(time.Second),
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	if result.CandidateCountTotal == 0 {
		t.Skip("no WAL prune candidates available in this runtime state")
	}
	require.Greater(t, result.PrunedCountTotal, uint64(0))
}

func TestEmbeddedPruneAllWAL(t *testing.T) {
	embedded, _ := startTestEmbedded(t)
	_, _, _ = seedWALPruneCollection(t, embedded)

	result, err := embedded.PruneAllWAL(
		WithWALPruneDryRun(),
		WithWALPruneVacuum(),
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.DryRun)
	require.True(t, result.VacuumRequested)
	require.False(t, result.VacuumExecuted)
}

func TestServerPruneAllWALRestartsServer(t *testing.T) {
	server, _ := startTestServer(t)

	result, err := server.PruneAllWAL(WithWALPruneDryRun())
	require.NoError(t, err)
	require.NotNil(t, result)
	requireServerHeartbeat(t, server.URL())
}

func TestServerPruneAllWALRestartFailureReturnsResultAndError(t *testing.T) {
	server, persistDir := startTestServer(t)

	cfg := DefaultServerConfig()
	cfg.Port = reserveFreeLoopbackPort(t)
	cfg.ListenAddress = "256.256.256.256"
	cfg.PersistPath = persistDir
	cfg.AllowReset = true

	server.stateMu.Lock()
	server.config = StartServerConfig{ConfigString: cfg.toYAML()}
	server.stateMu.Unlock()

	result, err := server.PruneAllWAL(WithWALPruneDryRun())
	require.Error(t, err)
	require.NotNil(t, result)
	require.Contains(t, err.Error(), "wal prune completed but server restart failed; server remains stopped")
	requireServerUnavailable(t, server.URL())
	require.ErrorIs(t, server.Stop(), ErrServerNotStarted)
}

func TestWALPruneInputValidation(t *testing.T) {
	embedded, _ := startTestEmbedded(t)
	server, _ := startTestServer(t)

	_, err := embedded.PruneCollectionWAL("")
	require.EqualError(t, err, "name is required")

	_, err = embedded.PruneCollectionWAL("docs")
	require.EqualError(t, err, "at least one WAL prune policy is required unless dry-run is enabled")

	_, err = embedded.PruneAllWAL()
	require.EqualError(t, err, "at least one WAL prune policy is required unless dry-run is enabled")

	_, err = embedded.PruneCollectionWAL("docs", WithWALPruneDatabaseName("ab"), WithWALPruneDryRun())
	require.EqualError(t, err, "database_name must be at least 3 characters")

	_, err = embedded.PruneCollectionWAL("docs", WithWALPruneTenantID("ab"), WithWALPruneDryRun())
	require.EqualError(t, err, "tenant_id must be at least 3 characters")

	_, err = embedded.PruneCollectionWAL("docs", WithWALPruneWatermark(100, 200), WithWALPruneDryRun())
	require.EqualError(t, err, "invalid wal prune option at index 0: wal prune watermark low bytes must be less than or equal to high bytes")

	_, err = embedded.PruneCollectionWAL("docs", WithWALPruneMaxAge(0), WithWALPruneDryRun())
	require.EqualError(t, err, "invalid wal prune option at index 0: max_age must be greater than 0")

	var nilOption WALPruneOption
	_, err = embedded.PruneCollectionWAL("docs", nilOption)
	require.EqualError(t, err, "wal prune option at index 0 is nil")

	_, err = server.PruneCollectionWAL("")
	require.EqualError(t, err, "name is required")

	_, err = server.PruneAllWAL()
	require.EqualError(t, err, "at least one WAL prune policy is required unless dry-run is enabled")
	requireServerHeartbeat(t, server.URL())

	var nilEmbedded *Embedded
	_, err = nilEmbedded.PruneAllWAL(WithWALPruneDryRun())
	require.ErrorIs(t, err, ErrEmbeddedNotStarted)
	_, err = nilEmbedded.PruneCollectionWAL("docs", WithWALPruneDryRun())
	require.ErrorIs(t, err, ErrEmbeddedNotStarted)

	var nilServer *Server
	_, err = nilServer.PruneAllWAL(WithWALPruneDryRun())
	require.ErrorIs(t, err, ErrServerNotStarted)
	_, err = nilServer.PruneCollectionWAL("docs", WithWALPruneDryRun())
	require.ErrorIs(t, err, ErrServerNotStarted)
}

func TestEmbeddedPruneCollectionWALNonexistentCollection(t *testing.T) {
	embedded, _ := startTestEmbedded(t)

	_, err := embedded.PruneCollectionWAL(
		"does-not-exist",
		WithWALPruneDatabaseName("default_database"),
		WithWALPruneDryRun(),
	)
	require.Error(t, err)
}

func seedWALPruneCollection(t *testing.T, embedded *Embedded) (string, string, string) {
	t.Helper()

	databaseName := fmt.Sprintf("wal_prune_db_%d", time.Now().UnixNano())
	require.NoError(t, embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}))

	collectionName := fmt.Sprintf("wal_prune_col_%d", time.Now().UnixNano())
	collection, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
		Configuration: map[string]any{
			"hnsw": map[string]any{
				"sync_threshold": 2,
			},
		},
		GetOrCreate: true,
	})
	require.NoError(t, err)

	require.NoError(t, embedded.Add(EmbeddedAddRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"wal-doc-1", "wal-doc-2"},
		Embeddings:   [][]float32{{0.1, 0.2, 0.3}, {0.3, 0.2, 0.1}},
		Documents:    []string{"first", "second"},
	}))

	return databaseName, collectionName, collection.ID
}
