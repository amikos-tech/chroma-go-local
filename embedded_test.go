package chroma

import (
	"fmt"
	"strings"
	"testing"
	"time"
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
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	var recordCount uint32
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		recordCount, err = embedded.CountRecords(EmbeddedCountRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
		})
		if err == nil && recordCount == 2 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("CountRecords failed: %v", err)
	}
	if recordCount != 2 {
		t.Fatalf("expected 2 records, got %d", recordCount)
	}

	getResp, err := embedded.GetRecords(EmbeddedGetRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-1"},
		Include:      []string{"documents"},
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

	err = embedded.UpdateRecords(EmbeddedUpdateRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-1"},
		Documents:    []string{"first-updated"},
	})
	if err != nil {
		t.Fatalf("UpdateRecords failed: %v", err)
	}

	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		getResp, err = embedded.GetRecords(EmbeddedGetRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
			IDs:          []string{"doc-1"},
			Include:      []string{"documents"},
		})
		if err == nil && len(getResp.Documents) == 1 && getResp.Documents[0] != nil && *getResp.Documents[0] == "first-updated" {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GetRecords after update failed: %v", err)
	}
	if len(getResp.Documents) != 1 || getResp.Documents[0] == nil || *getResp.Documents[0] != "first-updated" {
		t.Fatalf("expected updated document first-updated, got %#v", getResp.Documents)
	}

	err = embedded.UpsertRecords(EmbeddedUpsertRecordsRequest{
		CollectionID: collection.ID,
		DatabaseName: databaseName,
		IDs:          []string{"doc-3"},
		Embeddings:   [][]float32{{0.0, 0.0, 1.0}},
		Documents:    []string{"third"},
	})
	if err != nil {
		t.Fatalf("UpsertRecords failed: %v", err)
	}

	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		recordCount, err = embedded.CountRecords(EmbeddedCountRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
		})
		if err == nil && recordCount == 3 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("CountRecords after upsert failed: %v", err)
	}
	if recordCount != 3 {
		t.Fatalf("expected 3 records after upsert, got %d", recordCount)
	}

	indexingStatus, err := embedded.IndexingStatus(EmbeddedIndexingStatusRequest{
		CollectionID: collection.ID,
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

	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		recordCount, err = embedded.CountRecords(EmbeddedCountRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
		})
		if err == nil && recordCount == 2 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("CountRecords after delete failed: %v", err)
	}
	if recordCount != 2 {
		t.Fatalf("expected 2 records after delete, got %d", recordCount)
	}

	var queryResp *EmbeddedQueryResponse
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		queryResp, err = embedded.Query(EmbeddedQueryRequest{
			CollectionID:    collection.ID,
			DatabaseName:    databaseName,
			QueryEmbeddings: [][]float32{{0.1, 0.2, 0.3}},
			NResults:        1,
		})
		if err == nil && len(queryResp.IDs) > 0 && len(queryResp.IDs[0]) > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if queryResp == nil || len(queryResp.IDs) == 0 || len(queryResp.IDs[0]) == 0 {
		t.Fatal("query returned no ids")
	}
	if queryResp.IDs[0][0] != "doc-1" {
		t.Fatalf("expected top match doc-1, got %q", queryResp.IDs[0][0])
	}

	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		queryResp, err = embedded.Query(EmbeddedQueryRequest{
			CollectionID:    collection.ID,
			DatabaseName:    databaseName,
			QueryEmbeddings: [][]float32{{0.1, 0.2, 0.3}},
			NResults:        1,
			WhereDocument: map[string]any{
				"$contains": "updated",
			},
		})
		if err == nil && len(queryResp.IDs) > 0 && len(queryResp.IDs[0]) > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Query with where_document failed: %v", err)
	}
	if queryResp.IDs[0][0] != "doc-1" {
		t.Fatalf("expected filtered top match doc-1, got %q", queryResp.IDs[0][0])
	}

	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		getResp, err = embedded.GetRecords(EmbeddedGetRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
			WhereDocument: map[string]any{
				"$contains": "updated",
			},
			Include: []string{"documents"},
		})
		if err == nil && len(getResp.IDs) > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GetRecords with where_document failed: %v", err)
	}
	if len(getResp.IDs) == 0 || getResp.IDs[0] != "doc-1" {
		t.Fatalf("expected filtered get to return doc-1, got %#v", getResp.IDs)
	}

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

	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		recordCount, err = embedded.CountRecords(EmbeddedCountRecordsRequest{
			CollectionID: collection.ID,
			DatabaseName: databaseName,
		})
		if err == nil && recordCount == 1 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("CountRecords after filter delete failed: %v", err)
	}
	if recordCount != 1 {
		t.Fatalf("expected 1 record after filter delete, got %d", recordCount)
	}

	if err := embedded.Reset(); err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
}
