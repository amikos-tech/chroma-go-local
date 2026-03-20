package runtime

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeMetadataValueUintOverflow(t *testing.T) {
	_, err := normalizeMetadataValue("metadatas[0].count", uint64(math.MaxUint64), false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds int64 range")
}

func TestNormalizeMetadataValuePointerHandling(t *testing.T) {
	label := "alpha"
	normalized, err := normalizeMetadataValue("metadatas[0].label", &label, false)
	require.NoError(t, err)
	require.Equal(t, "alpha", normalized)

	var nilPtr *string
	_, err = normalizeMetadataValue("metadatas[0].label", nilPtr, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be null")

	normalized, err = normalizeMetadataValue("metadatas[0].label", nilPtr, true)
	require.NoError(t, err)
	require.Nil(t, normalized)
}

func TestNormalizeMetadataSliceRejectsByteSlice(t *testing.T) {
	_, err := normalizeMetadataSlice("metadatas[0].blob", reflect.ValueOf([]byte{1, 2, 3}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported metadata array type")
}

func TestNormalizeMetadataSliceEmptyTypedSlices(t *testing.T) {
	testCases := []struct {
		name      string
		input     any
		assertion func(t *testing.T, value any)
	}{
		{
			name:  "bool",
			input: []bool{},
			assertion: func(t *testing.T, value any) {
				v, ok := value.([]bool)
				require.True(t, ok)
				require.Len(t, v, 0)
			},
		},
		{
			name:  "string",
			input: []string{},
			assertion: func(t *testing.T, value any) {
				v, ok := value.([]string)
				require.True(t, ok)
				require.Len(t, v, 0)
			},
		},
		{
			name:  "int",
			input: []int{},
			assertion: func(t *testing.T, value any) {
				v, ok := value.([]int64)
				require.True(t, ok)
				require.Len(t, v, 0)
			},
		},
		{
			name:  "uint",
			input: []uint64{},
			assertion: func(t *testing.T, value any) {
				v, ok := value.([]int64)
				require.True(t, ok)
				require.Len(t, v, 0)
			},
		},
		{
			name:  "float",
			input: []float64{},
			assertion: func(t *testing.T, value any) {
				v, ok := value.([]metadataFloat64)
				require.True(t, ok)
				require.Len(t, v, 0)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			normalized, err := normalizeMetadataSlice("metadatas[0].value", reflect.ValueOf(tc.input))
			require.NoError(t, err)
			tc.assertion(t, normalized)
		})
	}
}

func TestNormalizeMetadataSlicePromotesIntsToFloats(t *testing.T) {
	normalized, err := normalizeMetadataSlice(
		"metadatas[0].scores",
		reflect.ValueOf([]any{int64(1), float64(2.5), int32(3)}),
	)
	require.NoError(t, err)

	values, ok := normalized.([]metadataFloat64)
	require.True(t, ok)
	require.Equal(t, []metadataFloat64{1, 2.5, 3}, values)

	encoded, err := json.Marshal(values)
	require.NoError(t, err)
	require.Equal(t, "[1.0,2.5,3.0]", string(encoded))
}

func TestNormalizeMetadataSlicePromotesFloatFirstThenInts(t *testing.T) {
	normalized, err := normalizeMetadataSlice(
		"metadatas[0].scores",
		reflect.ValueOf([]any{float64(1.5), int64(2), int32(3)}),
	)
	require.NoError(t, err)

	values, ok := normalized.([]metadataFloat64)
	require.True(t, ok)
	require.Equal(t, []metadataFloat64{1.5, 2, 3}, values)

	encoded, err := json.Marshal(values)
	require.NoError(t, err)
	require.Equal(t, "[1.5,2.0,3.0]", string(encoded))
}

func TestAddRejectsNilMetadataValues(t *testing.T) {
	fakeEmbedded := &Embedded{handle: 1}

	err := fakeEmbedded.Add(EmbeddedAddRequest{
		CollectionID: "c",
		IDs:          []string{"id-1"},
		Embeddings:   makeEmbeddings(1),
		Metadatas: []map[string]any{
			{"nullable": nil},
		},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid metadatas")
	require.Contains(t, err.Error(), "cannot be null")
}

func TestValidateAndNormalizeMetadataNilAndEmpty(t *testing.T) {
	normalizedNil, err := validateAndNormalizeMetadata(nil, false)
	require.NoError(t, err)
	require.Nil(t, normalizedNil)

	normalizedEmpty, err := validateAndNormalizeMetadata(map[string]any{}, false)
	require.NoError(t, err)
	require.NotNil(t, normalizedEmpty)
	require.Len(t, normalizedEmpty, 0)
}

func TestCreateCollectionRejectsInvalidMetadata(t *testing.T) {
	fakeEmbedded := &Embedded{handle: 1}

	_, err := fakeEmbedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name: "collection",
		Metadata: map[string]any{
			"nested": map[string]any{"unsupported": true},
		},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid metadata")
	require.Contains(t, err.Error(), "nested objects are not supported")
}

func TestCreateCollectionRejectsNilMetadataValues(t *testing.T) {
	fakeEmbedded := &Embedded{handle: 1}

	_, err := fakeEmbedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name: "collection",
		Metadata: map[string]any{
			"nullable": nil,
		},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid metadata")
	require.Contains(t, err.Error(), "cannot be null")
}

func TestCreateCollectionWrapsMarshalErrors(t *testing.T) {
	fakeEmbedded := &Embedded{handle: 1}

	_, err := fakeEmbedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name: "collection",
		Configuration: map[string]any{
			"invalid": make(chan int),
		},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "create collection request")
}

func TestUpdateCollectionRequiresNameOrMetadata(t *testing.T) {
	fakeEmbedded := &Embedded{handle: 1}

	err := fakeEmbedded.UpdateCollection(EmbeddedUpdateCollectionRequest{
		CollectionID: "collection-id",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one of new_name or new_metadata is required")
}

func TestUpdateCollectionRejectsEmptyNewMetadata(t *testing.T) {
	fakeEmbedded := &Embedded{handle: 1}

	err := fakeEmbedded.UpdateCollection(EmbeddedUpdateCollectionRequest{
		CollectionID: "collection-id",
		NewMetadata:  map[string]any{},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "new_metadata must not be empty")
}

func TestUpdateCollectionRejectsInvalidNewMetadata(t *testing.T) {
	fakeEmbedded := &Embedded{handle: 1}

	err := fakeEmbedded.UpdateCollection(EmbeddedUpdateCollectionRequest{
		CollectionID: "collection-id",
		NewMetadata: map[string]any{
			"nested": map[string]any{"unsupported": true},
		},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid new_metadata")
	require.Contains(t, err.Error(), "nested objects are not supported")
}

func TestUpdateCollectionRejectsNilNewMetadataValues(t *testing.T) {
	fakeEmbedded := &Embedded{handle: 1}

	err := fakeEmbedded.UpdateCollection(EmbeddedUpdateCollectionRequest{
		CollectionID: "collection-id",
		NewMetadata: map[string]any{
			"deprecated": nil,
		},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid new_metadata")
	require.Contains(t, err.Error(), "cannot be null")
}

func TestUpdateCollectionValidNewMetadataIsSerialized(t *testing.T) {
	originalUpdate := chromaEmbeddedUpdateCollection
	defer func() { chromaEmbeddedUpdateCollection = originalUpdate }()

	var capturedPayload string
	chromaEmbeddedUpdateCollection = func(_ uintptr, requestJSON *byte) int32 {
		capturedPayload = goStringFromPtr(requestJSON)
		return Success
	}

	embedded := &Embedded{handle: 1}
	err := embedded.UpdateCollection(EmbeddedUpdateCollectionRequest{
		CollectionID: "00000000-0000-0000-0000-000000000001",
		NewMetadata: map[string]any{
			"owner":  "qa",
			"score":  1.0,
			"levels": []int{1, 2},
		},
	})
	require.NoError(t, err)

	var payload struct {
		NewMetadata map[string]any `json:"new_metadata"`
	}
	require.NoError(t, json.Unmarshal([]byte(capturedPayload), &payload))
	require.NotNil(t, payload.NewMetadata)

	require.Equal(t, "qa", payload.NewMetadata["owner"])
	score, ok := payload.NewMetadata["score"].(float64)
	require.True(t, ok)
	require.Equal(t, 1.0, score)

	levelsRaw, ok := payload.NewMetadata["levels"]
	require.True(t, ok)
	levels, ok := levelsRaw.([]any)
	require.True(t, ok)
	require.Equal(t, []any{1.0, 2.0}, levels)
}

func TestCreateCollectionValidMetadataIsSerialized(t *testing.T) {
	originalCreate := chromaEmbeddedCreateCollection
	originalStringFree := chromaStringFree
	defer func() {
		chromaEmbeddedCreateCollection = originalCreate
		chromaStringFree = originalStringFree
	}()

	response := []byte("{\"id\":\"col-1\",\"name\":\"collection\",\"tenant\":\"default_tenant\",\"database\":\"default_database\"}\x00")
	var capturedPayload string
	chromaEmbeddedCreateCollection = func(_ uintptr, requestJSON *byte) *byte {
		capturedPayload = goStringFromPtr(requestJSON)
		return &response[0]
	}
	chromaStringFree = func(*byte) {}

	embedded := &Embedded{handle: 1}
	_, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name: "collection",
		Metadata: map[string]any{
			"score":  1.0,
			"labels": []string{"alpha", "beta"},
			"levels": []int{1, 2},
		},
	})
	require.NoError(t, err)

	var payload struct {
		Metadata map[string]any `json:"metadata"`
	}
	require.NoError(t, json.Unmarshal([]byte(capturedPayload), &payload))
	require.NotNil(t, payload.Metadata)

	score, ok := payload.Metadata["score"].(float64)
	require.True(t, ok)
	require.Equal(t, 1.0, score)

	labelsRaw, ok := payload.Metadata["labels"]
	require.True(t, ok)
	labels, ok := labelsRaw.([]any)
	require.True(t, ok)
	require.Equal(t, []any{"alpha", "beta"}, labels)

	levelsRaw, ok := payload.Metadata["levels"]
	require.True(t, ok)
	levels, ok := levelsRaw.([]any)
	require.True(t, ok)
	require.Len(t, levels, 2)
	level0, ok := levels[0].(float64)
	require.True(t, ok)
	level1, ok := levels[1].(float64)
	require.True(t, ok)
	require.Equal(t, 1.0, level0)
	require.Equal(t, 2.0, level1)
}

func TestEmbeddedCollectionUnmarshalWithMetadataConfigurationAndSchema(t *testing.T) {
	raw := []byte(`{
		"id":"col-1",
		"name":"collection",
		"tenant":"tenant-1",
		"database":"db-1",
		"metadata":{"owner":"qa","priority":1},
		"configuration_json":{"hnsw":{"space":"cosine"}},
		"schema":{"defaults":{"float_list":{"vector_index":{"config":{"space":"cosine"}}}}}
	}`)

	var collection EmbeddedCollection
	err := json.Unmarshal(raw, &collection)
	require.NoError(t, err)
	require.Equal(t, "col-1", collection.ID)
	require.Equal(t, "collection", collection.Name)
	require.Equal(t, "tenant-1", collection.Tenant)
	require.Equal(t, "db-1", collection.Database)

	require.NotNil(t, collection.Metadata)
	require.Equal(t, "qa", collection.Metadata["owner"])
	priority, ok := collection.Metadata["priority"].(float64)
	require.True(t, ok)
	require.Equal(t, 1.0, priority)

	require.NotNil(t, collection.ConfigurationJSON)
	hnswRaw, ok := collection.ConfigurationJSON["hnsw"]
	require.True(t, ok)
	hnsw, ok := hnswRaw.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "cosine", hnsw["space"])

	require.NotNil(t, collection.Schema)
	defaultsRaw, ok := collection.Schema["defaults"]
	require.True(t, ok)
	defaults, ok := defaultsRaw.(map[string]any)
	require.True(t, ok)
	require.NotNil(t, defaults)
}

func TestNormalizeMetadataValueRejectsNaNAndInf(t *testing.T) {
	_, err := normalizeMetadataValue("metadatas[0].score", math.NaN(), false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be finite")

	_, err = normalizeMetadataValue("metadatas[0].score", math.Inf(1), false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be finite")
}

func TestMetadataFloat64MarshalRejectsNaNAndInf(t *testing.T) {
	_, err := json.Marshal(metadataFloat64(math.NaN()))
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be finite")

	_, err = json.Marshal(metadataFloat64(math.Inf(-1)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be finite")

	encoded, err := json.Marshal(metadataFloat64(1))
	require.NoError(t, err)
	require.Equal(t, "1.0", string(encoded))
}
