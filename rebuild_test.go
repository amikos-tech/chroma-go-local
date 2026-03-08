package chroma

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedRebuildCollection(t *testing.T) {
	embedded, _ := startTestEmbedded(t)

	databaseName := fmt.Sprintf("rebuild_db_%d", time.Now().UnixNano())
	require.NoError(t, embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}))

	collectionName := fmt.Sprintf("rebuild_col_%d", time.Now().UnixNano())
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
		IDs:          []string{"doc-1", "doc-2"},
		Embeddings:   [][]float32{{0.1, 0.2, 0.3}, {0.3, 0.2, 0.1}},
		Documents:    []string{"first", "second"},
	}))

	_, err = embedded.CompactCollection(CompactCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)

	result, err := embedded.RebuildCollection(collectionName, WithRebuildDatabaseName(databaseName))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, collection.ID, result.CollectionID)
	require.Equal(t, collectionName, result.Name)
	require.Equal(t, databaseName, result.DatabaseName)
	require.False(t, result.Precheck)
	require.True(t, result.WouldRebuild)
	require.True(t, result.Rebuilt)
	require.GreaterOrEqual(t, result.RecordsScanned, uint64(1))
	require.GreaterOrEqual(t, result.VectorsReindexed, uint64(1))
	require.NotEmpty(t, result.BackupPath)
	_, statErr := os.Stat(result.BackupPath)
	require.NoError(t, statErr)

	count, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.Equal(t, uint32(2), count)

	query, err := embedded.Query(EmbeddedQueryRequest{
		CollectionID:    collection.ID,
		DatabaseName:    databaseName,
		QueryEmbeddings: [][]float32{{0.1, 0.2, 0.3}},
		NResults:        1,
	})
	require.NoError(t, err)
	require.Len(t, query.IDs, 1)
	require.Len(t, query.IDs[0], 1)
	require.Equal(t, "doc-1", query.IDs[0][0])
}

func TestEmbeddedRebuildCollectionPrecheckNoMutation(t *testing.T) {
	embedded, _ := startTestEmbedded(t)

	databaseName := fmt.Sprintf("rebuild_precheck_db_%d", time.Now().UnixNano())
	require.NoError(t, embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}))

	collectionName := fmt.Sprintf("rebuild_precheck_col_%d", time.Now().UnixNano())
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
		IDs:          []string{"precheck-1", "precheck-2"},
		Embeddings:   [][]float32{{0.42, 0.24, 0.12}, {0.11, 0.12, 0.13}},
	}))

	_, err = embedded.CompactCollection(CompactCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)

	countBefore, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)

	result, err := embedded.RebuildCollection(
		collectionName,
		WithRebuildDatabaseName(databaseName),
		WithRebuildPrecheck(),
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Precheck)
	require.True(t, result.WouldRebuild)
	require.False(t, result.Rebuilt)
	require.Equal(t, uint64(0), result.VectorsReindexed)
	require.Empty(t, result.BackupPath)

	countAfter, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.Equal(t, countBefore, countAfter)

	query, err := embedded.Query(EmbeddedQueryRequest{
		CollectionID:    collection.ID,
		DatabaseName:    databaseName,
		QueryEmbeddings: [][]float32{{0.42, 0.24, 0.12}},
		NResults:        1,
	})
	require.NoError(t, err)
	require.Len(t, query.IDs, 1)
	require.Len(t, query.IDs[0], 1)
	require.Equal(t, "precheck-1", query.IDs[0][0])
}

