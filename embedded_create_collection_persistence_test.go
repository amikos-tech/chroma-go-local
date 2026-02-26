package chroma

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEmbeddedCreateCollectionPersistsMetadataAndConfigurationAcrossRestart(t *testing.T) {
	require.NoError(t, Init(""))

	persistPath := filepath.Join(t.TempDir(), "embedded-persist")
	collectionName := fmt.Sprintf("persisted_collection_%d", time.Now().UnixNano())
	expectedMetadata := map[string]any{
		"owner":  "integration",
		"active": true,
	}
	expectedSpace := "cosine"

	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath(persistPath),
		WithEmbeddedAllowReset(true),
	)
	require.NoError(t, err)
	defer func() {
		if closeErr := embedded.Close(); closeErr != nil {
			t.Errorf("failed to close embedded runtime: %v", closeErr)
		}
	}()

	created, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:     collectionName,
		Metadata: expectedMetadata,
		Configuration: map[string]any{
			"hnsw": map[string]any{
				"space": expectedSpace,
			},
		},
		GetOrCreate: true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	requireCollectionMetadataAndSpace(t, created, expectedMetadata, expectedSpace)

	require.NoError(t, embedded.Close())

	reopened, err := NewEmbedded(
		WithEmbeddedPersistPath(persistPath),
		WithEmbeddedAllowReset(true),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if closeErr := reopened.Close(); closeErr != nil {
			t.Errorf("failed to close reopened embedded runtime: %v", closeErr)
		}
	})

	got, err := reopened.GetCollection(EmbeddedGetCollectionRequest{
		Name: collectionName,
	})
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	requireCollectionMetadataAndSpace(t, got, expectedMetadata, expectedSpace)

	collections, err := reopened.ListCollections(EmbeddedListCollectionsRequest{})
	require.NoError(t, err)
	var listed *EmbeddedCollection
	for i := range collections {
		if collections[i].ID == created.ID {
			listed = &collections[i]
			break
		}
	}
	require.NotNil(t, listed)
	requireCollectionMetadataAndSpace(t, listed, expectedMetadata, expectedSpace)
}

func TestEmbeddedCreateCollectionPersistsSchemaAcrossRestart(t *testing.T) {
	require.NoError(t, Init(""))

	persistPath := filepath.Join(t.TempDir(), "embedded-persist-schema")
	sourceName := fmt.Sprintf("schema_source_%d", time.Now().UnixNano())
	targetName := fmt.Sprintf("schema_target_%d", time.Now().UnixNano())
	expectedSpace := "cosine"

	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath(persistPath),
		WithEmbeddedAllowReset(true),
	)
	require.NoError(t, err)
	defer func() {
		if closeErr := embedded.Close(); closeErr != nil {
			t.Errorf("failed to close embedded runtime: %v", closeErr)
		}
	}()

	source, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name: sourceName,
		Configuration: map[string]any{
			"hnsw": map[string]any{
				"space": expectedSpace,
			},
		},
		GetOrCreate: true,
	})
	require.NoError(t, err)
	require.NotNil(t, source.Schema)
	require.Equal(t, expectedSpace, extractSchemaHNSWSpace(t, source.Schema))

	target, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:        targetName,
		Schema:      source.Schema,
		GetOrCreate: true,
	})
	require.NoError(t, err)
	require.NotNil(t, target.Schema)
	require.Equal(t, expectedSpace, extractSchemaHNSWSpace(t, target.Schema))

	require.NoError(t, embedded.Close())

	reopened, err := NewEmbedded(
		WithEmbeddedPersistPath(persistPath),
		WithEmbeddedAllowReset(true),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if closeErr := reopened.Close(); closeErr != nil {
			t.Errorf("failed to close reopened embedded runtime: %v", closeErr)
		}
	})

	got, err := reopened.GetCollection(EmbeddedGetCollectionRequest{Name: targetName})
	require.NoError(t, err)
	require.NotNil(t, got.Schema)
	require.Equal(t, expectedSpace, extractSchemaHNSWSpace(t, got.Schema))

	collections, err := reopened.ListCollections(EmbeddedListCollectionsRequest{})
	require.NoError(t, err)
	var listed *EmbeddedCollection
	for i := range collections {
		if collections[i].ID == target.ID {
			listed = &collections[i]
			break
		}
	}
	require.NotNil(t, listed)
	require.NotNil(t, listed.Schema)
	require.Equal(t, expectedSpace, extractSchemaHNSWSpace(t, listed.Schema))
}

func requireCollectionMetadataAndSpace(
	t *testing.T,
	collection *EmbeddedCollection,
	expectedMetadata map[string]any,
	expectedSpace string,
) {
	t.Helper()

	require.NotNil(t, collection)
	require.NotNil(t, collection.Metadata)
	for key, expected := range expectedMetadata {
		value, ok := collection.Metadata[key]
		require.Truef(t, ok, "metadata key %q missing", key)
		if expectedNumber, ok := coerceToFloat64(expected); ok {
			actualNumber, actualIsNumber := coerceToFloat64(value)
			require.Truef(t, actualIsNumber, "metadata key %q expected numeric value, got %T", key, value)
			require.Equalf(t, expectedNumber, actualNumber, "metadata key %q mismatch", key)
			continue
		}
		require.Equalf(t, expected, value, "metadata key %q mismatch", key)
	}

	require.NotNil(t, collection.ConfigurationJSON)
	hnswRaw, ok := collection.ConfigurationJSON["hnsw"]
	require.True(t, ok)
	hnsw, ok := hnswRaw.(map[string]any)
	require.Truef(t, ok, "expected hnsw config to decode as map, got %T", hnswRaw)

	spaceRaw, ok := hnsw["space"]
	require.True(t, ok)
	space, ok := spaceRaw.(string)
	require.Truef(t, ok, "expected hnsw.space to decode as string, got %T", spaceRaw)
	require.Equal(t, expectedSpace, space)
}

func extractSchemaHNSWSpace(t *testing.T, schema map[string]any) string {
	t.Helper()

	defaultsRaw, ok := schema["defaults"]
	require.True(t, ok)
	defaults, ok := defaultsRaw.(map[string]any)
	require.Truef(t, ok, "expected schema.defaults as map, got %T", defaultsRaw)

	floatListRaw, ok := defaults["float_list"]
	require.True(t, ok)
	floatList, ok := floatListRaw.(map[string]any)
	require.Truef(t, ok, "expected schema.defaults.float_list as map, got %T", floatListRaw)

	vectorIndexRaw, ok := floatList["vector_index"]
	require.True(t, ok)
	vectorIndex, ok := vectorIndexRaw.(map[string]any)
	require.Truef(t, ok, "expected schema.defaults.float_list.vector_index as map, got %T", vectorIndexRaw)

	configRaw, ok := vectorIndex["config"]
	require.True(t, ok)
	config, ok := configRaw.(map[string]any)
	require.Truef(t, ok, "expected schema.defaults.float_list.vector_index.config as map, got %T", configRaw)

	spaceRaw, ok := config["space"]
	require.True(t, ok)
	space, ok := spaceRaw.(string)
	require.Truef(t, ok, "expected schema hnsw space as string, got %T", spaceRaw)
	return space
}

func coerceToFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}
