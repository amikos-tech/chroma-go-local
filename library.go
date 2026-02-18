package chroma

import (
	"os"

	"github.com/pkg/errors"
)

const envLibPath = "CHROMA_LIB_PATH"

func resolveLibraryPath(path string) (string, error) {
	if path == "" {
		path = os.Getenv(envLibPath)
	}
	if path == "" {
		return "", errors.New("library path not specified and CHROMA_LIB_PATH not set")
	}
	return path, nil
}
