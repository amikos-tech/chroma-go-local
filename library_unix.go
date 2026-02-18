//go:build !windows

package chroma

import (
	"github.com/ebitengine/purego"
	"github.com/pkg/errors"
)

func loadLibrary(path string) (uintptr, error) {
	resolvedPath, err := resolveLibraryPath(path)
	if err != nil {
		return 0, err
	}

	libHandle, err := purego.Dlopen(resolvedPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil || libHandle == 0 {
		return 0, errors.Wrapf(err, "failed to load library: %s", resolvedPath)
	}
	return libHandle, nil
}
