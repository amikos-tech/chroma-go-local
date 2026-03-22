package chroma

import "github.com/amikos-tech/chroma-go-local/internal/runtime"

const (
	Success           = runtime.Success
	ErrNullInput      = runtime.ErrNullInput
	ErrInvalidUTF8    = runtime.ErrInvalidUTF8
	ErrConfigParse    = runtime.ErrConfigParse
	ErrServerStart    = runtime.ErrServerStart
	ErrInvalidHandle  = runtime.ErrInvalidHandle
	ErrRuntimeCreate  = runtime.ErrRuntimeCreate
	ErrAlreadyStopped = runtime.ErrAlreadyStopped
	ErrOperation      = runtime.ErrOperation
)

var (
	ErrNullPointer        = runtime.ErrNullPointer
	ErrLibraryNotLoaded   = runtime.ErrLibraryNotLoaded
	ErrServerNotStarted   = runtime.ErrServerNotStarted
	ErrServerAlreadyStop  = runtime.ErrServerAlreadyStop
	ErrEmbeddedNotStarted = runtime.ErrEmbeddedNotStarted
)
