package chroma

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServerBackupRestartsAndWritesManifest(t *testing.T) {
	server, _ := startTestServer(t)

	backupDir := filepath.Join(t.TempDir(), "server-backup")
	manifest, err := server.Backup(ServerBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: backupDir,
			IncludeMetadata: true,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, manifest)
	require.Equal(t, BackupModeServer, manifest.Mode)
	require.NotEmpty(t, manifest.WrapperVersion)
	require.NotEmpty(t, manifest.SourcePaths)
	require.GreaterOrEqual(t, manifest.FileCount, 1)
	require.NotEmpty(t, manifest.Files)

	manifestPayload, err := os.ReadFile(filepath.Join(backupDir, backupManifestFilename))
	require.NoError(t, err)
	var decoded BackupManifest
	require.NoError(t, json.Unmarshal(manifestPayload, &decoded))
	require.Equal(t, backupSchemaVersion, decoded.SchemaVersion)
	require.Equal(t, manifest.Mode, decoded.Mode)
	require.Equal(t, manifest.FileCount, decoded.FileCount)

	snapshotSentinel := filepath.Join(backupDir, backupSnapshotDirname, "sentinel.txt")
	data, err := os.ReadFile(snapshotSentinel)
	require.NoError(t, err)
	require.Equal(t, "server-backup", string(data))
	sentinelMetadata := requireFileMetadataByPath(t, manifest.Files, "sentinel.txt")
	require.Equal(t, sha256Hex([]byte("server-backup")), sentinelMetadata.SHA256)

	requireServerHeartbeat(t, server.URL())

	restoredPort := reserveFreeLoopbackPort(t)
	restoredServer, err := NewServer(
		WithPort(restoredPort),
		WithListenAddress("127.0.0.1"),
		WithPersistPath(filepath.Join(backupDir, backupSnapshotDirname)),
		WithAllowReset(true),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = restoredServer.Close() })
	requireServerHeartbeat(t, restoredServer.URL())
}

func TestServerBackupRejectsEmptyDestinationWithoutStoppingServer(t *testing.T) {
	server, _ := startTestServer(t)

	_, err := server.Backup()
	require.Error(t, err)
	require.Contains(t, err.Error(), "destination_path is required")

	requireServerHeartbeat(t, server.URL())
}

func TestServerBackupRejectsNonEmptyDestinationWithoutStoppingServer(t *testing.T) {
	server, _ := startTestServer(t)
	originalHandle := atomic.LoadUintptr(&server.handle)

	backupDir := filepath.Join(t.TempDir(), "server-backup")
	require.NoError(t, os.MkdirAll(backupDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(backupDir, "preexisting.txt"), []byte("occupied"), 0o644))

	_, err := server.Backup(WithDestination(backupDir))
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be empty")
	require.Equal(t, originalHandle, atomic.LoadUintptr(&server.handle))

	// Validation should fail before closing the server.
	requireServerHeartbeat(t, server.URL())
}

func TestServerBackupRejectsDestinationInsideSourceWithoutStoppingServer(t *testing.T) {
	server, persistDir := startTestServer(t)

	_, err := server.Backup(WithDestination(filepath.Join(persistDir, "nested-backup")))
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be inside source persist path")

	// Rejected before shutdown; server should remain available.
	requireServerHeartbeat(t, server.URL())
}

func TestServerBackupRejectsEmbeddedOnlyOptionWithoutStoppingServer(t *testing.T) {
	server, _ := startTestServer(t)

	_, err := server.Backup(
		WithDestination(filepath.Join(t.TempDir(), "server-backup")),
		WithLeaveClosed(),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "WithLeaveClosed is only valid for embedded backups")

	requireServerHeartbeat(t, server.URL())
}

func TestServerBackupRejectsDuplicateDestinationOptionWithoutStoppingServer(t *testing.T) {
	server, _ := startTestServer(t)

	_, err := server.Backup(
		WithDestination(filepath.Join(t.TempDir(), "server-backup-a")),
		WithDestination(filepath.Join(t.TempDir(), "server-backup-b")),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "destination path already set")

	requireServerHeartbeat(t, server.URL())
}