func TestEmbeddedRebuildCollectionValidation(t *testing.T) {
	embedded, _ := startTestEmbedded(t)
	server, _ := startTestServer(t)

	_, err := embedded.RebuildCollection("")
	require.EqualError(t, err, "name is required")
	_, err = embedded.RebuildCollection("   ")
	require.EqualError(t, err, "name is required")

	_, err = embedded.RebuildCollection("docs", WithRebuildDatabaseName("ab"))
	require.EqualError(t, err, "database_name must be at least 3 characters")

	_, err = embedded.RebuildCollection("docs", WithRebuildTenantID("ab"))
	require.EqualError(t, err, "tenant_id must be at least 3 characters")

	var nilOption RebuildCollectionOption
	_, err = embedded.RebuildCollection("docs", nilOption)
	require.EqualError(t, err, "rebuild option at index 0 is nil")

	_, err = server.RebuildCollection("")
	require.EqualError(t, err, "name is required")
	_, err = server.RebuildCollection("   ")
	require.EqualError(t, err, "name is required")

	_, err = server.RebuildCollection("docs", WithRebuildDatabaseName("ab"))
	require.EqualError(t, err, "database_name must be at least 3 characters")

	_, err = server.RebuildCollection("docs", WithRebuildTenantID("ab"))
	require.EqualError(t, err, "tenant_id must be at least 3 characters")

	_, err = server.RebuildCollection("docs", nilOption)
	require.EqualError(t, err, "rebuild option at index 0 is nil")
	requireServerHeartbeat(t, server.URL())

	var nilEmbedded *Embedded
	_, err = nilEmbedded.RebuildCollection("docs")
	require.ErrorIs(t, err, ErrEmbeddedNotStarted)

	var nilServer *Server
	_, err = nilServer.RebuildCollection("docs")
	require.ErrorIs(t, err, ErrServerNotStarted)
}

func TestEmbeddedRebuildCollectionNonexistent(t *testing.T) {
	embedded, _ := startTestEmbedded(t)

	_, err := embedded.RebuildCollection("missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "get collection failed")
}

func TestEmbeddedRebuildCollectionEmptyCollectionSkips(t *testing.T) {
	embedded, _ := startTestEmbedded(t)

	databaseName := fmt.Sprintf("rebuild_empty_db_%d", time.Now().UnixNano())
	require.NoError(t, embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}))

	collectionName := fmt.Sprintf("rebuild_empty_col_%d", time.Now().UnixNano())
	collection, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
		GetOrCreate:  true,
	})
	require.NoError(t, err)

	result, err := embedded.RebuildCollection(collectionName, WithRebuildDatabaseName(databaseName))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.WouldRebuild)
	require.False(t, result.Rebuilt)
	require.Empty(t, result.BackupPath)
	require.NotEmpty(t, result.Warnings)

	count, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.Equal(t, uint32(0), count)
}

func TestServerRebuildCollectionRestartsServer(t *testing.T) {
	server, databaseName, collectionName := startTestServerWithRebuildReadyCollection(t)

	result, err := server.RebuildCollection(collectionName, WithRebuildDatabaseName(databaseName))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, collectionName, result.Name)
	require.Equal(t, databaseName, result.DatabaseName)
	require.True(t, result.WouldRebuild)
	requireServerHeartbeat(t, server.URL())

	// Verify data survives the stop -> rebuild -> restart lifecycle.
	require.NoError(t, server.Close())
	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath(server.persistPath),
		WithEmbeddedAllowReset(true),
	)
	require.NoError(t, err)
	defer embedded.Close()

	collection, err := embedded.GetCollection(EmbeddedGetCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)

	count, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.Equal(t, uint32(2), count)
}

func TestServerRebuildCollectionPrecheckRestartsServer(t *testing.T) {
	server, databaseName, collectionName := startTestServerWithRebuildReadyCollection(t)

	result, err := server.RebuildCollection(
		collectionName,
		WithRebuildDatabaseName(databaseName),
		WithRebuildPrecheck(),
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Precheck)
	require.True(t, result.WouldRebuild)
	require.False(t, result.Rebuilt)
	require.Empty(t, result.BackupPath)
	requireServerHeartbeat(t, server.URL())

	// Verify data survives the stop -> precheck -> restart lifecycle.
	require.NoError(t, server.Close())
	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath(server.persistPath),
		WithEmbeddedAllowReset(true),
	)
	require.NoError(t, err)
	defer embedded.Close()

	collection, err := embedded.GetCollection(EmbeddedGetCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)

	count, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.Equal(t, uint32(2), count)
}

