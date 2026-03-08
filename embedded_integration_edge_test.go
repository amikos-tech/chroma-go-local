package chroma

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	collectionEventuallyTimeout      = 5 * time.Second
	collectionEventuallyPollInterval = 100 * time.Millisecond
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

func isRetriableGetCollectionError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

func waitForCollectionConvergence(
	t *testing.T,
	embedded *Embedded,
	request EmbeddedGetCollectionRequest,
	description string,
	predicate func(*EmbeddedCollection) (bool, error),
) {
	t.Helper()

	var lastMetadata map[string]any
	var lastErr error
	var hardErr error

	require.Eventually(t, func() bool {
		updated, getErr := embedded.GetCollection(request)
		if getErr != nil {
			if isRetriableGetCollectionError(getErr) {
				lastErr = getErr
				return false
			}
			hardErr = getErr
			return true
		}
		lastErr = nil
		lastMetadata = updated.Metadata

		done, predicateErr := predicate(updated)
		if predicateErr != nil {
			hardErr = predicateErr
			return true
		}
		return done
	}, collectionEventuallyTimeout, collectionEventuallyPollInterval, "%s did not converge (last metadata=%#v, last err=%v)", description, lastMetadata, lastErr)

	if hardErr != nil {
		t.Fatalf("%s failed with non-retriable error: %v (last metadata=%#v, last err=%v)", description, hardErr, lastMetadata, lastErr)
	}
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

func TestEmbeddedDeleteByDocumentFilterWithLimit(t *testing.T) {
	embedded := newEmbeddedForIntegrationTest(t)

	databaseName := fmt.Sprintf("delete_filter_limit_db_%d", time.Now().UnixNano())
	if err := embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}); err != nil {
		t.Fatalf("CreateDatabase failed: %v", err)
	}

	collection, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         fmt.Sprintf("delete_filter_limit_collection_%d", time.Now().UnixNano()),
		DatabaseName: databaseName,
		GetOrCreate:  true,
	})
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	if err := embedded.Add(EmbeddedAddRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-1", "doc-2", "doc-3"},
		Embeddings: [][]float32{
			{0.1, 0.2, 0.3},
			{0.3, 0.2, 0.1},
			{0.4, 0.4, 0.4},
		},
		Documents: []string{"delete me first", "delete me second", "keep me"},
	}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	limit := uint32(1)
	if err := embedded.DeleteRecords(EmbeddedDeleteRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		WhereDocument: map[string]any{
			"$contains": "delete me",
		},
		Limit: &limit,
	}); err != nil {
		t.Fatalf("DeleteRecords with limit failed: %v", err)
	}

	var count uint32
	require.Eventually(t, func() bool {
		count, err = embedded.CountRecords(EmbeddedCountRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
		})
		return err == nil && count == 2
	}, 5*time.Second, 100*time.Millisecond, "CountRecords did not reach expected count after limited delete")
}

func TestEmbeddedDeleteByMetadataFilterWithLimit(t *testing.T) {
	embedded := newEmbeddedForIntegrationTest(t)

	databaseName := fmt.Sprintf("delete_meta_limit_db_%d", time.Now().UnixNano())
	if err := embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}); err != nil {
		t.Fatalf("CreateDatabase failed: %v", err)
	}

	collection, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         fmt.Sprintf("delete_meta_limit_collection_%d", time.Now().UnixNano()),
		DatabaseName: databaseName,
		GetOrCreate:  true,
	})
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	if err := embedded.Add(EmbeddedAddRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-1", "doc-2", "doc-3"},
		Embeddings: [][]float32{
			{0.1, 0.2, 0.3},
			{0.3, 0.2, 0.1},
			{0.4, 0.4, 0.4},
		},
		Documents: []string{"stale one", "stale two", "fresh"},
		Metadatas: []map[string]any{
			{"status": "stale"},
			{"status": "stale"},
			{"status": "fresh"},
		},
	}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	limit := uint32(1)
	if err := embedded.DeleteRecords(EmbeddedDeleteRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		Where: map[string]any{
			"status": "stale",
		},
		Limit: &limit,
	}); err != nil {
		t.Fatalf("DeleteRecords with metadata limit failed: %v", err)
	}

	var count uint32
	require.Eventually(t, func() bool {
		count, err = embedded.CountRecords(EmbeddedCountRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
		})
		return err == nil && count == 2
	}, 5*time.Second, 100*time.Millisecond, "CountRecords did not reach expected count after limited metadata delete")

	var getResp *EmbeddedGetRecordsResponse
	require.Eventually(t, func() bool {
		getResp, err = embedded.GetRecords(EmbeddedGetRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
			Where: map[string]any{
				"status": "stale",
			},
		})
		return err == nil && getResp != nil
	}, 5*time.Second, 100*time.Millisecond, "GetRecords did not converge after limited metadata delete")
	require.Len(t, getResp.IDs, 1)
}

