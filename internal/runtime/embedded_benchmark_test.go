package runtime

import (
	"testing"
)

func newEmbeddedForBenchmark(b *testing.B) *Embedded {
	b.Helper()

	if err := Init(""); err != nil {
		b.Skipf("Init failed (set CHROMA_LIB_PATH for benchmarks): %v", err)
	}

	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath(b.TempDir()),
		WithEmbeddedAllowReset(true),
	)
	if err != nil {
		b.Fatalf("Failed to start embedded mode for benchmark: %v", err)
	}
	b.Cleanup(func() { _ = embedded.Close() })
	return embedded
}

func BenchmarkEmbeddedHeartbeat(b *testing.B) {
	embedded := newEmbeddedForBenchmark(b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		heartbeat, err := embedded.Heartbeat()
		if err != nil {
			b.Fatalf("Heartbeat failed: %v", err)
		}
		if heartbeat == 0 {
			b.Fatal("heartbeat should not be zero")
		}
	}
}

func BenchmarkEmbeddedMaxBatchSize(b *testing.B) {
	embedded := newEmbeddedForBenchmark(b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		maxBatchSize, err := embedded.MaxBatchSize()
		if err != nil {
			b.Fatalf("MaxBatchSize failed: %v", err)
		}
		if maxBatchSize == 0 {
			b.Fatal("max batch size should be greater than zero")
		}
	}
}

func BenchmarkMarshalRequestJSON(b *testing.B) {
	request := EmbeddedAddRequest{
		CollectionID: "collection-bench",
		IDs:          []string{"id-1", "id-2"},
		Embeddings: [][]float32{
			{0.1, 0.2, 0.3},
			{0.4, 0.5, 0.6},
		},
		Documents: []string{"doc-1", "doc-2"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload, err := marshalRequestJSON(request)
		if err != nil {
			b.Fatalf("marshalRequestJSON failed: %v", err)
		}
		if len(payload) == 0 || payload[len(payload)-1] != 0 {
			b.Fatal("expected null-terminated payload")
		}
	}
}

func BenchmarkAddValidationFailure(b *testing.B) {
	fakeEmbedded := &Embedded{handle: 1}
	request := EmbeddedAddRequest{
		CollectionID: "collection-bench",
		IDs:          []string{"id-1", "id-2"},
		Embeddings:   [][]float32{{0.1, 0.2, 0.3}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := fakeEmbedded.Add(request); err == nil {
			b.Fatal("expected validation error")
		}
	}
}