func TestServerRebuildCollectionNonexistentRestartsServer(t *testing.T) {
	server, _ := startTestServer(t)

	_, err := server.RebuildCollection("missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "get collection failed")
	requireServerHeartbeat(t, server.URL())
}

func TestServerRebuildCollectionRestartFailureReturnsResultAndError(t *testing.T) {
	server, databaseName, collectionName := startTestServerWithRebuildReadyCollection(t)
	blockedPort, releaseBlockedPort := reserveBusyLoopbackPort(t)
	defer releaseBlockedPort()

	cfg := DefaultServerConfig()
	cfg.Port = blockedPort
	cfg.ListenAddress = "127.0.0.1"
	cfg.PersistPath = server.persistPath
	cfg.AllowReset = true

	server.stateMu.Lock()
	server.config = StartServerConfig{ConfigString: cfg.toYAML()}
	server.stateMu.Unlock()

	result, err := server.RebuildCollection(collectionName, WithRebuildDatabaseName(databaseName))
	require.Error(t, err)
	require.NotNil(t, result)
	require.Contains(t, err.Error(), "rebuild completed but server restart failed; server remains stopped")
	requireServerUnavailable(t, server.URL())
	require.ErrorIs(t, server.Stop(), ErrServerNotStarted)
}

func TestEmbeddedRebuildCollectionBackupDefaultAndDisable(t *testing.T) {
	embedded, _ := startTestEmbedded(t)

	databaseName := fmt.Sprintf("rebuild_backup_db_%d", time.Now().UnixNano())
	require.NoError(t, embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}))

	collectionOneName := fmt.Sprintf("rebuild_backup_one_%d", time.Now().UnixNano())
	collectionOne, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         collectionOneName,
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
		CollectionID: collectionOne.ID,
		DatabaseName: databaseName,
		IDs:          []string{"backup-1", "backup-1b"},
		Embeddings:   [][]float32{{0.11, 0.22, 0.33}, {0.21, 0.32, 0.43}},
	}))

	_, err = embedded.CompactCollection(CompactCollectionRequest{
		Name:         collectionOneName,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)

	collectionTwoName := fmt.Sprintf("rebuild_backup_two_%d", time.Now().UnixNano())
	collectionTwo, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         collectionTwoName,
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
		CollectionID: collectionTwo.ID,
		DatabaseName: databaseName,
		IDs:          []string{"backup-2", "backup-2b"},
		Embeddings:   [][]float32{{0.44, 0.55, 0.66}, {0.54, 0.65, 0.76}},
	}))

	_, err = embedded.CompactCollection(CompactCollectionRequest{
		Name:         collectionTwoName,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)

	withBackup, err := embedded.RebuildCollection(collectionOneName, WithRebuildDatabaseName(databaseName))
	require.NoError(t, err)
	require.NotNil(t, withBackup)
	require.NotEmpty(t, withBackup.BackupPath)
	_, statErr := os.Stat(withBackup.BackupPath)
	require.NoError(t, statErr)

	withoutBackup, err := embedded.RebuildCollection(
		collectionTwoName,
		WithRebuildDatabaseName(databaseName),
		WithRebuildKeepBackup(false),
	)
	require.NoError(t, err)
	require.NotNil(t, withoutBackup)
	require.Empty(t, withoutBackup.BackupPath)
}

func startTestServerWithRebuildReadyCollection(t *testing.T) (*Server, string, string) {
	t.Helper()
	require.NoError(t, Init(""))

	rootDir := makeManagedTestTempDir(t, "chroma-server-rebuild-")
	persistDir := filepath.Join(rootDir, "server-rebuild-persist")
	require.NoError(t, os.MkdirAll(persistDir, 0o755))

	databaseName := fmt.Sprintf("server_rebuild_db_%d", time.Now().UnixNano())
	collectionName := fmt.Sprintf("server_rebuild_col_%d", time.Now().UnixNano())

	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath(persistDir),
		WithEmbeddedAllowReset(true),
	)
	require.NoError(t, err)
	embeddedClosed := false
	defer func() {
		if embeddedClosed {
			return
		}
		_ = embedded.Close()
		waitForWindowsDirectoryUnlock(t, persistDir)
	}()
	require.NoError(t, embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}))
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
		IDs:          []string{"seed-1", "seed-2"},
		Embeddings:   [][]float32{{0.1, 0.1, 0.1}, {0.2, 0.2, 0.2}},
		Documents:    []string{"seed-1", "seed-2"},
	}))
	_, err = embedded.CompactCollection(CompactCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	closeErr := embedded.Close()
	embeddedClosed = true
	require.NoError(t, closeErr)
	waitForWindowsDirectoryUnlock(t, persistDir)

	serverPort := reserveFreeLoopbackPort(t)
	server, err := NewServer(
		WithPort(serverPort),
		WithListenAddress("127.0.0.1"),
		WithPersistPath(persistDir),
		WithAllowReset(true),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = server.Close()
		waitForWindowsDirectoryUnlock(t, persistDir)
	})
	requireServerHeartbeat(t, server.URL())
	return server, databaseName, collectionName
}

