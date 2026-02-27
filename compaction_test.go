package chroma

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEmbeddedCompactionAPIs(t *testing.T) {
	embedded, _ := startTestEmbedded(t)

	databaseName := fmt.Sprintf("compact_db_%d", time.Now().UnixNano())
	require.NoError(t, embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}))

	collectionAName := fmt.Sprintf("compact_a_%d", time.Now().UnixNano())
	collectionA, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         collectionAName,
		DatabaseName: databaseName,
		GetOrCreate:  true,
	})
	require.NoError(t, err)

	collectionBName := fmt.Sprintf("compact_b_%d", time.Now().UnixNano())
	collectionB, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         collectionBName,
		DatabaseName: databaseName,
		GetOrCreate:  true,
	})
	require.NoError(t, err)

	require.NoError(t, embedded.Add(EmbeddedAddRequest{
		CollectionID: collectionA.ID,
		DatabaseName: databaseName,
		IDs:          []string{"a-1"},
		Embeddings:   [][]float32{{0.1, 0.2, 0.3}},
		Documents:    []string{"a-doc"},
	}))
	require.NoError(t, embedded.Add(EmbeddedAddRequest{
		CollectionID: collectionB.ID,
		DatabaseName: databaseName,
		IDs:          []string{"b-1"},
		Embeddings:   [][]float32{{0.3, 0.2, 0.1}},
		Documents:    []string{"b-doc"},
	}))

	compactedOne, err := embedded.CompactCollection(CompactCollectionRequest{
		Name:         collectionAName,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.NotNil(t, compactedOne)
	require.Equal(t, uint32(1), compactedOne.CollectionCount)
	require.Len(t, compactedOne.Collections, 1)
	require.Equal(t, collectionA.ID, compactedOne.Collections[0].CollectionID)
	require.Empty(t, compactedOne.Collections[0].Error)
	if compactedOne.Collections[0].PendingOpsBefore == nil {
		require.NotEmpty(t, compactedOne.Collections[0].PendingOpsBeforeError)
	} else {
		require.Empty(t, compactedOne.Collections[0].PendingOpsBeforeError)
	}
	if compactedOne.Collections[0].PendingOpsAfter == nil {
		require.NotEmpty(t, compactedOne.Collections[0].PendingOpsAfterError)
	} else {
		require.Empty(t, compactedOne.Collections[0].PendingOpsAfterError)
	}

	countA, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collectionA.ID,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.Equal(t, uint32(1), countA)

	compactedAll, err := embedded.CompactAll(CompactAllRequest{
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.NotNil(t, compactedAll)
	require.GreaterOrEqual(t, compactedAll.CollectionCount, uint32(2))

	found := map[string]bool{}
	for _, entry := range compactedAll.Collections {
		found[entry.Name] = true
		require.Empty(t, entry.Error)
		if entry.PendingOpsBefore == nil {
			require.NotEmpty(t, entry.PendingOpsBeforeError)
		} else {
			require.Empty(t, entry.PendingOpsBeforeError)
		}
		if entry.PendingOpsAfter == nil {
			require.NotEmpty(t, entry.PendingOpsAfterError)
		} else {
			require.Empty(t, entry.PendingOpsAfterError)
		}
	}
	require.True(t, found[collectionAName])
	require.True(t, found[collectionBName])

	unscopedResult, err := embedded.CompactAll(CompactAllRequest{})
	require.NoError(t, err)
	require.NotNil(t, unscopedResult)
	// The fixture creates two collections in the default tenant/database scope.
	require.GreaterOrEqual(t, unscopedResult.CollectionCount, uint32(2))
}

func TestServerCompactCollectionRestartsServer(t *testing.T) {
	server, databaseName, collectionName := startTestServerWithCollection(t)

	result, err := server.CompactCollection(CompactCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, uint32(1), result.CollectionCount)
	require.Len(t, result.Collections, 1)
	require.Equal(t, collectionName, result.Collections[0].Name)
	requireServerHeartbeat(t, server.URL())

	// Verify data survives the stop -> compact -> restart lifecycle.
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
	require.Equal(t, uint32(1), count)
}

func TestServerCompactAllRestartsServer(t *testing.T) {
	server, _ := startTestServer(t)

	result, err := server.CompactAll(CompactAllRequest{})
	require.NoError(t, err)
	require.NotNil(t, result)
	requireServerHeartbeat(t, server.URL())
}

func TestCompactCollectionNonexistentName(t *testing.T) {
	embedded, _ := startTestEmbedded(t)

	databaseName := fmt.Sprintf("missing_db_%d", time.Now().UnixNano())
	require.NoError(t, embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}))

	_, err := embedded.CompactCollection(CompactCollectionRequest{
		Name:         "does-not-exist",
		DatabaseName: databaseName,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "get collection failed")
}

func TestServerCompactCollectionNonexistentNameRestartsServer(t *testing.T) {
	server, _ := startTestServer(t)

	_, err := server.CompactCollection(CompactCollectionRequest{Name: "missing"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "get collection failed")
	requireServerHeartbeat(t, server.URL())
}

func TestServerCompactAllRestartFailureReturnsResultAndError(t *testing.T) {
	server, persistDir := startTestServer(t)

	cfg := DefaultServerConfig()
	cfg.Port = reserveFreeLoopbackPort(t)
	cfg.ListenAddress = "256.256.256.256"
	cfg.PersistPath = persistDir
	cfg.AllowReset = true

	server.stateMu.Lock()
	server.config = StartServerConfig{ConfigString: cfg.toYAML()}
	server.stateMu.Unlock()

	result, err := server.CompactAll(CompactAllRequest{})
	require.Error(t, err)
	require.NotNil(t, result)
	require.Contains(t, err.Error(), "compaction completed but server restart failed; server remains stopped")
	requireServerUnavailable(t, server.URL())
	require.ErrorIs(t, server.Stop(), ErrServerNotStarted)
}

func TestServerCompactAllDoubleFailureLeavesServerStopped(t *testing.T) {
	server, _ := startTestServer(t)

	server.stateMu.Lock()
	server.config = StartServerConfig{}
	server.stateMu.Unlock()

	result, err := server.CompactAll(CompactAllRequest{})
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "server remains stopped")
	requireServerUnavailable(t, server.URL())
	require.ErrorIs(t, server.Stop(), ErrServerNotStarted)
}

func TestCompactionInputValidation(t *testing.T) {
	embedded, _ := startTestEmbedded(t)
	server, _ := startTestServer(t)

	_, err := embedded.CompactCollection(CompactCollectionRequest{})
	require.EqualError(t, err, "name is required")

	_, err = embedded.CompactAll(CompactAllRequest{DatabaseName: "ab"})
	require.EqualError(t, err, "database_name must be at least 3 characters")

	_, err = server.CompactCollection(CompactCollectionRequest{})
	require.EqualError(t, err, "name is required")

	_, err = server.CompactAll(CompactAllRequest{DatabaseName: "ab"})
	require.EqualError(t, err, "database_name must be at least 3 characters")
	requireServerHeartbeat(t, server.URL())

	var nilEmbedded *Embedded
	_, err = nilEmbedded.CompactAll(CompactAllRequest{})
	require.ErrorIs(t, err, ErrEmbeddedNotStarted)

	_, err = nilEmbedded.CompactCollection(CompactCollectionRequest{Name: "n/a"})
	require.ErrorIs(t, err, ErrEmbeddedNotStarted)

	var nilServer *Server
	_, err = nilServer.CompactAll(CompactAllRequest{})
	require.ErrorIs(t, err, ErrServerNotStarted)

	_, err = nilServer.CompactCollection(CompactCollectionRequest{Name: "n/a"})
	require.ErrorIs(t, err, ErrServerNotStarted)
}

func startTestServerWithCollection(t *testing.T) (*Server, string, string) {
	t.Helper()
	require.NoError(t, Init(""))

	persistDir := filepath.Join(t.TempDir(), "server-compaction-persist")
	require.NoError(t, os.MkdirAll(persistDir, 0o755))

	databaseName := fmt.Sprintf("server_compact_db_%d", time.Now().UnixNano())
	collectionName := fmt.Sprintf("server_compact_col_%d", time.Now().UnixNano())

	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath(persistDir),
		WithEmbeddedAllowReset(true),
	)
	require.NoError(t, err)
	require.NoError(t, embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}))
	collection, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
		GetOrCreate:  true,
	})
	require.NoError(t, err)
	require.NoError(t, embedded.Add(EmbeddedAddRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"seed-1"},
		Embeddings:   [][]float32{{0.1, 0.1, 0.1}},
		Documents:    []string{"seed"},
	}))
	require.NoError(t, embedded.Close())

	serverPort := reserveFreeLoopbackPort(t)
	server, err := NewServer(
		WithPort(serverPort),
		WithListenAddress("127.0.0.1"),
		WithPersistPath(persistDir),
		WithAllowReset(true),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Close() })
	requireServerHeartbeat(t, server.URL())
	return server, databaseName, collectionName
}
