package chroma

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