func TestEmbeddedRebuildCollectionShrinksAfterDeletesProperty(t *testing.T) {
	type scenario struct {
		name string
		add  int
		keep int
		seed int64
	}

	scenarios := make([]scenario, 0, 16)
	scenarioRNG := rand.New(rand.NewSource(20260302))
	for i := 0; i < 12; i++ {
		add := 64 + scenarioRNG.Intn(129) // [64, 192]
		maxKeep := add / 4                // force dense delete workloads
		if maxKeep < 1 {
			maxKeep = 1
		}
		keep := 1 + scenarioRNG.Intn(maxKeep)
		scenarios = append(scenarios, scenario{
			name: fmt.Sprintf("dense_delete_%02d_add_%d_keep_%d", i+1, add, keep),
			add:  add,
			keep: keep,
			seed: scenarioRNG.Int63(),
		})
	}

	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			embedded, _ := startTestEmbedded(t)
			databaseName := fmt.Sprintf("rebuild_shrink_db_%d", time.Now().UnixNano())
			require.NoError(t, embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}))

			collectionName := fmt.Sprintf("rebuild_shrink_col_%d", time.Now().UnixNano())
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

			ids := make([]string, 0, scenario.add)
			embeddings := make([][]float32, 0, scenario.add)
			for i := 0; i < scenario.add; i++ {
				ids = append(ids, fmt.Sprintf("bulk-%03d", i))
				embeddings = append(embeddings, []float32{
					float32(i) / 100.0,
					float32(i%17) / 25.0,
					float32((i*3)%23) / 30.0,
				})
			}
			require.NoError(t, embedded.Add(EmbeddedAddRequest{
				CollectionID: collection.ID,
				DatabaseName: databaseName,
				IDs:          ids,
				Embeddings:   embeddings,
			}))

			_, err = embedded.CompactCollection(CompactCollectionRequest{
				Name:         collectionName,
				DatabaseName: databaseName,
			})
			require.NoError(t, err)

			rng := rand.New(rand.NewSource(scenario.seed))
			permutation := rng.Perm(scenario.add)
			keepSet := make(map[string]struct{}, scenario.keep)
			for i := 0; i < scenario.keep; i++ {
				keepSet[ids[permutation[i]]] = struct{}{}
			}

			deleteIDs := make([]string, 0, scenario.add-scenario.keep)
			for _, id := range ids {
				if _, keep := keepSet[id]; !keep {
					deleteIDs = append(deleteIDs, id)
				}
			}
			require.NoError(t, embedded.DeleteRecords(EmbeddedDeleteRecordsRequest{
				CollectionID: collection.ID,
				DatabaseName: databaseName,
				IDs:          deleteIDs,
			}))

			_, err = embedded.CompactCollection(CompactCollectionRequest{
				Name:         collectionName,
				DatabaseName: databaseName,
			})
			require.NoError(t, err)

			rebuildResult, err := embedded.RebuildCollection(collectionName, WithRebuildDatabaseName(databaseName))
			require.NoError(t, err)
			require.NotNil(t, rebuildResult)
			require.True(t, rebuildResult.Rebuilt)
			require.NotEmpty(t, rebuildResult.BackupPath)

			canonicalPath, err := canonicalIndexPathFromBackup(rebuildResult.BackupPath)
			require.NoError(t, err)
			rebuiltSize, err := directorySizeBytes(canonicalPath)
			require.NoError(t, err)
			backupSize, err := directorySizeBytes(rebuildResult.BackupPath)
			require.NoError(t, err)
			require.Less(t, rebuiltSize, backupSize, "rebuilt index should shrink after many deletes")

			count, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
				CollectionID: collection.ID,
				DatabaseName: databaseName,
			})
			require.NoError(t, err)
			require.Equal(t, uint32(scenario.keep), count)
		})
	}
}

