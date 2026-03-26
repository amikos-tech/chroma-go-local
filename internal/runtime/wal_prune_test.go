package runtime

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
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
	require.Equal(t, uint64(1), result.CollectionCount)
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
	require.Equal(t, uint64(1), result.CollectionCount)
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
		require.NotEmpty(t, result.Warnings, "vacuum failure should be surfaced as warning when prune succeeds")
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

	// Runtime defaults may keep sqlite auto-purge enabled, leaving no
	// prune-eligible WAL prefix for this fixture. Skip instead of failing when
	// the environment has no candidates.
	dryRun, err := embedded.PruneCollectionWAL(
		collectionName,
		WithWALPruneDatabaseName(databaseName),
		WithWALPruneDryRun(),
		WithWALPruneMaxBytes(0),
	)
	require.NoError(t, err)
	if dryRun.CandidateCountTotal == 0 {
		t.Skip("no WAL prune candidates available in this runtime state")
	}

	require.Eventually(t, func() bool {
		result, err := embedded.PruneCollectionWAL(
			collectionName,
			WithWALPruneDatabaseName(databaseName),
			WithWALPruneMaxAge(time.Second),
		)
		if err != nil {
			return false
		}
		return result.PrunedCountTotal > 0
	}, 6*time.Second, 250*time.Millisecond, "expected max-age prune to delete WAL rows when candidates exist")
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
	blockedPort, releaseBlockedPort := reserveBusyLoopbackPort(t)
	defer releaseBlockedPort()

	cfg := DefaultServerConfig()
	cfg.Port = blockedPort
	cfg.ListenAddress = "127.0.0.1"
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

func TestWALPruneInvariantsGopter(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 15
	parameters.Workers = 1
	parameters.Rng.Seed(20260307)
	properties := gopter.NewProperties(parameters)

	properties.Property("prune preserves record count and query results", prop.ForAll(
		func(recordCount uint8, seed int64) bool {
			return runWALPruneInvariantCase(t, int(recordCount), seed)
		},
		gen.UInt8Range(4, 50),
		gen.Int64(),
	))

	properties.TestingRun(t)
}