func TestServerBackupLeaveStoppedSkipsRestart(t *testing.T) {
	server, _ := startTestServer(t)

	backupDir := filepath.Join(t.TempDir(), "server-backup")
	manifest, err := server.Backup(ServerBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: backupDir,
			IncludeMetadata: false,
		},
		LeaveStopped: true,
	})
	require.NoError(t, err)
	require.NotNil(t, manifest)
	require.False(t, manifest.IncludeMetadata)
	require.Empty(t, manifest.Files)
	requireServerUnavailable(t, server.URL())
	require.ErrorIs(t, server.Stop(), ErrServerNotStarted)
}

func TestServerBackupReportsRestartFailureAfterSuccessfulSnapshot(t *testing.T) {
	require.NoError(t, Init(""))

	persistDir := filepath.Join(t.TempDir(), "server-persist")
	require.NoError(t, os.MkdirAll(persistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(persistDir, "sentinel.txt"), []byte("server-backup"), 0o644))

	port := reserveFreeLoopbackPort(t)
	server, err := NewServer(
		WithPort(port),
		WithListenAddress("127.0.0.1"),
		WithPersistPath(persistDir),
		WithAllowReset(true),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Close() })

	requireServerHeartbeat(t, server.URL())

	server.stateMu.Lock()
	server.config = StartServerConfig{}
	server.stateMu.Unlock()

	backupDir := filepath.Join(t.TempDir(), "server-backup")
	manifest, err := server.Backup(ServerBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: backupDir,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "backup completed but server restart failed")
	require.NotNil(t, manifest)

	snapshotSentinel := filepath.Join(backupDir, backupSnapshotDirname, "sentinel.txt")
	data, readErr := os.ReadFile(snapshotSentinel)
	require.NoError(t, readErr)
	require.Equal(t, "server-backup", string(data))

	requireServerUnavailable(t, server.URL())
	require.ErrorIs(t, server.Stop(), ErrServerNotStarted)
}

func TestServerBackupConcurrentCallFailsWhileFirstBackupLeavesServerStopped(t *testing.T) {
	server, _ := startTestServer(t)

	firstResult := make(chan error, 1)
	backupDirOne := filepath.Join(t.TempDir(), "server-backup-one")
	go func() {
		_, firstErr := server.Backup(ServerBackupOptions{
			BackupOptions: BackupOptions{
				DestinationPath: backupDirOne,
			},
			LeaveStopped: true,
		})
		firstResult <- firstErr
	}()

	requireServerUnavailable(t, server.URL())

	backupDirTwo := filepath.Join(t.TempDir(), "server-backup-two")
	_, err := server.Backup(ServerBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: backupDirTwo,
		},
	})
	require.ErrorIs(t, err, ErrServerNotStarted)
	require.NoError(t, <-firstResult)
}

func TestServerBackupConcurrentCallsWithRestartBothSucceed(t *testing.T) {
	server, _ := startTestServer(t)

	type backupResult struct {
		manifest *BackupManifest
		err      error
	}

	start := make(chan struct{})
	results := make(chan backupResult, 2)
	runBackup := func(destination string) {
		<-start
		manifest, backupErr := server.Backup(ServerBackupOptions{
			BackupOptions: BackupOptions{
				DestinationPath: destination,
			},
		})
		results <- backupResult{manifest: manifest, err: backupErr}
	}

	go runBackup(filepath.Join(t.TempDir(), "server-backup-one"))
	go runBackup(filepath.Join(t.TempDir(), "server-backup-two"))
	close(start)

	first := <-results
	second := <-results
	for _, result := range []backupResult{first, second} {
		require.NoError(t, result.err)
		require.NotNil(t, result.manifest)
	}

	requireServerHeartbeat(t, server.URL())
}

func TestServerBackupGuardsNilAndClosedReceiver(t *testing.T) {
	require.NoError(t, Init(""))

	var nilServer *Server
	_, err := nilServer.Backup(ServerBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: t.TempDir(),
		},
	})
	require.ErrorIs(t, err, ErrServerNotStarted)

	port := reserveFreeLoopbackPort(t)
	server, err := NewServer(
		WithPort(port),
		WithListenAddress("127.0.0.1"),
		WithPersistPath(filepath.Join(t.TempDir(), "server-persist")),
		WithAllowReset(true),
	)
	require.NoError(t, err)
	require.NoError(t, server.Close())

	_, err = server.Backup(ServerBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: t.TempDir(),
		},
	})
	require.ErrorIs(t, err, ErrServerNotStarted)
}

