//go:build windows

package chroma

import (
	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

func loadLibrary(path string) (uintptr, error) {
	resolvedPath, err := resolveLibraryPath(path)
	if err != nil {
		return 0, err
	}

	handle, err := windows.LoadLibrary(resolvedPath)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to load library: %s", resolvedPath)
	}
	if handle == 0 {
		return 0, errors.Errorf("failed to load library: %s", resolvedPath)
	}

	return uintptr(handle), nil
}