func runWALPruneInvariantCase(t *testing.T, recordCount int, seed int64) bool {
	t.Helper()

	if err := Init(""); err != nil {
		t.Logf("init failed: %v", err)
		return false
	}

	rootDir := makeManagedTestTempDir(t, "chroma-wal-prune-gopter-")
	persistPath := filepath.Join(rootDir, fmt.Sprintf("wal-prune-gopter-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(persistPath, 0o755); err != nil {
		t.Logf("persist path create failed: %v", err)
		return false
	}

	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath(persistPath),
		WithEmbeddedAllowReset(true),
	)
	if err != nil {
		t.Logf("embedded start failed: %v", err)
		return false
	}
	defer func() {
		_ = embedded.Close()
		waitForWindowsDirectoryUnlock(t, persistPath)
	}()

	databaseName := fmt.Sprintf("wal_prune_gopter_db_%d", time.Now().UnixNano())
	if err := embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}); err != nil {
		t.Logf("database create failed: %v", err)
		return false
	}

	rng := rand.New(rand.NewSource(seed))
	collectionName := fmt.Sprintf("wal_prune_gopter_col_%d", time.Now().UnixNano())
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
	if err != nil {
		t.Logf("collection create failed: %v", err)
		return false
	}

	ids := make([]string, 0, recordCount)
	embeddings := make([][]float32, 0, recordCount)
	for i := 0; i < recordCount; i++ {
		ids = append(ids, fmt.Sprintf("wp-%03d", i))
		embeddings = append(embeddings, []float32{
			rng.Float32(),
			rng.Float32(),
			rng.Float32(),
		})
	}

	if err := embedded.Add(EmbeddedAddRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          ids,
		Embeddings:   embeddings,
	}); err != nil {
		t.Logf("add failed: %v", err)
		return false
	}

	countBefore, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
	})
	if err != nil {
		t.Logf("count before failed: %v", err)
		return false
	}
	if countBefore != uint32(recordCount) {
		t.Logf("count before mismatch: got %d, want %d", countBefore, recordCount)
		return false
	}

	// Invariant 1: dry-run never changes record count
	dryResult, err := embedded.PruneCollectionWAL(
		collectionName,
		WithWALPruneDatabaseName(databaseName),
		WithWALPruneDryRun(),
		WithWALPruneMaxBytes(0),
	)
	if err != nil {
		t.Logf("dry-run prune failed: %v", err)
		return false
	}
	if !dryResult.DryRun {
		t.Log("dry-run result did not reflect dry_run=true")
		return false
	}

	countAfterDry, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
	})
	if err != nil {
		t.Logf("count after dry-run failed: %v", err)
		return false
	}
	if countAfterDry != countBefore {
		t.Logf("dry-run changed record count: before=%d, after=%d", countBefore, countAfterDry)
		return false
	}

	// Invariant 2: actual prune preserves record count (WAL prune != record delete)
	pruneResult, err := embedded.PruneCollectionWAL(
		collectionName,
		WithWALPruneDatabaseName(databaseName),
		WithWALPruneMaxBytes(0),
	)
	if err != nil {
		t.Logf("prune failed: %v", err)
		return false
	}

	countAfterPrune, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
	})
	if err != nil {
		t.Logf("count after prune failed: %v", err)
		return false
	}
	if countAfterPrune != countBefore {
		t.Logf("prune changed record count: before=%d, after=%d", countBefore, countAfterPrune)
		return false
	}

	// Invariant 3: PrunedCountTotal <= CandidateCountTotal
	if pruneResult.PrunedCountTotal > pruneResult.CandidateCountTotal {
		t.Logf("pruned %d > candidates %d", pruneResult.PrunedCountTotal, pruneResult.CandidateCountTotal)
		return false
	}

	// Invariant 4: per-collection totals aggregate correctly
	var sumPruned, sumCandidates uint64
	for _, c := range pruneResult.Collections {
		sumPruned += c.PrunedCount
		sumCandidates += c.CandidateCount
	}
	if sumPruned != pruneResult.PrunedCountTotal {
		t.Logf("per-collection pruned sum %d != total %d", sumPruned, pruneResult.PrunedCountTotal)
		return false
	}
	if sumCandidates != pruneResult.CandidateCountTotal {
		t.Logf("per-collection candidate sum %d != total %d", sumCandidates, pruneResult.CandidateCountTotal)
		return false
	}

	// Invariant 5: records remain queryable after prune
	queryIdx := rng.Intn(recordCount)
	query, err := embedded.Query(EmbeddedQueryRequest{
		CollectionID:    collection.ID,
		DatabaseName:    databaseName,
		QueryEmbeddings: [][]float32{embeddings[queryIdx]},
		NResults:        1,
	})
	if err != nil {
		t.Logf("query after prune failed: %v", err)
		return false
	}
	if len(query.IDs) != 1 || len(query.IDs[0]) != 1 {
		t.Log("query after prune returned unexpected result shape")
		return false
	}

	// Invariant 6: second prune has fewer or equal candidates
	secondResult, err := embedded.PruneCollectionWAL(
		collectionName,
		WithWALPruneDatabaseName(databaseName),
		WithWALPruneDryRun(),
		WithWALPruneMaxBytes(0),
	)
	if err != nil {
		t.Logf("second dry-run failed: %v", err)
		return false
	}
	if secondResult.CandidateCountTotal > dryResult.CandidateCountTotal {
		t.Logf("second dry-run candidates %d > first %d (prune should not increase candidates)",
			secondResult.CandidateCountTotal, dryResult.CandidateCountTotal)
		return false
	}

	return true
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