func TestServerBackupUsesRuntimePersistPathFromEnvOverride(t *testing.T) {
	require.NoError(t, Init(""))

	yamlPersistDir := filepath.Join(t.TempDir(), "yaml-persist")
	overridePersistDir := filepath.Join(t.TempDir(), "override-persist")
	require.NoError(t, os.MkdirAll(yamlPersistDir, 0o755))
	require.NoError(t, os.MkdirAll(overridePersistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(yamlPersistDir, "yaml.txt"), []byte("yaml"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(overridePersistDir, "override.txt"), []byte("override"), 0o644))

	t.Setenv("CHROMA_PERSIST_PATH", overridePersistDir)

	port := reserveFreeLoopbackPort(t)
	server, err := StartServer(StartServerConfig{
		ConfigString: fmt.Sprintf(
			"port: %d\nlisten_address: \"127.0.0.1\"\npersist_path: %q\nallow_reset: true\n",
			port,
			yamlPersistDir,
		),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Close() })

	requireServerHeartbeat(t, server.URL())

	backupDir := filepath.Join(t.TempDir(), "server-backup-env")
	manifest, err := server.Backup(ServerBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: backupDir,
		},
		LeaveStopped: true,
	})
	require.NoError(t, err)
	require.NotNil(t, manifest)

	data, err := os.ReadFile(filepath.Join(backupDir, backupSnapshotDirname, "override.txt"))
	require.NoError(t, err)
	require.Equal(t, "override", string(data))

	_, err = os.Stat(filepath.Join(backupDir, backupSnapshotDirname, "yaml.txt"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestEmbeddedBackupReopensAndPreservesData(t *testing.T) {
	embedded, _ := startTestEmbedded(t)

	collectionName := fmt.Sprintf("backup_%d", time.Now().UnixNano())
	collection, err := embedded.CreateCollection(EmbeddedCreateCollectionRequest{
		Name:        collectionName,
		GetOrCreate: true,
	})
	require.NoError(t, err)

	err = embedded.Add(EmbeddedAddRequest{
		CollectionID: collection.ID,
		IDs:          []string{"doc-1"},
		Embeddings: [][]float32{
			{0.1, 0.2, 0.3},
		},
		Documents: []string{"hello"},
	})
	require.NoError(t, err)

	beforeCount, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, beforeCount)

	backupDir := filepath.Join(t.TempDir(), "embedded-backup")
	manifest, err := embedded.Backup(EmbeddedBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: backupDir,
			IncludeMetadata: true,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, manifest)
	require.Equal(t, BackupModeEmbedded, manifest.Mode)
	require.NotEmpty(t, manifest.WrapperVersion)
	require.GreaterOrEqual(t, manifest.FileCount, 1)

	heartbeat, err := embedded.Heartbeat()
	require.NoError(t, err)
	require.NotZero(t, heartbeat)

	afterCount, err := embedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
	})
	require.NoError(t, err)
	require.Equal(t, beforeCount, afterCount)

	snapshotSentinel := filepath.Join(backupDir, backupSnapshotDirname, "sentinel.txt")
	data, err := os.ReadFile(snapshotSentinel)
	require.NoError(t, err)
	require.Equal(t, "embedded-backup", string(data))
	sentinelMetadata := requireFileMetadataByPath(t, manifest.Files, "sentinel.txt")
	require.Equal(t, sha256Hex([]byte("embedded-backup")), sentinelMetadata.SHA256)

	restoredEmbedded, err := NewEmbedded(
		WithEmbeddedPersistPath(filepath.Join(backupDir, backupSnapshotDirname)),
		WithEmbeddedAllowReset(true),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = restoredEmbedded.Close() })

	restoredCount, err := restoredEmbedded.CountRecords(EmbeddedCountRecordsRequest{
		CollectionID: collection.ID,
	})
	require.NoError(t, err)
	require.Equal(t, beforeCount, restoredCount)
}