func TestEmbeddedRebuildCollectionShrinksAfterDeletesGopter(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20
	parameters.Workers = 1
	parameters.Rng.Seed(20260302)
	properties := gopter.NewProperties(parameters)

	properties.Property("rebuild shrinks after dense delete workloads", prop.ForAll(
		func(add uint16, keepPercent uint8, seed int64) bool {
			return runRebuildShrinkPropertyCase(t, int(add), int(keepPercent), seed)
		},
		gen.UInt16Range(64, 220),
		gen.UInt8Range(1, 30),
		gen.Int64(),
	))

	properties.TestingRun(t)
}

func runRebuildShrinkPropertyCase(t *testing.T, addCount int, keepPercent int, seed int64) bool {
	t.Helper()

	if err := Init(""); err != nil {
		t.Logf("init failed: %v", err)
		return false
	}

	rootDir := makeManagedTestTempDir(t, "chroma-rebuild-gopter-")
	persistPath := filepath.Join(rootDir, fmt.Sprintf("rebuild-gopter-persist-%d", time.Now().UnixNano()))
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

	databaseName := fmt.Sprintf("rebuild_shrink_gopter_db_%d", time.Now().UnixNano())
	if err := embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}); err != nil {
		t.Logf("database create failed: %v", err)
		return false
	}

	collectionName := fmt.Sprintf("rebuild_shrink_gopter_col_%d", time.Now().UnixNano())
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

	ids := make([]string, 0, addCount)
	embeddings := make([][]float32, 0, addCount)
	for i := 0; i < addCount; i++ {
		ids = append(ids, fmt.Sprintf("prop-%03d", i))
		embeddings = append(embeddings, []float32{
			float32(i) / 100.0,
			float32(i%17) / 25.0,
			float32((i*3)%23) / 30.0,
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

	if _, err := embedded.CompactCollection(CompactCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
	}); err != nil {
		t.Logf("first compact failed: %v", err)
		return false
	}

	keepCount := (addCount * keepPercent) / 100
	if keepCount < 1 {
		keepCount = 1
	}
	if keepCount >= addCount {
		keepCount = addCount - 1
	}

	rng := rand.New(rand.NewSource(seed))
	permutation := rng.Perm(addCount)
	keepSet := make(map[string]struct{}, keepCount)
	for i := 0; i < keepCount; i++ {
		keepSet[ids[permutation[i]]] = struct{}{}
	}

	deleteIDs := make([]string, 0, addCount-keepCount)
	for _, id := range ids {
		if _, keep := keepSet[id]; !keep {
			deleteIDs = append(deleteIDs, id)
		}
	}

	if err := embedded.DeleteRecords(EmbeddedDeleteRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          deleteIDs,
	}); err != nil {
		t.Logf("delete failed: %v", err)
		return false
	}

	if _, err := embedded.CompactCollection(CompactCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
	}); err != nil {
		t.Logf("second compact failed: %v", err)
		return false
	}

	rebuildResult, err := embedded.RebuildCollection(collectionName, WithRebuildDatabaseName(databaseName))
	if err != nil {
		t.Logf("rebuild failed (add=%d keep=%d seed=%d): %v", addCount, keepCount, seed, err)
		return false
	}
	if rebuildResult == nil || !rebuildResult.Rebuilt || rebuildResult.BackupPath == "" {
		t.Logf("unexpected rebuild result (add=%d keep=%d seed=%d): %#v", addCount, keepCount, seed, rebuildResult)
		return false
	}

	canonicalPath, err := canonicalIndexPathFromBackup(rebuildResult.BackupPath)
	if err != nil {
		t.Logf("canonical path parse failed: %v", err)
		return false
	}
	rebuiltSize, err := directorySizeBytes(canonicalPath)
	if err != nil {
		t.Logf("rebuilt size read failed: %v", err)
		return false
	}
	backupSize, err := directorySizeBytes(rebuildResult.BackupPath)
	if err != nil {
		t.Logf("backup size read failed: %v", err)
		return false
	}
	if rebuiltSize >= backupSize {
		t.Logf("expected shrink but got rebuilt=%d backup=%d add=%d keep=%d seed=%d", rebuiltSize, backupSize, addCount, keepCount, seed)
		return false
	}

	count, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
	})
	if err != nil {
		t.Logf("count failed: %v", err)
		return false
	}
	if count != uint32(keepCount) {
		t.Logf("unexpected record count got=%d expected=%d", count, keepCount)
		return false
	}

	return true
}

