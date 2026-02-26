package chroma

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newEmbeddedForIntegrationTest(t *testing.T) *Embedded {
	t.Helper()

	if err := Init(""); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath(t.TempDir()),
		WithEmbeddedAllowReset(true),
	)
	if err != nil {
		t.Fatalf("Failed to start embedded mode: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := embedded.Close(); closeErr != nil {
			t.Errorf("failed to close embedded runtime: %v", closeErr)
		}
	})
	return embedded
}

func TestEmbeddedWhereValidationEdgeCases(t *testing.T) {
	embedded := newEmbeddedForIntegrationTest(t)

	databaseName := fmt.Sprintf("edge_db_%d", time.Now().UnixNano())
	if err := embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}); err != nil {
		t.Fatalf("CreateDatabase failed: %v", err)
	}

	collectionName := fmt.Sprintf("edge_collection_%d", time.Now().UnixNano())
	collection, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
		GetOrCreate:  true,
	})
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	if err := embedded.Add(EmbeddedAddRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-edge"},
		Embeddings:   [][]float32{{0.1, 0.2, 0.3}},
		Documents:    []string{"edge document"},
	}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	_, err = embedded.Query(EmbeddedQueryRequest{
		CollectionID:    collection.ID,
		DatabaseName:    databaseName,
		QueryEmbeddings: [][]float32{{0.1, 0.2, 0.3}},
		WhereDocument: map[string]any{
			"$contains": 123, // invalid: must be string
		},
	})
	if err == nil {
		t.Fatal("expected Query with invalid where_document to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "where document") &&
		!strings.Contains(strings.ToLower(err.Error()), "where_document") {
		t.Fatalf("expected where_document validation error, got: %v", err)
	}

	_, err = embedded.GetRecords(EmbeddedGetRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		Where: map[string]any{
			"a": "x",
			"b": "y", // invalid: metadata where object must have exactly one key
		},
	})
	if err == nil {
		t.Fatal("expected GetRecords with invalid where to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "where clause") &&
		!strings.Contains(strings.ToLower(err.Error()), "where") {
		t.Fatalf("expected where validation error, got: %v", err)
	}
}

func TestEmbeddedDeleteByDocumentFilterOnly(t *testing.T) {
	embedded := newEmbeddedForIntegrationTest(t)

	databaseName := fmt.Sprintf("delete_filter_db_%d", time.Now().UnixNano())
	if err := embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}); err != nil {
		t.Fatalf("CreateDatabase failed: %v", err)
	}

	collection, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         fmt.Sprintf("delete_filter_collection_%d", time.Now().UnixNano()),
		DatabaseName: databaseName,
		GetOrCreate:  true,
	})
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	if err := embedded.Add(EmbeddedAddRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-keep", "doc-delete"},
		Embeddings: [][]float32{
			{0.1, 0.2, 0.3},
			{0.3, 0.2, 0.1},
		},
		Documents: []string{"keep me", "delete me"},
	}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if err := embedded.DeleteRecords(EmbeddedDeleteRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		WhereDocument: map[string]any{
			"$contains": "delete",
		},
	}); err != nil {
		t.Fatalf("DeleteRecords by filter failed: %v", err)
	}

	var count uint32
	require.Eventually(t, func() bool {
		count, err = embedded.CountRecords(EmbeddedCountRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
		})
		return err == nil && count == 1
	}, 5*time.Second, 100*time.Millisecond, "CountRecords did not reach expected count after filtered delete")
}

func TestEmbeddedCreateCollectionRejectsInvalidConfigurationIntegration(t *testing.T) {
	embedded := newEmbeddedForIntegrationTest(t)

	_, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name: fmt.Sprintf("invalid_config_collection_%d", time.Now().UnixNano()),
		Configuration: map[string]any{
			"hnsw": map[string]any{
				"space": "invalid_space",
			},
		},
		GetOrCreate: true,
	})
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "invalid configuration")
}
