package chroma

import "github.com/amikos-tech/chroma-go-local/internal/runtime"

type RebuildCollectionResult = runtime.RebuildCollectionResult
type RebuildCollectionOption = runtime.RebuildCollectionOption

func WithRebuildTenantID(tenantID string) RebuildCollectionOption {
	return runtime.WithRebuildTenantID(tenantID)
}

func WithRebuildDatabaseName(databaseName string) RebuildCollectionOption {
	return runtime.WithRebuildDatabaseName(databaseName)
}

func WithRebuildPrecheck() RebuildCollectionOption {
	return runtime.WithRebuildPrecheck()
}

func WithRebuildKeepBackup(keepBackup bool) RebuildCollectionOption {
	return runtime.WithRebuildKeepBackup(keepBackup)
}