func TestEmbeddedRebuildCollectionShrinksLargeDataset(t *testing.T) {
	if os.Getenv("CHROMA_LONG_REBUILD_TESTS") != "1" {
		t.Skip("set CHROMA_LONG_REBUILD_TESTS=1 to run large rebuild shrink test")
	}

	const addCount = 6000
	const keepCount = 600
	const seed = int64(20260302)

	embedded, _ := startTestEmbedded(t)
	databaseName := fmt.Sprintf("rebuild_large_db_%d", time.Now().UnixNano())
	require.NoError(t, embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}))

	collectionName := fmt.Sprintf("rebuild_large_col_%d", time.Now().UnixNano())
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

	ids := make([]string, 0, addCount)
	embeddings := make([][]float32, 0, addCount)
	for i := 0; i < addCount; i++ {
		ids = append(ids, fmt.Sprintf("large-%05d", i))
		embeddings = append(embeddings, []float32{
			float32(i) / 100.0,
			float32(i%17) / 25.0,
			float32((i*3)%23) / 30.0,
		})
	}
	require.NoError(t, embedded.Add(EmbeddedAddRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          ids,
		Embeddings:   embeddings,
	}))

	_, err = embedded.CompactCollection(CompactCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)

	rng := rand.New(rand.NewSource(seed))
	permutation := rng.Perm(addCount)
	keepSet := make(map[string]struct{}, keepCount)
	for i := 0; i < keepCount; i++ {
		keepSet[ids[permutation[i]]] = struct{}{}
	}

	deleteIDs := make([]string, 0, addCount-keepCount)
	for _, id := range ids {
		if _, keep := keepSet[id]; !keep {
			deleteIDs = append(deleteIDs, id)
		}
	}
	require.NoError(t, embedded.DeleteRecords(EmbeddedDeleteRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          deleteIDs,
	}))

	_, err = embedded.CompactCollection(CompactCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)

	rebuildResult, err := embedded.RebuildCollection(collectionName, WithRebuildDatabaseName(databaseName))
	require.NoError(t, err)
	require.NotNil(t, rebuildResult)
	require.True(t, rebuildResult.Rebuilt)
	require.NotEmpty(t, rebuildResult.BackupPath)

	canonicalPath, err := canonicalIndexPathFromBackup(rebuildResult.BackupPath)
	require.NoError(t, err)
	rebuiltSize, err := directorySizeBytes(canonicalPath)
	require.NoError(t, err)
	backupSize, err := directorySizeBytes(rebuildResult.BackupPath)
	require.NoError(t, err)
	require.Less(t, rebuiltSize, backupSize)

	ratio := float64(rebuiltSize) / float64(backupSize)
	require.Lessf(t, ratio, 0.90, "expected clear shrink for large dataset, rebuilt=%d backup=%d ratio=%.3f", rebuiltSize, backupSize, ratio)

	count, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.Equal(t, uint32(keepCount), count)
}

