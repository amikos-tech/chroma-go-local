package chroma

// Compaction types only — no builder functions because compaction uses direct
// request structs, not the variadic option pattern used by backup/rebuild/WAL prune.

import "github.com/amikos-tech/chroma-go-local/internal/runtime"

type CompactCollectionRequest = runtime.CompactCollectionRequest
type CompactAllRequest = runtime.CompactAllRequest
type CompactionCollectionResult = runtime.CompactionCollectionResult
type CompactionResult = runtime.CompactionResult