func TestEmbeddedBackupLeaveClosedSkipsReopen(t *testing.T) {
	embedded, _ := startTestEmbedded(t)

	backupDir := filepath.Join(t.TempDir(), "embedded-backup")
	manifest, err := embedded.Backup(EmbeddedBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: backupDir,
			IncludeMetadata: false,
		},
		LeaveClosed: true,
	})
	require.NoError(t, err)
	require.NotNil(t, manifest)
	require.False(t, manifest.IncludeMetadata)
	require.Empty(t, manifest.Files)
	_, err = embedded.Heartbeat()
	require.ErrorIs(t, err, ErrEmbeddedNotStarted)
}

func TestEmbeddedBackupRejectsServerOnlyOptionWithoutClosingEmbedded(t *testing.T) {
	embedded, _ := startTestEmbedded(t)

	_, err := embedded.Backup(
		WithDestination(filepath.Join(t.TempDir(), "embedded-backup")),
		WithLeaveStopped(),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "WithLeaveStopped is only valid for server backups")

	heartbeat, err := embedded.Heartbeat()
	require.NoError(t, err)
	require.NotZero(t, heartbeat)
}

func TestEmbeddedBackupReportsReopenFailureAfterSuccessfulSnapshot(t *testing.T) {
	require.NoError(t, Init(""))

	persistDir := filepath.Join(t.TempDir(), "embedded-persist")
	require.NoError(t, os.MkdirAll(persistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(persistDir, "sentinel.txt"), []byte("embedded-backup"), 0o644))

	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath(persistDir),
		WithEmbeddedAllowReset(true),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = embedded.Close() })

	embedded.stateMu.Lock()
	embedded.config = StartEmbeddedConfig{}
	embedded.stateMu.Unlock()

	backupDir := filepath.Join(t.TempDir(), "embedded-backup")
	manifest, err := embedded.Backup(EmbeddedBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: backupDir,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "backup completed but embedded reopen failed")
	require.NotNil(t, manifest)

	snapshotSentinel := filepath.Join(backupDir, backupSnapshotDirname, "sentinel.txt")
	data, readErr := os.ReadFile(snapshotSentinel)
	require.NoError(t, readErr)
	require.Equal(t, "embedded-backup", string(data))

	requireEmbeddedUnavailable(t, embedded)
}

func TestEmbeddedBackupConcurrentCallFailsWhileFirstBackupLeavesEmbeddedClosed(t *testing.T) {
	embedded, _ := startTestEmbedded(t)

	firstResult := make(chan error, 1)
	backupDirOne := filepath.Join(t.TempDir(), "embedded-backup-one")
	go func() {
		_, firstErr := embedded.Backup(EmbeddedBackupOptions{
			BackupOptions: BackupOptions{
				DestinationPath: backupDirOne,
			},
			LeaveClosed: true,
		})
		firstResult <- firstErr
	}()

	requireEmbeddedUnavailable(t, embedded)

	backupDirTwo := filepath.Join(t.TempDir(), "embedded-backup-two")
	_, err := embedded.Backup(EmbeddedBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: backupDirTwo,
		},
	})
	require.ErrorIs(t, err, ErrEmbeddedNotStarted)
	require.NoError(t, <-firstResult)
}

func TestEmbeddedBackupConcurrentCallsWithReopenBothSucceed(t *testing.T) {
	embedded, _ := startTestEmbedded(t)

	type backupResult struct {
		manifest *BackupManifest
		err      error
	}

	start := make(chan struct{})
	results := make(chan backupResult, 2)
	runBackup := func(destination string) {
		<-start
		manifest, backupErr := embedded.Backup(EmbeddedBackupOptions{
			BackupOptions: BackupOptions{
				DestinationPath: destination,
			},
		})
		results <- backupResult{manifest: manifest, err: backupErr}
	}

	go runBackup(filepath.Join(t.TempDir(), "embedded-backup-one"))
	go runBackup(filepath.Join(t.TempDir(), "embedded-backup-two"))
	close(start)

	first := <-results
	second := <-results
	for _, result := range []backupResult{first, second} {
		require.NoError(t, result.err)
		require.NotNil(t, result.manifest)
	}

	heartbeat, err := embedded.Heartbeat()
	require.NoError(t, err)
	require.NotZero(t, heartbeat)
}

