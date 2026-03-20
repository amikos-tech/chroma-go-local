package chroma

import "github.com/amikos-tech/chroma-go-local/internal/runtime"

type Server = runtime.Server

type StartServerConfig = runtime.StartServerConfig

func Init(libPath string) error {
	return runtime.Init(libPath)
}

func StartServer(config StartServerConfig) (*Server, error) {
	return runtime.StartServer(config)
}

func Version() string {
	return runtime.Version()
}

func VersionWithError() (string, error) {
	return runtime.VersionWithError()
}
