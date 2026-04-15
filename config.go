package chroma

import "github.com/amikos-tech/chroma-go-local/internal/runtime"

type ServerConfig = runtime.ServerConfig

type ServerOption = runtime.ServerOption

func DefaultServerConfig() *ServerConfig {
	return runtime.DefaultServerConfig()
}

func NewServer(opts ...ServerOption) (*Server, error) {
	return runtime.NewServer(opts...)
}

func WithPort(port int) ServerOption {
	return runtime.WithPort(port)
}

func WithListenAddress(addr string) ServerOption {
	return runtime.WithListenAddress(addr)
}

func WithMaxPayloadSize(bytes int) ServerOption {
	return runtime.WithMaxPayloadSize(bytes)
}

func WithCORSAllowOrigins(origins ...string) ServerOption {
	return runtime.WithCORSAllowOrigins(origins...)
}

func WithPersistPath(path string) ServerOption {
	return runtime.WithPersistPath(path)
}

func WithSQLiteFilename(filename string) ServerOption {
	return runtime.WithSQLiteFilename(filename)
}

func WithAllowReset(allow bool) ServerOption {
	return runtime.WithAllowReset(allow)
}

func WithOpenTelemetry(endpoint, serviceName string) ServerOption {
	return runtime.WithOpenTelemetry(endpoint, serviceName)
}

func WithTLSCertPath(certPath string) ServerOption {
	return runtime.WithTLSCertPath(certPath)
}

func WithTLSKeyPath(keyPath string) ServerOption {
	return runtime.WithTLSKeyPath(keyPath)
}

func WithRawYAML(yaml string) ServerOption {
	return runtime.WithRawYAML(yaml)
}
