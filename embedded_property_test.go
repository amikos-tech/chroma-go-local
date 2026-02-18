package chroma

import (
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

func TestEmbeddedValidationProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

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

	properties.Property("Update rejects empty payload mutations", prop.ForAll(
		func(idsLen uint8) bool {
			if idsLen == 0 {
				return true
			}
			err := fakeEmbedded.UpdateRecords(EmbeddedUpdateRecordsRequest{
				CollectionID: "c",
				IDs:          makeIDs(int(idsLen)),
			})
			return err != nil && strings.Contains(err.Error(), "at least one of embeddings, documents, or uris")
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

	properties.TestingRun(t)
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