func TestEmbeddedBackupGuardsNilAndClosedReceiver(t *testing.T) {
	require.NoError(t, Init(""))

	var nilEmbedded *Embedded
	_, err := nilEmbedded.Backup(EmbeddedBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: t.TempDir(),
		},
	})
	require.ErrorIs(t, err, ErrEmbeddedNotStarted)

	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath(filepath.Join(t.TempDir(), "embedded-persist")),
		WithEmbeddedAllowReset(true),
	)
	require.NoError(t, err)
	require.NoError(t, embedded.Close())

	_, err = embedded.Backup(EmbeddedBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: t.TempDir(),
		},
	})
	require.ErrorIs(t, err, ErrEmbeddedNotStarted)
}

func TestEmbeddedBackupUsesRuntimePersistPathFromEnvOverride(t *testing.T) {
	require.NoError(t, Init(""))

	yamlPersistDir := filepath.Join(t.TempDir(), "yaml-persist")
	overridePersistDir := filepath.Join(t.TempDir(), "override-persist")
	require.NoError(t, os.MkdirAll(yamlPersistDir, 0o755))
	require.NoError(t, os.MkdirAll(overridePersistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(yamlPersistDir, "yaml.txt"), []byte("yaml"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(overridePersistDir, "override.txt"), []byte("override"), 0o644))

	t.Setenv("CHROMA_PERSIST_PATH", overridePersistDir)

	embedded, err := StartEmbedded(StartEmbeddedConfig{
		ConfigString: fmt.Sprintf("persist_path: %q\nallow_reset: true\n", yamlPersistDir),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = embedded.Close() })

	backupDir := filepath.Join(t.TempDir(), "embedded-backup-env")
	manifest, err := embedded.Backup(EmbeddedBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: backupDir,
		},
		LeaveClosed: true,
	})
	require.NoError(t, err)
	require.NotNil(t, manifest)

	data, err := os.ReadFile(filepath.Join(backupDir, backupSnapshotDirname, "override.txt"))
	require.NoError(t, err)
	require.Equal(t, "override", string(data))

	_, err = os.Stat(filepath.Join(backupDir, backupSnapshotDirname, "yaml.txt"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestServerBackupRecoversFromSourceReadFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on windows")
	}

	require.NoError(t, Init(""))

	persistDir := filepath.Join(t.TempDir(), "server-persist")
	require.NoError(t, os.MkdirAll(persistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(persistDir, "a-readable.txt"), []byte("ok"), 0o644))

	unreadablePath := filepath.Join(persistDir, "z-unreadable.txt")
	require.NoError(t, os.WriteFile(unreadablePath, []byte("nope"), 0o644))
	require.NoError(t, os.Chmod(unreadablePath, 0o000))
	t.Cleanup(func() { _ = os.Chmod(unreadablePath, 0o644) })

	probe, probeErr := os.Open(unreadablePath)
	if probeErr == nil {
		_ = probe.Close()
		t.Skip("filesystem does not enforce unreadable file mode for test process")
	}

	port := reserveFreeLoopbackPort(t)
	server, err := NewServer(
		WithPort(port),
		WithListenAddress("127.0.0.1"),
		WithPersistPath(persistDir),
		WithAllowReset(true),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Close() })

	requireServerHeartbeat(t, server.URL())

	backupDir := filepath.Join(t.TempDir(), "server-backup")
	manifest, err := server.Backup(ServerBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: backupDir,
		},
	})
	require.Error(t, err)
	require.Nil(t, manifest)
	require.Contains(t, err.Error(), "failed to copy persistence directory")

	// Backup should recover availability via restart when copy fails.
	requireServerHeartbeat(t, server.URL())
}

