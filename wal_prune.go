package chroma

import (
	"time"

	"github.com/amikos-tech/chroma-go-local/internal/runtime"
)

type WALPruneCollectionResult = runtime.WALPruneCollectionResult
type WALPruneResult = runtime.WALPruneResult
type WALPruneOption = runtime.WALPruneOption

func WithWALPruneTenantID(tenantID string) WALPruneOption {
	return runtime.WithWALPruneTenantID(tenantID)
}

func WithWALPruneDatabaseName(databaseName string) WALPruneOption {
	return runtime.WithWALPruneDatabaseName(databaseName)
}

func WithWALPruneDryRun() WALPruneOption {
	return runtime.WithWALPruneDryRun()
}

func WithWALPruneVacuum() WALPruneOption {
	return runtime.WithWALPruneVacuum()
}

func WithWALPruneMaxAge(maxAge time.Duration) WALPruneOption {
	return runtime.WithWALPruneMaxAge(maxAge)
}

func WithWALPruneMaxBytes(maxBytes uint64) WALPruneOption {
	return runtime.WithWALPruneMaxBytes(maxBytes)
}

func WithWALPruneWatermark(highBytes, lowBytes uint64) WALPruneOption {
	return runtime.WithWALPruneWatermark(highBytes, lowBytes)
}
