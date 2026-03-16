package chroma

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func makeIDs(n int) []string {
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ids[i] = "id-" + string(rune('a'+i))
	}
	return ids
}

func makeEmbeddings(n int) [][]float32 {
	embeddings := make([][]float32, n)
	for i := 0; i < n; i++ {
		embeddings[i] = []float32{0.1, 0.2, 0.3}
	}
	return embeddings
}

func makeDocs(n int) []string {
	docs := make([]string, n)
	for i := 0; i < n; i++ {
		docs[i] = "doc-" + string(rune('a'+i))
	}
	return docs
}

func makeURIs(n int) []string {
	uris := make([]string, n)
	for i := 0; i < n; i++ {
		uris[i] = "uri-" + string(rune('a'+i))
	}
	return uris
}

func makeMetadatas(n int) []map[string]any {
	metadatas := make([]map[string]any, n)
	for i := 0; i < n; i++ {
		metadatas[i] = map[string]any{"i": i}
	}
	return metadatas
}

func TestEmbeddedValidationProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Non-zero handle bypasses the nil-handle check; validation fires before FFI.
	fakeEmbedded := &Embedded{handle: 1}

	properties.Property("Add rejects mismatched ids/embeddings lengths", prop.ForAll(
		func(idsLen uint8, embLen uint8) bool {
			if idsLen == 0 || embLen == 0 || idsLen == embLen {
				return true
			}
			err := fakeEmbedded.Add(EmbeddedAddRequest{
				CollectionID: "c",
				IDs:          makeIDs(int(idsLen)),
				Embeddings:   makeEmbeddings(int(embLen)),
			})
			return err != nil && strings.Contains(err.Error(), "same length")
		},
		gen.UInt8Range(0, 10),
		gen.UInt8Range(0, 10),
	))

	properties.Property("Add rejects mismatched ids/metadatas lengths", prop.ForAll(
		func(idsLen uint8, mdLen uint8) bool {
			if idsLen == 0 || mdLen == 0 || idsLen == mdLen {
				return true
			}
			err := fakeEmbedded.Add(EmbeddedAddRequest{
				CollectionID: "c",
				IDs:          makeIDs(int(idsLen)),
				Embeddings:   makeEmbeddings(int(idsLen)),
				Metadatas:    makeMetadatas(int(mdLen)),
			})
			return err != nil && strings.Contains(err.Error(), "metadatas must have same length")
		},
		gen.UInt8Range(0, 10),
		gen.UInt8Range(0, 10),
	))

	properties.Property("Upsert rejects mismatched ids/embeddings lengths", prop.ForAll(
		func(idsLen uint8, embLen uint8) bool {
			if idsLen == 0 || embLen == 0 || idsLen == embLen {
				return true
			}
			err := fakeEmbedded.UpsertRecords(EmbeddedUpsertRecordsRequest{
				CollectionID: "c",
				IDs:          makeIDs(int(idsLen)),
				Embeddings:   makeEmbeddings(int(embLen)),
			})
			return err != nil && strings.Contains(err.Error(), "same length")
		},
		gen.UInt8Range(0, 10),
		gen.UInt8Range(0, 10),
	))

	properties.Property("Upsert rejects mismatched ids/metadatas lengths", prop.ForAll(
		func(idsLen uint8, mdLen uint8) bool {
			if idsLen == 0 || mdLen == 0 || idsLen == mdLen {
				return true
			}
			err := fakeEmbedded.UpsertRecords(EmbeddedUpsertRecordsRequest{
				CollectionID: "c",
				IDs:          makeIDs(int(idsLen)),
				Embeddings:   makeEmbeddings(int(idsLen)),
				Metadatas:    makeMetadatas(int(mdLen)),
			})
			return err != nil && strings.Contains(err.Error(), "metadatas must have same length")
		},
		gen.UInt8Range(0, 10),
		gen.UInt8Range(0, 10),
	))

	properties.Property("Update rejects empty payload mutations", prop.ForAll(
		func(idsLen uint8) bool {
			if idsLen == 0 {
				return true
			}
			err := fakeEmbedded.UpdateRecords(EmbeddedUpdateRecordsRequest{
				CollectionID: "c",
				IDs:          makeIDs(int(idsLen)),
			})
			return err != nil && strings.Contains(err.Error(), "at least one of embeddings, documents, uris, or metadatas")
		},
		gen.UInt8Range(0, 10),
	))

	properties.Property("Update rejects document length mismatch", prop.ForAll(
		func(idsLen uint8, docsLen uint8) bool {
			if idsLen == 0 || docsLen == 0 || idsLen == docsLen {
				return true
			}
			err := fakeEmbedded.UpdateRecords(EmbeddedUpdateRecordsRequest{
				CollectionID: "c",
				IDs:          makeIDs(int(idsLen)),
				Documents:    makeDocs(int(docsLen)),
			})
			return err != nil && strings.Contains(err.Error(), "documents must have same length")
		},
		gen.UInt8Range(0, 10),
		gen.UInt8Range(0, 10),
	))

	properties.Property("Update rejects uri length mismatch", prop.ForAll(
		func(idsLen uint8, urisLen uint8) bool {
			if idsLen == 0 || urisLen == 0 || idsLen == urisLen {
				return true
			}
			err := fakeEmbedded.UpdateRecords(EmbeddedUpdateRecordsRequest{
				CollectionID: "c",
				IDs:          makeIDs(int(idsLen)),
				URIs:         makeURIs(int(urisLen)),
			})
			return err != nil && strings.Contains(err.Error(), "uris must have same length")
		},
		gen.UInt8Range(0, 10),
		gen.UInt8Range(0, 10),
	))

	properties.Property("Update rejects metadata length mismatch", prop.ForAll(
		func(idsLen uint8, mdLen uint8) bool {
			if idsLen == 0 || mdLen == 0 || idsLen == mdLen {
				return true
			}
			err := fakeEmbedded.UpdateRecords(EmbeddedUpdateRecordsRequest{
				CollectionID: "c",
				IDs:          makeIDs(int(idsLen)),
				Metadatas:    makeMetadatas(int(mdLen)),
			})
			return err != nil && strings.Contains(err.Error(), "metadatas must have same length")
		},
		gen.UInt8Range(0, 10),
		gen.UInt8Range(0, 10),
	))

	properties.Property("Add rejects nested metadata objects", prop.ForAll(
		func(key string, value string) bool {
			if strings.TrimSpace(key) == "" {
				key = "nested"
			}
			err := fakeEmbedded.Add(EmbeddedAddRequest{
				CollectionID: "c",
				IDs:          []string{"id-1"},
				Embeddings:   makeEmbeddings(1),
				Metadatas: []map[string]any{
					{
						key: map[string]any{"value": value},
					},
				},
			})
			return err != nil && strings.Contains(err.Error(), "nested objects are not supported")
		},
		gen.AnyString(),
		gen.AnyString(),
	))

	properties.Property("Add rejects heterogeneous metadata arrays", prop.ForAll(
		func(i int64, s string) bool {
			err := fakeEmbedded.Add(EmbeddedAddRequest{
				CollectionID: "c",
				IDs:          []string{"id-1"},
				Embeddings:   makeEmbeddings(1),
				Metadatas: []map[string]any{
					{
						"mixed": []any{i, s},
					},
				},
			})
			return err != nil && strings.Contains(err.Error(), "homogeneous array")
		},
		gen.Int64(),
		gen.AnyString(),
	))

	properties.Property("Metadata normalization encodes float values with decimal or exponent", prop.ForAll(
		func(f float64) bool {
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return true
			}
			normalized, err := validateAndNormalizeMetadatas([]map[string]any{
				{"score": f},
			}, false)
			if err != nil {
				return false
			}
			encoded, err := json.Marshal(normalized)
			if err != nil {
				return false
			}
			jsonStr := string(encoded)
			return strings.ContainsAny(jsonStr, ".eE")
		},
		gen.Float64Range(-1e6, 1e6),
	))

	properties.Property("Create metadata normalization encodes float values with decimal or exponent", prop.ForAll(
		func(f float64) bool {
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return true
			}
			normalized, err := validateAndNormalizeMetadata(map[string]any{
				"score": f,
			}, false)
			if err != nil {
				return false
			}
			encoded, err := json.Marshal(normalized)
			if err != nil {
				return false
			}
			jsonStr := string(encoded)
			return strings.ContainsAny(jsonStr, ".eE")
		},
		gen.Float64Range(-1e6, 1e6),
	))

	properties.Property("Record metadata normalization allows nil values for update/upsert payloads", prop.ForAll(
		func(key string) bool {
			if strings.TrimSpace(key) == "" {
				key = "k"
			}
			_, err := validateAndNormalizeMetadatas([]map[string]any{
				{key: nil},
			}, true)
			return err == nil
		},
		gen.AnyString(),
	))

	properties.Property("Collection metadata normalization rejects nil values", prop.ForAll(
		func(key string) bool {
			if strings.TrimSpace(key) == "" {
				key = "k"
			}
			_, err := validateAndNormalizeMetadata(map[string]any{
				key: nil,
			}, false)
			return err != nil && strings.Contains(err.Error(), "cannot be null")
		},
		gen.AnyString(),
	))

	properties.Property("DeleteRecords rejects requests without ids/where/where_document", prop.ForAll(
		func(collectionID string) bool {
			err := fakeEmbedded.DeleteRecords(EmbeddedDeleteRecordsRequest{
				CollectionID: collectionID,
			})
			if strings.TrimSpace(collectionID) == "" {
				return err != nil && strings.Contains(err.Error(), "collection_id is required")
			}
			return err != nil && strings.Contains(err.Error(), "at least one of ids, where, or where_document")
		},
		gen.AnyString(),
	))

	properties.Property("DeleteRecords rejects limit without where/where_document", prop.ForAll(
		func(collectionID string, limit uint32) bool {
			if limit == 0 {
				limit = 1
			}
			err := fakeEmbedded.DeleteRecords(EmbeddedDeleteRecordsRequest{
				CollectionID: collectionID,
				IDs:          []string{"id-a"},
				Limit:        &limit,
			})
			if strings.TrimSpace(collectionID) == "" {
				return err != nil && strings.Contains(err.Error(), "collection_id is required")
			}
			return err != nil && strings.Contains(err.Error(), deleteRecordsLimitRequiresFilterErr)
		},
		gen.AnyString(),
		gen.UInt32(),
	))

	properties.TestingRun(t)
}

func TestDeleteRecordsRejectsZeroLimit(t *testing.T) {
	// Non-zero handle bypasses the nil-handle check; validation fires before FFI.
	fakeEmbedded := &Embedded{handle: 1}
	limit := uint32(0)

	err := fakeEmbedded.DeleteRecords(EmbeddedDeleteRecordsRequest{
		CollectionID: "collection-id",
		Where: map[string]any{
			"status": "stale",
		},
		Limit: &limit,
	})

	if err == nil || !strings.Contains(err.Error(), deleteRecordsLimitMustBePositiveErr) {
		t.Fatalf("expected zero limit to be rejected with %q, got %v", deleteRecordsLimitMustBePositiveErr, err)
	}
}

func TestCStringRoundTripProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("goStringFromPtr reads up to first null byte", prop.ForAll(
		func(s string) bool {
			bytes := cStringFromGo(s)
			got := goStringFromPtr(&bytes[0])

			cut := strings.IndexByte(s, 0)
			want := s
			if cut >= 0 {
				want = s[:cut]
			}
			return got == want
		},
		gen.AnyString(),
	))

	properties.TestingRun(t)
}