func TestEmbeddedRebuildCollectionCorruptMetadataDoesNotSwap(t *testing.T) {
	embedded, persistPath := startTestEmbedded(t)
	databaseName := fmt.Sprintf("rebuild_corrupt_metadata_db_%d", time.Now().UnixNano())
	require.NoError(t, embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}))

	collectionName := fmt.Sprintf("rebuild_corrupt_metadata_col_%d", time.Now().UnixNano())
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
		IDs:          []string{"stable-1", "stable-2"},
		Embeddings:   [][]float32{{0.1, 0.2, 0.3}, {0.3, 0.2, 0.1}},
	}))
	_, err = embedded.CompactCollection(CompactCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)

	metadataPath, err := findMetadataPath(persistPath)
	require.NoError(t, err)
	indexDir := filepath.Dir(metadataPath)

	require.NoError(t, os.WriteFile(metadataPath, []byte("not-a-pickle"), 0o644))
	hashBefore, err := directoryHash(indexDir)
	require.NoError(t, err)

	_, err = embedded.RebuildCollection(collectionName, WithRebuildDatabaseName(databaseName))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode hnsw metadata")

	hashAfter, err := directoryHash(indexDir)
	require.NoError(t, err)
	require.Equal(t, hashBefore, hashAfter, "failed rebuild must not mutate source index artifacts")
	requireNoBackupOrRollbackDirs(t, persistPath)
}

func TestEmbeddedRebuildCollectionCorruptIndexDoesNotSwap(t *testing.T) {
	embedded, persistPath := startTestEmbedded(t)
	databaseName := fmt.Sprintf("rebuild_corrupt_index_db_%d", time.Now().UnixNano())
	require.NoError(t, embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}))

	collectionName := fmt.Sprintf("rebuild_corrupt_index_col_%d", time.Now().UnixNano())
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
		IDs:          []string{"stable-1", "stable-2"},
		Embeddings:   [][]float32{{0.1, 0.2, 0.3}, {0.3, 0.2, 0.1}},
	}))
	_, err = embedded.CompactCollection(CompactCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)

	metadataPath, err := findMetadataPath(persistPath)
	require.NoError(t, err)
	indexDir := filepath.Dir(metadataPath)

	headerPath := filepath.Join(indexDir, "header.bin")
	require.NoError(t, os.WriteFile(headerPath, []byte("corrupt-header"), 0o644))
	hashBefore, err := directoryHash(indexDir)
	require.NoError(t, err)

	_, err = embedded.RebuildCollection(collectionName, WithRebuildDatabaseName(databaseName))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load source hnsw index")

	hashAfter, err := directoryHash(indexDir)
	require.NoError(t, err)
	require.Equal(t, hashBefore, hashAfter, "failed rebuild must not mutate source index artifacts")
	requireNoBackupOrRollbackDirs(t, persistPath)
}

func canonicalIndexPathFromBackup(backupPath string) (string, error) {
	base := filepath.Base(backupPath)
	marker := "_backup_"
	pos := strings.Index(base, marker)
	if pos <= 0 {
		return "", fmt.Errorf("unexpected backup path format %q", backupPath)
	}
	return filepath.Join(filepath.Dir(backupPath), base[:pos]), nil
}

func directorySizeBytes(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(filePath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d == nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func findMetadataPath(persistPath string) (string, error) {
	matches := make([]string, 0, 2)
	err := filepath.WalkDir(persistPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d != nil && !d.IsDir() && d.Name() == "index_metadata.pickle" {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("expected exactly one metadata file under %q, got %d", persistPath, len(matches))
	}
	return matches[0], nil
}

func directoryHash(path string) (string, error) {
	files := make([]string, 0, 8)
	if err := filepath.WalkDir(path, func(filePath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d != nil && !d.IsDir() {
			files = append(files, filePath)
		}
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(files)

	hasher := sha256.New()
	for _, filePath := range files {
		rel, err := filepath.Rel(path, filePath)
		if err != nil {
			return "", err
		}
		if _, err := io.WriteString(hasher, rel); err != nil {
			return "", err
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		if _, err := hasher.Write(data); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func requireNoBackupOrRollbackDirs(t *testing.T, persistPath string) {
	t.Helper()

	err := filepath.WalkDir(persistPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d == nil || !d.IsDir() {
			return nil
		}

		name := d.Name()
		if strings.Contains(name, "_backup_") || strings.Contains(name, "_rollback_") || strings.Contains(name, ".rebuild.") {
			return fmt.Errorf("unexpected backup/rollback/rebuild directory found: %s", path)
		}
		return nil
	})
	require.NoError(t, err)
}
