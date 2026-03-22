package runtime

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInitAndVersion(t *testing.T) {
	if err := Init(""); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	version := Version()
	if version == "" {
		t.Fatal("Version returned empty string")
	}

	versionWithError, err := VersionWithError()
	require.NoError(t, err)
	require.Equal(t, version, versionWithError)
	require.Equal(t, readShimCargoVersion(t), version)

	t.Logf("Chroma shim version: %s", version)
}

func TestVersionWithErrorWhenLibraryNotLoaded(t *testing.T) {
	originalHandle := libHandle
	libHandle = 0
	t.Cleanup(func() {
		libHandle = originalHandle
	})

	version, err := VersionWithError()
	require.ErrorIs(t, err, ErrLibraryNotLoaded)
	require.Empty(t, version)
}

func TestStartServerFromString(t *testing.T) {
	if err := Init(""); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	config := `
port: 8765
listen_address: "127.0.0.1"
persist_path: "./chroma_test_data"
allow_reset: true
`

	server, err := StartServer(StartServerConfig{
		ConfigString: config,
	})
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	if server.Port() != 8765 {
		t.Errorf("Expected port 8765, got %d", server.Port())
	}

	if server.Address() != "127.0.0.1" {
		t.Errorf("Expected address 127.0.0.1, got %s", server.Address())
	}

	require.Eventually(t, func() bool {
		resp, err := http.Get("http://127.0.0.1:8765/api/v2/heartbeat")
		if err != nil {
			return false
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "server heartbeat did not become ready")

	t.Log("Server is running and responding to heartbeat")
}

func readShimCargoVersion(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile("../../shim/Cargo.toml")
	require.NoError(t, err)

	section := ""
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line
			continue
		}

		if section != "[package]" || !strings.HasPrefix(line, "version") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		require.Len(t, parts, 2)

		version := strings.TrimSpace(parts[1])
		version = strings.Trim(version, "\"")
		require.NotEmpty(t, version)
		return version
	}

	t.Fatal("failed to find shim package version in ../../shim/Cargo.toml")
	return ""
}

func TestStartServerFromStringReportsBindConflicts(t *testing.T) {
	if err := Init(""); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	config := `
port: 8766
listen_address: "127.0.0.1"
persist_path: "./chroma_test_data_bind_conflict"
allow_reset: true
`

	firstServer, err := StartServer(StartServerConfig{
		ConfigString: config,
	})
	require.NoError(t, err)
	defer func() { _ = firstServer.Close() }()

	require.Eventually(t, func() bool {
		resp, err := http.Get("http://127.0.0.1:8766/api/v2/heartbeat")
		if err != nil {
			return false
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "first server did not become ready")

	_, err = StartServer(StartServerConfig{
		ConfigString: config,
	})
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "startup")
}

func TestServerConfigToYAML(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.Port = 9000
	cfg.PersistPath = "/data/chroma"
	cfg.AllowReset = true
	cfg.CORSAllowOrigins = []string{"http://localhost:3000", "http://example.com"}

	yaml := cfg.toYAML()

	if !strings.Contains(yaml, "port: 9000") {
		t.Errorf("YAML missing port, got: %s", yaml)
	}
	if !strings.Contains(yaml, `persist_path: "/data/chroma"`) {
		t.Errorf("YAML missing persist_path, got: %s", yaml)
	}
	if !strings.Contains(yaml, "allow_reset: true") {
		t.Errorf("YAML missing allow_reset, got: %s", yaml)
	}
	if !strings.Contains(yaml, "cors_allow_origins:") {
		t.Errorf("YAML missing cors_allow_origins, got: %s", yaml)
	}
}

func TestServerConfigWithOptions(t *testing.T) {
	cfg := DefaultServerConfig()

	WithPort(9090)(cfg)
	WithListenAddress("192.168.1.1")(cfg)
	WithPersistPath("/custom/path")(cfg)
	WithAllowReset(true)(cfg)
	WithOpenTelemetry("http://otel:4317", "my-service")(cfg)

	if cfg.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Port)
	}
	if cfg.ListenAddress != "192.168.1.1" {
		t.Errorf("Expected address 192.168.1.1, got %s", cfg.ListenAddress)
	}
	if cfg.PersistPath != "/custom/path" {
		t.Errorf("Expected persist path /custom/path, got %s", cfg.PersistPath)
	}
	if !cfg.AllowReset {
		t.Error("Expected allow_reset to be true")
	}
	if cfg.OTelEndpoint != "http://otel:4317" {
		t.Errorf("Expected OTel endpoint http://otel:4317, got %s", cfg.OTelEndpoint)
	}
}

func TestNewServerWithOptions(t *testing.T) {
	if err := Init(""); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	server, err := NewServer(
		WithPort(8766),
		WithListenAddress("127.0.0.1"),
		WithPersistPath("./chroma_test_data_builder"),
		WithAllowReset(true),
	)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	if server.Port() != 8766 {
		t.Errorf("Expected port 8766, got %d", server.Port())
	}

	require.Eventually(t, func() bool {
		resp, err := http.Get("http://127.0.0.1:8766/api/v2/heartbeat")
		if err != nil {
			return false
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "server heartbeat did not become ready")

	t.Log("NewServer with options is working")
}
