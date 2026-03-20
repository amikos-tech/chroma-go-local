package chroma

import "github.com/amikos-tech/chroma-go-local/internal/runtime"

type BackupMode = runtime.BackupMode
type BackupOptions = runtime.BackupOptions
type ServerBackupOptions = runtime.ServerBackupOptions
type EmbeddedBackupOptions = runtime.EmbeddedBackupOptions
type BackupOption = runtime.BackupOption
type BackupFileMetadata = runtime.BackupFileMetadata
type BackupManifest = runtime.BackupManifest

const (
	BackupModeServer   = runtime.BackupModeServer
	BackupModeEmbedded = runtime.BackupModeEmbedded
)

func WithDestination(path string) BackupOption {
	return runtime.WithDestination(path)
}

func WithIncludeMetadata() BackupOption {
	return runtime.WithIncludeMetadata()
}

func WithLeaveStopped() BackupOption {
	return runtime.WithLeaveStopped()
}

func WithLeaveClosed() BackupOption {
	return runtime.WithLeaveClosed()
}
