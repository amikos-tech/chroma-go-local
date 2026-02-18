package chroma

import (
	"os"

	"github.com/ebitengine/purego"
	"github.com/pkg/errors"
)

const envLibPath = "CHROMA_LIB_PATH"

func loadLibrary(path string) (uintptr, error) {
	if path == "" {
		path = os.Getenv(envLibPath)
	}
	if path == "" {
		return 0, errors.New("library path not specified and CHROMA_LIB_PATH not set")
	}

	libHandle, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil || libHandle == 0 {
		return 0, errors.Wrapf(err, "failed to load library: %s", path)
	}
	return libHandle, nil
}
