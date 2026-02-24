package chroma

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEmbeddedModeBasicFlow(t *testing.T) {
	if err := Init(""); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath("./chroma_test_data_embedded"),
		WithEmbeddedAllowReset(true),
	)
	if err != nil {
		t.Fatalf("Failed to start embedded mode: %v", err)
	}
	defer func() { _ = embedded.Close() }()

	heartbeat, err := embedded.Heartbeat()
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}
	if heartbeat == 0 {
		t.Fatal("heartbeat should not be zero")
	}

	maxBatchSize, err := embedded.MaxBatchSize()
	if err != nil {
		t.Fatalf("MaxBatchSize failed: %v", err)
	}
	if maxBatchSize == 0 {
		t.Fatal("max batch size should be greater than zero")
	}

	healthcheck, err := embedded.Healthcheck()
	if err != nil {
		t.Fatalf("Healthcheck failed: %v", err)
	}
	if healthcheck == nil {
		t.Fatal("healthcheck should not be nil")
	}

	tenantName := fmt.Sprintf("tenant_%d", time.Now().UnixNano())
	if err := embedded.CreateTenant(EmbeddedCreateTenantRequest{Name: tenantName}); err != nil {
		t.Fatalf("CreateTenant failed: %v", err)
	}

	tenant, err := embedded.GetTenant(EmbeddedGetTenantRequest{Name: tenantName})
	if err != nil {
		t.Fatalf("GetTenant failed: %v", err)
	}
	if tenant.Name != tenantName {
		t.Fatalf("expected tenant %q, got %q", tenantName, tenant.Name)
	}

	resourceName := fmt.Sprintf("resource_%d", time.Now().UnixNano())
	if err := embedded.UpdateTenant(EmbeddedUpdateTenantRequest{
		TenantID:     tenantName,
		ResourceName: resourceName,
	}); err != nil {
		t.Fatalf("UpdateTenant failed: %v", err)
	}

	tenant, err = embedded.GetTenant(EmbeddedGetTenantRequest{Name: tenantName})
	if err != nil {
		t.Fatalf("GetTenant after update failed: %v", err)
	}
	if tenant.Name != tenantName {
		t.Fatalf("expected tenant %q after update, got %q", tenantName, tenant.Name)
	}
	if tenant.ResourceName != nil && *tenant.ResourceName != resourceName {
		t.Fatalf("expected tenant resource_name %q when present, got %#v", resourceName, tenant.ResourceName)
	}

	databaseName := fmt.Sprintf("db_%d", time.Now().UnixNano())
	if err := embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: databaseName}); err != nil {
		t.Fatalf("CreateDatabase failed: %v", err)
	}

	db, err := embedded.GetDatabase(EmbeddedGetDatabaseRequest{Name: databaseName})
	if err != nil {
		t.Fatalf("GetDatabase failed: %v", err)
	}
	if db.Name != databaseName {
		t.Fatalf("expected database %q, got %q", databaseName, db.Name)
	}

	databases, err := embedded.ListDatabases(EmbeddedListDatabasesRequest{})
	if err != nil {
		t.Fatalf("ListDatabases failed: %v", err)
	}
	foundDB := false
	for _, item := range databases {
		if item.Name == databaseName {
			foundDB = true
			break
		}
	}
	if !foundDB {
		t.Fatalf("expected database %q in list", databaseName)
	}

	deleteDBName := fmt.Sprintf("db_del_%d", time.Now().UnixNano())
	if err := embedded.CreateDatabase(EmbeddedCreateDatabaseRequest{Name: deleteDBName}); err != nil {
		t.Fatalf("CreateDatabase (delete target) failed: %v", err)
	}
	if err := embedded.DeleteDatabase(EmbeddedDeleteDatabaseRequest{Name: deleteDBName}); err != nil {
		t.Fatalf("DeleteDatabase failed: %v", err)
	}

	collectionName := fmt.Sprintf("embedded_test_%d", time.Now().UnixNano())
	collection, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
		GetOrCreate:  true,
	})
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}
	if collection.ID == "" {
		t.Fatal("expected non-empty collection id")
	}

	collections, err := embedded.ListCollections(EmbeddedListCollectionsRequest{
		DatabaseName: databaseName,
	})
	if err != nil {
		t.Fatalf("ListCollections failed: %v", err)
	}
	foundCollection := false
	for _, item := range collections {
		if item.ID == collection.ID {
			foundCollection = true
			break
		}
	}
	if !foundCollection {
		t.Fatalf("expected collection %q in list", collection.ID)
	}

	gotCollection, err := embedded.GetCollection(EmbeddedGetCollectionRequest{
		Name:         collectionName,
		DatabaseName: databaseName,
	})
	if err != nil {
		t.Fatalf("GetCollection failed: %v", err)
	}
	if gotCollection.ID != collection.ID {
		t.Fatalf("expected collection id %q, got %q", collection.ID, gotCollection.ID)
	}

	renamedCollectionName := fmt.Sprintf("%s_renamed", collectionName)
	if err := embedded.UpdateCollection(EmbeddedUpdateCollectionRequest{
		CollectionID: collection.ID,
		NewName:      renamedCollectionName,
		DatabaseName: databaseName,
	}); err != nil {
		t.Fatalf("UpdateCollection failed: %v", err)
	}

	renamedCollection, err := embedded.GetCollection(EmbeddedGetCollectionRequest{
		Name:         renamedCollectionName,
		DatabaseName: databaseName,
	})
	if err != nil {
		t.Fatalf("GetCollection after update failed: %v", err)
	}
	if renamedCollection.ID != collection.ID {
		t.Fatalf("expected renamed collection id %q, got %q", collection.ID, renamedCollection.ID)
	}

	forkName := fmt.Sprintf("%s_fork", collectionName)
	forkedCollection, err := embedded.ForkCollection(EmbeddedForkCollectionRequest{
		SourceCollectionID:   collection.ID,
		TargetCollectionName: forkName,
		DatabaseName:         databaseName,
	})
	if err != nil {
		lower := strings.ToLower(err.Error())
		if !strings.Contains(lower, "unimplemented") && !strings.Contains(lower, "unsupported") && !strings.Contains(lower, "local chroma") {
			t.Fatalf("ForkCollection failed unexpectedly: %v", err)
		}
	} else {
		if forkedCollection.ID == "" {
			t.Fatal("fork returned empty collection id")
		}
		if forkedCollection.ID == collection.ID {
			t.Fatalf("expected forked collection id to differ from source %q", collection.ID)
		}
	}

	deleteCollectionName := fmt.Sprintf("%s_delete", collectionName)
	_, err = embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:         deleteCollectionName,
		DatabaseName: databaseName,
		GetOrCreate:  true,
	})
	if err != nil {
		t.Fatalf("CreateCollection for delete failed: %v", err)
	}
	if err := embedded.DeleteCollection(EmbeddedDeleteCollectionRequest{
		Name:         deleteCollectionName,
		DatabaseName: databaseName,
	}); err != nil {
		t.Fatalf("DeleteCollection failed: %v", err)
	}
	if _, err := embedded.GetCollection(EmbeddedGetCollectionRequest{
		Name:         deleteCollectionName,
		DatabaseName: databaseName,
	}); err == nil {
		t.Fatal("expected deleted collection lookup to fail")
	}

	count, err := embedded.CountCollections(EmbeddedCountCollectionsRequest{
		DatabaseName: databaseName,
	})
	if err != nil {
		t.Fatalf("CountCollections failed: %v", err)
	}
	if count == 0 {
		t.Fatal("expected collection count > 0")
	}

	err = embedded.Add(EmbeddedAddRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-1", "doc-2"},
		Embeddings: [][]float32{
			{0.1, 0.2, 0.3},
			{0.9, 0.1, 0.1},
		},
		Documents: []string{"first", "second"},
		Metadatas: []map[string]any{
			{
				"labels": []string{"alpha", "beta"},
				"scores": []float64{1.1, 2.2},
				"flags":  []bool{true, false},
			},
			{
				"labels": []string{"beta", "gamma"},
				"scores": []float64{3.3, 4.4},
				"flags":  []bool{false, true},
			},
		},
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	var recordCount uint32
	require.Eventually(t, func() bool {
		recordCount, err = embedded.CountRecords(EmbeddedCountRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
		})
		return err == nil && recordCount == 2
	}, 5*time.Second, 200*time.Millisecond, "CountRecords did not reach expected value after add")

	getResp, err := embedded.GetRecords(EmbeddedGetRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-1"},
		Include:      []string{"documents", "metadatas"},
	})
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}
	if len(getResp.IDs) != 1 || getResp.IDs[0] != "doc-1" {
		t.Fatalf("expected get ids [doc-1], got %#v", getResp.IDs)
	}
	if len(getResp.Documents) != 1 || getResp.Documents[0] == nil || *getResp.Documents[0] != "first" {
		t.Fatalf("expected document first, got %#v", getResp.Documents)
	}
	if len(getResp.Metadatas) != 1 {
		t.Fatalf("expected one metadata entry, got %#v", getResp.Metadatas)
	}
	labelsRaw, ok := getResp.Metadatas[0]["labels"]
	if !ok {
		t.Fatalf("expected metadata labels key, got %#v", getResp.Metadatas[0])
	}
	labels, ok := labelsRaw.([]any)
	if !ok {
		t.Fatalf("expected labels to decode as []any, got %T", labelsRaw)
	}
	require.Contains(t, labels, "alpha")
	require.Contains(t, labels, "beta")
	scoresRaw, ok := getResp.Metadatas[0]["scores"]
	if !ok {
		t.Fatalf("expected metadata scores key, got %#v", getResp.Metadatas[0])
	}
	scores, ok := scoresRaw.([]any)
	if !ok {
		t.Fatalf("expected scores to decode as []any, got %T", scoresRaw)
	}
	require.Len(t, scores, 2)
	score0, ok := scores[0].(float64)
	if !ok {
		t.Fatalf("expected score[0] to decode as float64, got %T", scores[0])
	}
	score1, ok := scores[1].(float64)
	if !ok {
		t.Fatalf("expected score[1] to decode as float64, got %T", scores[1])
	}
	require.InDelta(t, 1.1, score0, 1e-9)
	require.InDelta(t, 2.2, score1, 1e-9)
	flagsRaw, ok := getResp.Metadatas[0]["flags"]
	if !ok {
		t.Fatalf("expected metadata flags key, got %#v", getResp.Metadatas[0])
	}
	flags, ok := flagsRaw.([]any)
	if !ok {
		t.Fatalf("expected flags to decode as []any, got %T", flagsRaw)
	}
	require.Equal(t, []any{true, false}, flags)

	err = embedded.UpdateRecords(EmbeddedUpdateRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-1"},
		Documents:    []string{"first-updated"},
		Metadatas: []map[string]any{
			{
				"labels": []string{"alpha", "updated"},
				"flags":  []bool{true, true},
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateRecords failed: %v", err)
	}

	require.Eventually(t, func() bool {
		getResp, err = embedded.GetRecords(EmbeddedGetRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
			IDs:          []string{"doc-1"},
			Include:      []string{"documents", "metadatas"},
		})
		if err != nil || len(getResp.Documents) != 1 || getResp.Documents[0] == nil || *getResp.Documents[0] != "first-updated" {
			return false
		}
		if len(getResp.Metadatas) != 1 {
			return false
		}
		labelsRaw, ok := getResp.Metadatas[0]["labels"]
		if !ok {
			return false
		}
		labels, ok := labelsRaw.([]any)
		if !ok {
			return false
		}
		if len(labels) != 2 || labels[0] != "alpha" || labels[1] != "updated" {
			return false
		}
		flagsRaw, ok := getResp.Metadatas[0]["flags"]
		if !ok {
			return false
		}
		flags, ok := flagsRaw.([]any)
		if !ok || len(flags) != 2 {
			return false
		}
		return flags[0] == true && flags[1] == true
	}, 5*time.Second, 200*time.Millisecond, "GetRecords did not return updated document")

	err = embedded.UpdateRecords(EmbeddedUpdateRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-1"},
		Metadatas: []map[string]any{
			{
				"labels": []string{"alpha", "updated"},
				"scores": []float64{9.9, 10.1},
				"flags":  []bool{false, true},
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateRecords metadatas-only failed: %v", err)
	}

	require.Eventually(t, func() bool {
		getResp, err = embedded.GetRecords(EmbeddedGetRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
			IDs:          []string{"doc-1"},
			Include:      []string{"documents", "metadatas"},
		})
		if err != nil || len(getResp.Documents) != 1 || getResp.Documents[0] == nil || *getResp.Documents[0] != "first-updated" {
			return false
		}
		if len(getResp.Metadatas) != 1 {
			return false
		}
		scoresRaw, ok := getResp.Metadatas[0]["scores"]
		if !ok {
			return false
		}
		scores, ok := scoresRaw.([]any)
		if !ok || len(scores) != 2 {
			return false
		}
		score0, ok := scores[0].(float64)
		if !ok {
			return false
		}
		score1, ok := scores[1].(float64)
		if !ok {
			return false
		}
		flagsRaw, ok := getResp.Metadatas[0]["flags"]
		if !ok {
			return false
		}
		flags, ok := flagsRaw.([]any)
		if !ok || len(flags) != 2 {
			return false
		}
		return math.Abs(score0-9.9) < 1e-9 && math.Abs(score1-10.1) < 1e-9 && flags[0] == false && flags[1] == true
	}, 5*time.Second, 200*time.Millisecond, "GetRecords did not return metadatas-only update")

	err = embedded.UpsertRecords(EmbeddedUpsertRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-3"},
		Embeddings:   [][]float32{{0.0, 0.0, 1.0}},
		Documents:    []string{"third"},
		Metadatas: []map[string]any{
			{
				"labels": []string{"third", "delta"},
				"scores": []float64{7.7, 8.8},
				"flags":  []bool{true, false},
			},
		},
	})
	if err != nil {
		t.Fatalf("UpsertRecords failed: %v", err)
	}

	require.Eventually(t, func() bool {
		recordCount, err = embedded.CountRecords(EmbeddedCountRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
		})
		return err == nil && recordCount == 3
	}, 5*time.Second, 200*time.Millisecond, "CountRecords did not reach expected value after upsert")

	require.Eventually(t, func() bool {
		getResp, err = embedded.GetRecords(EmbeddedGetRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
			IDs:          []string{"doc-3"},
			Include:      []string{"documents", "metadatas"},
		})
		if err != nil || len(getResp.IDs) != 1 || getResp.IDs[0] != "doc-3" {
			return false
		}
		if len(getResp.Documents) != 1 || getResp.Documents[0] == nil || *getResp.Documents[0] != "third" {
			return false
		}
		if len(getResp.Metadatas) != 1 {
			return false
		}
		labelsRaw, ok := getResp.Metadatas[0]["labels"]
		if !ok {
			return false
		}
		labels, ok := labelsRaw.([]any)
		if !ok || len(labels) != 2 || labels[0] != "third" || labels[1] != "delta" {
			return false
		}
		scoresRaw, ok := getResp.Metadatas[0]["scores"]
		if !ok {
			return false
		}
		scores, ok := scoresRaw.([]any)
		if !ok || len(scores) != 2 {
			return false
		}
		score0, ok := scores[0].(float64)
		if !ok {
			return false
		}
		score1, ok := scores[1].(float64)
		if !ok {
			return false
		}
		flagsRaw, ok := getResp.Metadatas[0]["flags"]
		if !ok {
			return false
		}
		flags, ok := flagsRaw.([]any)
		if !ok || len(flags) != 2 {
			return false
		}
		return math.Abs(score0-7.7) < 1e-9 && math.Abs(score1-8.8) < 1e-9 && flags[0] == true && flags[1] == false
	}, 5*time.Second, 200*time.Millisecond, "Upsert metadata round-trip did not match expected values")

	indexingStatus, err := embedded.IndexingStatus(EmbeddedIndexingStatusRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
	})
	if err != nil {
		lower := strings.ToLower(err.Error())
		if !strings.Contains(lower, "unimplemented") &&
			!strings.Contains(lower, "not implemented") &&
			!strings.Contains(lower, "unsupported") {
			t.Fatalf("IndexingStatus failed unexpectedly: %v", err)
		}
	} else if indexingStatus.TotalOps < indexingStatus.NumIndexedOps {
		t.Fatalf("expected total_ops >= num_indexed_ops, got total=%d indexed=%d", indexingStatus.TotalOps, indexingStatus.NumIndexedOps)
	}

	err = embedded.DeleteRecords(EmbeddedDeleteRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-2"},
	})
	if err != nil {
		t.Fatalf("DeleteRecords failed: %v", err)
	}

	require.Eventually(t, func() bool {
		recordCount, err = embedded.CountRecords(EmbeddedCountRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
		})
		return err == nil && recordCount == 2
	}, 5*time.Second, 200*time.Millisecond, "CountRecords did not reach expected value after delete")

	var queryResp *EmbeddedQueryResponse
	require.Eventually(t, func() bool {
		queryResp, err = embedded.Query(EmbeddedQueryRequest{
			CollectionID:    collection.ID,
			DatabaseName:    databaseName,
			QueryEmbeddings: [][]float32{{0.1, 0.2, 0.3}},
			NResults:        1,
		})
		return err == nil && queryResp != nil && len(queryResp.IDs) > 0 && len(queryResp.IDs[0]) > 0
	}, 5*time.Second, 200*time.Millisecond, "Query did not return IDs")
	require.Equal(t, "doc-1", queryResp.IDs[0][0])

	require.Eventually(t, func() bool {
		queryResp, err = embedded.Query(EmbeddedQueryRequest{
			CollectionID:    collection.ID,
			DatabaseName:    databaseName,
			QueryEmbeddings: [][]float32{{0.1, 0.2, 0.3}},
			NResults:        1,
			Where: map[string]any{
				"labels": map[string]any{
					"$contains": "updated",
				},
			},
		})
		return err == nil && queryResp != nil && len(queryResp.IDs) > 0 && len(queryResp.IDs[0]) > 0
	}, 5*time.Second, 200*time.Millisecond, "Query with metadata array contains filter did not return IDs")
	require.Equal(t, "doc-1", queryResp.IDs[0][0])

	require.Eventually(t, func() bool {
		queryResp, err = embedded.Query(EmbeddedQueryRequest{
			CollectionID:    collection.ID,
			DatabaseName:    databaseName,
			QueryEmbeddings: [][]float32{{0.1, 0.2, 0.3}},
			NResults:        1,
			WhereDocument: map[string]any{
				"$contains": "updated",
			},
		})
		return err == nil && queryResp != nil && len(queryResp.IDs) > 0 && len(queryResp.IDs[0]) > 0
	}, 5*time.Second, 200*time.Millisecond, "Query with where_document did not return IDs")
	require.Equal(t, "doc-1", queryResp.IDs[0][0])

	require.Eventually(t, func() bool {
		getResp, err = embedded.GetRecords(EmbeddedGetRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
			WhereDocument: map[string]any{
				"$contains": "updated",
			},
			Include: []string{"documents"},
		})
		return err == nil && len(getResp.IDs) > 0
	}, 5*time.Second, 200*time.Millisecond, "GetRecords with where_document did not return IDs")
	require.Equal(t, "doc-1", getResp.IDs[0])

	err = embedded.DeleteRecords(EmbeddedDeleteRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		WhereDocument: map[string]any{
			"$contains": "third",
		},
	})
	if err != nil {
		t.Fatalf("DeleteRecords with where_document failed: %v", err)
	}

	require.Eventually(t, func() bool {
		recordCount, err = embedded.CountRecords(EmbeddedCountRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
		})
		return err == nil && recordCount == 1
	}, 5*time.Second, 200*time.Millisecond, "CountRecords did not reach expected value after filter delete")

	if err := embedded.Reset(); err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
}