func TestEmbeddedDeleteByMetadataFilterWithLimitResponse(t *testing.T) {
	embedded := newEmbeddedForIntegrationTest(t)

	databaseName := fmt.Sprintf("delete_meta_limit_response_db_%d", time.Now().UnixNano())
	if err := embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}); err != nil {
		t.Fatalf("CreateDatabase failed: %v", err)
	}

	collection, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         fmt.Sprintf("delete_meta_limit_response_collection_%d", time.Now().UnixNano()),
		DatabaseName: databaseName,
		GetOrCreate:  true,
	})
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	if err := embedded.Add(EmbeddedAddRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-1", "doc-2", "doc-3"},
		Embeddings: [][]float32{
			{0.1, 0.2, 0.3},
			{0.3, 0.2, 0.1},
			{0.4, 0.4, 0.4},
		},
		Metadatas: []map[string]any{
			{"status": "stale"},
			{"status": "stale"},
			{"status": "fresh"},
		},
	}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	limit := uint32(1)
	deleteResp, err := embedded.DeleteRecordsWithResponse(EmbeddedDeleteRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		Where: map[string]any{
			"status": "stale",
		},
		Limit: &limit,
	})
	if err != nil {
		t.Fatalf("DeleteRecordsWithResponse with metadata limit failed: %v", err)
	}
	require.NotNil(t, deleteResp)
	require.Equal(t, uint32(1), deleteResp.Deleted)

	var count uint32
	require.Eventually(t, func() bool {
		count, err = embedded.CountRecords(EmbeddedCountRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
		})
		return err == nil && count == 2
	}, 5*time.Second, 100*time.Millisecond, "CountRecords did not reach expected count after limited metadata delete with response")
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

func TestEmbeddedUpdateCollectionMetadataUsesReplacementSemantics(t *testing.T) {
	embedded := newEmbeddedForIntegrationTest(t)

	databaseName := fmt.Sprintf("update_meta_replace_db_%d", time.Now().UnixNano())
	if err := embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}); err != nil {
		t.Fatalf("CreateDatabase failed: %v", err)
	}

	collectionName := fmt.Sprintf("update_meta_replace_collection_%d", time.Now().UnixNano())
	collection, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
		Metadata: map[string]any{
			"keep": "yes",
			"drop": "remove-me",
		},
		GetOrCreate: true,
	})
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}
	require.Contains(t, collection.Metadata, "drop")

	err = embedded.UpdateCollection(EmbeddedUpdateCollectionRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		NewMetadata: map[string]any{
			"replacement": "yes",
		},
	})
	if err != nil {
		t.Fatalf("UpdateCollection metadata replacement failed: %v", err)
	}

	waitForCollectionConvergence(
		t,
		embedded,
		EmbeddedGetCollectionRequest{
			Name:         collectionName,
			DatabaseName: databaseName,
		},
		"collection metadata replacement",
		func(updated *EmbeddedCollection) (bool, error) {
			if updated.Metadata == nil {
				return false, nil
			}

			replacement, replacementExists := updated.Metadata["replacement"]
			if !replacementExists {
				return false, nil
			}
			replacementValue, replacementIsString := replacement.(string)
			if !replacementIsString {
				return false, fmt.Errorf("expected metadata[replacement] to be string, got %T", replacement)
			}

			_, keepExists := updated.Metadata["keep"]
			_, dropExists := updated.Metadata["drop"]
			return replacementValue == "yes" && !keepExists && !dropExists, nil
		},
	)
}