func TestNewBackupPlanRejectsSymlinkDestinationInsideSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows")
	}

	sourcePersist := filepath.Join(t.TempDir(), "source-persist")
	require.NoError(t, os.MkdirAll(sourcePersist, 0o755))

	linkRoot := t.TempDir()
	destinationLink := filepath.Join(linkRoot, "dest-link")
	if err := os.Symlink(sourcePersist, destinationLink); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}

	_, err := newBackupPlan(sourcePersist, "", BackupOptions{
		DestinationPath: destinationLink,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be inside source persist path")
}

func TestExecuteBackupCreatesEmptySnapshotWhenSourceMissing(t *testing.T) {
	destinationPath := filepath.Join(t.TempDir(), "backup-destination")
	plan := &backupPlan{
		sourcePersistPath: filepath.Join(t.TempDir(), "missing-persist"),
		sourcePathExists:  false,
		sourcePaths:       []string{"missing"},
		destinationPath:   destinationPath,
		includeMetadata:   false,
		wrapperVersion:    "test",
	}

	require.NoError(t, ensureEmptyDir(destinationPath))

	manifest, err := executeBackup(BackupModeServer, plan)
	require.NoError(t, err)
	require.NotNil(t, manifest)
	require.Equal(t, 0, manifest.FileCount)
	require.Zero(t, manifest.TotalBytes)

	info, err := os.Stat(filepath.Join(destinationPath, backupSnapshotDirname))
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestExecuteBackupLeavesPartialSnapshotOnCopyFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows")
	}

	sourcePersist := filepath.Join(t.TempDir(), "source-persist")
	require.NoError(t, os.MkdirAll(sourcePersist, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sourcePersist, "a-first.txt"), []byte("ok"), 0o644))

	linkPath := filepath.Join(sourcePersist, "b-link")
	if err := os.Symlink(filepath.Join(sourcePersist, "a-first.txt"), linkPath); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}

	destinationPath := filepath.Join(t.TempDir(), "backup-destination")
	require.NoError(t, ensureEmptyDir(destinationPath))

	plan := &backupPlan{
		sourcePersistPath: sourcePersist,
		sourcePathExists:  true,
		sourcePaths:       []string{sourcePersist},
		destinationPath:   destinationPath,
		includeMetadata:   false,
		wrapperVersion:    "test",
	}

	manifest, err := executeBackup(BackupModeServer, plan)
	require.Error(t, err)
	require.Nil(t, manifest)
	require.Contains(t, err.Error(), "backup does not support symbolic links")

	data, readErr := os.ReadFile(filepath.Join(destinationPath, backupSnapshotDirname, "a-first.txt"))
	require.NoError(t, readErr)
	require.Equal(t, "ok", string(data))

	_, statErr := os.Stat(filepath.Join(destinationPath, backupManifestFilename))
	require.Error(t, statErr)
	require.True(t, os.IsNotExist(statErr))
}

func TestExecuteBackupConcurrentWritersOnSameDestinationOneFails(t *testing.T) {
	sourcePersist := filepath.Join(t.TempDir(), "source-persist")
	require.NoError(t, os.MkdirAll(sourcePersist, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sourcePersist, "file.txt"), []byte("ok"), 0o644))

	destinationPath := filepath.Join(t.TempDir(), "backup-destination")
	require.NoError(t, ensureEmptyDir(destinationPath))

	plan := &backupPlan{
		sourcePersistPath: sourcePersist,
		sourcePathExists:  true,
		sourcePaths:       []string{sourcePersist},
		destinationPath:   destinationPath,
		includeMetadata:   false,
		wrapperVersion:    "test",
	}

	type result struct {
		manifest *BackupManifest
		err      error
	}

	start := make(chan struct{})
	results := make(chan result, 2)
	runBackup := func() {
		<-start
		manifest, err := executeBackup(BackupModeServer, plan)
		results <- result{manifest: manifest, err: err}
	}

	go runBackup()
	go runBackup()
	close(start)

	first := <-results
	second := <-results

	successes := 0
	failures := 0
	for _, outcome := range []result{first, second} {
		if outcome.err == nil {
			successes++
			require.NotNil(t, outcome.manifest)
			continue
		}
		failures++
		require.Nil(t, outcome.manifest)
		require.Contains(t, outcome.err.Error(), "failed to copy persistence directory")
	}

	require.Equal(t, 1, successes)
	require.Equal(t, 1, failures)
}

func TestIsWithinPathReturnsErrorForMixedPathTypes(t *testing.T) {
	_, err := isWithinPath("/tmp/a", "relative-parent")
	require.Error(t, err)
}

