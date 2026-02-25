package chroma

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServerBackupRestartsAndWritesManifest(t *testing.T) {
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
	require.NoError(t, Init(""))

	persistDir := filepath.Join(t.TempDir(), "server-persist")
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

	_, err = server.Backup(ServerBackupOptions{
		BackupOptions: BackupOptions{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "destination_path is required")

	requireServerHeartbeat(t, server.URL())
}

func TestServerBackupRejectsNonEmptyDestinationWithoutStoppingServer(t *testing.T) {
	require.NoError(t, Init(""))

	persistDir := filepath.Join(t.TempDir(), "server-persist")
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
	require.NoError(t, os.MkdirAll(backupDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(backupDir, "preexisting.txt"), []byte("occupied"), 0o644))

	_, err = server.Backup(ServerBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: backupDir,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be empty")

	// Backup should recover server availability after restart.
	requireServerHeartbeat(t, server.URL())
}

func TestServerBackupRejectsDestinationInsideSourceWithoutStoppingServer(t *testing.T) {
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

	_, err = server.Backup(ServerBackupOptions{
		BackupOptions: BackupOptions{
			DestinationPath: filepath.Join(persistDir, "nested-backup"),
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be inside source persist path")

	// Rejected before shutdown; server should remain available.
	requireServerHeartbeat(t, server.URL())
}

func TestServerBackupLeaveStoppedSkipsRestart(t *testing.T) {
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