func TestEmbeddedUpdateCollectionMetadataReplacementWithOverlappingKey(t *testing.T) {
	embedded := newEmbeddedForIntegrationTest(t)

	databaseName := fmt.Sprintf("update_meta_overlap_db_%d", time.Now().UnixNano())
	if err := embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}); err != nil {
		t.Fatalf("CreateDatabase failed: %v", err)
	}

	collectionName := fmt.Sprintf("update_meta_overlap_collection_%d", time.Now().UnixNano())
	collection, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
		Metadata: map[string]any{
			"owner":   "qa",
			"version": "v1",
			"drop":    "remove-me",
		},
		GetOrCreate: true,
	})
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}
	require.Contains(t, collection.Metadata, "owner")
	require.Contains(t, collection.Metadata, "version")
	require.Contains(t, collection.Metadata, "drop")

	err = embedded.UpdateCollection(EmbeddedUpdateCollectionRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		NewMetadata: map[string]any{
			"owner": "platform",
		},
	})
	if err != nil {
		t.Fatalf("UpdateCollection metadata replacement failed: %v", err)
	}

	waitForCollectionConvergence(
		t,
		embedded,
		EmbeddedGetCollectionRequest{
			Name:         collectionName,
			DatabaseName: databaseName,
		},
		"collection metadata overlapping replacement",
		func(updated *EmbeddedCollection) (bool, error) {
			if updated.Metadata == nil {
				return false, nil
			}

			owner, ownerExists := updated.Metadata["owner"]
			if !ownerExists {
				return false, nil
			}
			ownerValue, ownerIsString := owner.(string)
			if !ownerIsString {
				return false, fmt.Errorf("expected metadata[owner] to be string, got %T", owner)
			}

			_, versionExists := updated.Metadata["version"]
			_, dropExists := updated.Metadata["drop"]
			return ownerValue == "platform" && !versionExists && !dropExists, nil
		},
	)
}

func TestEmbeddedUpdateCollectionNameAndMetadataTogether(t *testing.T) {
	embedded := newEmbeddedForIntegrationTest(t)

	databaseName := fmt.Sprintf("update_name_meta_db_%d", time.Now().UnixNano())
	if err := embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}); err != nil {
		t.Fatalf("CreateDatabase failed: %v", err)
	}

	collectionName := fmt.Sprintf("update_name_meta_collection_%d", time.Now().UnixNano())
	collection, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
		Metadata: map[string]any{
			"owner":   "qa",
			"drop":    "remove-me",
			"version": "v1",
		},
		GetOrCreate: true,
	})
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	renamedCollectionName := fmt.Sprintf("%s_renamed", collectionName)
	err = embedded.UpdateCollection(EmbeddedUpdateCollectionRequest{
		CollectionID: collection.ID,
		NewName:      renamedCollectionName,
		DatabaseName: databaseName,
		NewMetadata: map[string]any{
			"owner":    "platform",
			"active":   true,
			"priority": 3,
		},
	})
	if err != nil {
		t.Fatalf("UpdateCollection name+metadata failed: %v", err)
	}

	waitForCollectionConvergence(
		t,
		embedded,
		EmbeddedGetCollectionRequest{
			Name:         renamedCollectionName,
			DatabaseName: databaseName,
		},
		"collection name+metadata replacement",
		func(updated *EmbeddedCollection) (bool, error) {
			if updated.Metadata == nil {
				return false, nil
			}

			owner, ownerExists := updated.Metadata["owner"]
			if !ownerExists {
				return false, nil
			}
			ownerValue, ownerIsString := owner.(string)
			if !ownerIsString {
				return false, fmt.Errorf("expected metadata[owner] to be string, got %T", owner)
			}
			if ownerValue != "platform" {
				return false, nil
			}

			active, activeExists := updated.Metadata["active"]
			if !activeExists {
				return false, nil
			}
			activeValue, activeIsBool := active.(bool)
			if !activeIsBool {
				return false, fmt.Errorf("expected metadata[active] to be bool, got %T", active)
			}
			if !activeValue {
				return false, nil
			}

			priority, priorityExists := updated.Metadata["priority"]
			if !priorityExists {
				return false, nil
			}
			priorityIsThree := false
			switch v := priority.(type) {
			case float64:
				priorityIsThree = v == 3
			case int64:
				priorityIsThree = v == 3
			case int:
				priorityIsThree = v == 3
			default:
				return false, fmt.Errorf("expected metadata[priority] to be numeric, got %T", priority)
			}
			if !priorityIsThree {
				return false, nil
			}

			_, dropExists := updated.Metadata["drop"]
			_, versionExists := updated.Metadata["version"]
			return !dropExists && !versionExists, nil
		},
	)
}