func TestServerCloseWaitsForStateLock(t *testing.T) {
	require.NoError(t, Init(""))

	port := reserveFreeLoopbackPort(t)
	server, err := NewServer(
		WithPort(port),
		WithListenAddress("127.0.0.1"),
		WithPersistPath(filepath.Join(t.TempDir(), "server-persist")),
		WithAllowReset(true),
	)
	require.NoError(t, err)
	defer func() { _ = server.Close() }()

	locked := true
	server.stateMu.Lock()
	defer func() {
		if locked {
			server.stateMu.Unlock()
		}
	}()

	done := make(chan struct{})
	go func() {
		_ = server.Close()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Close should block while state lock is held")
	case <-time.After(100 * time.Millisecond):
	}

	server.stateMu.Unlock()
	locked = false

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not complete after releasing state lock")
	}
}

func TestEmbeddedCloseWaitsForStateLock(t *testing.T) {
	require.NoError(t, Init(""))

	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath(filepath.Join(t.TempDir(), "embedded-persist")),
		WithEmbeddedAllowReset(true),
	)
	require.NoError(t, err)
	defer func() { _ = embedded.Close() }()

	locked := true
	embedded.stateMu.Lock()
	defer func() {
		if locked {
			embedded.stateMu.Unlock()
		}
	}()

	done := make(chan struct{})
	go func() {
		_ = embedded.Close()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Close should block while state lock is held")
	case <-time.After(100 * time.Millisecond):
	}

	embedded.stateMu.Unlock()
	locked = false

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not complete after releasing state lock")
	}
}

func startTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	require.NoError(t, Init(""))

	persistDir := filepath.Join(t.TempDir(), "server-persist")
	require.NoError(t, os.MkdirAll(persistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(persistDir, "sentinel.txt"), []byte("server-backup"), 0o644))

	port := reserveFreeLoopbackPort(t)
	server, err := NewServer(
		WithPort(port),
		WithListenAddress("127.0.0.1"),
		WithPersistPath(persistDir),
		WithAllowReset(true),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Close() })
	requireServerHeartbeat(t, server.URL())
	return server, persistDir
}

func startTestEmbedded(t *testing.T) (*Embedded, string) {
	t.Helper()
	require.NoError(t, Init(""))

	persistDir := filepath.Join(t.TempDir(), "embedded-persist")
	require.NoError(t, os.MkdirAll(persistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(persistDir, "sentinel.txt"), []byte("embedded-backup"), 0o644))

	embedded, err := NewEmbedded(
		WithEmbeddedPersistPath(persistDir),
		WithEmbeddedAllowReset(true),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = embedded.Close() })
	return embedded, persistDir
}

func reserveFreeLoopbackPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()
	return listener.Addr().(*net.TCPAddr).Port
}

func requireServerHeartbeat(t *testing.T, url string) {
	t.Helper()
	client := &http.Client{Timeout: 250 * time.Millisecond}
	heartbeatURL := fmt.Sprintf("%s/api/v2/heartbeat", url)
	require.Eventually(t, func() bool {
		resp, err := client.Get(heartbeatURL)
		if err != nil {
			return false
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "server heartbeat did not become ready")
}

func requireServerUnavailable(t *testing.T, url string) {
	t.Helper()
	client := &http.Client{Timeout: 250 * time.Millisecond}
	heartbeatURL := fmt.Sprintf("%s/api/v2/heartbeat", url)
	require.Eventually(t, func() bool {
		resp, err := client.Get(heartbeatURL)
		if err != nil {
			return true
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return false
	}, 5*time.Second, 100*time.Millisecond, "server heartbeat remained reachable")
}

func requireEmbeddedUnavailable(t *testing.T, embedded *Embedded) {
	t.Helper()
	require.Eventually(t, func() bool {
		_, err := embedded.Heartbeat()
		return errors.Is(err, ErrEmbeddedNotStarted)
	}, 5*time.Second, 100*time.Millisecond, "embedded mode remained reachable")
}

func requireFileMetadataByPath(t *testing.T, files []BackupFileMetadata, path string) BackupFileMetadata {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("manifest metadata for path %q not found", path)
	return BackupFileMetadata{}
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
