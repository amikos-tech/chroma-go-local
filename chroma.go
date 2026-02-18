package chroma

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/pkg/errors"
)

var (
	libHandle uintptr
	libOnce   sync.Once
	libErr    error

	// FFI functions
	chromaServerStart              func(*byte) uintptr
	chromaServerStartFromString    func(*byte) uintptr
	chromaServerPort               func(uintptr) int32
	chromaServerAddress            func(uintptr) *byte
	chromaServerStop               func(uintptr) int32
	chromaServerFree               func(uintptr)
	chromaEmbeddedStart            func(*byte) uintptr
	chromaEmbeddedStartFromString  func(*byte) uintptr
	chromaEmbeddedFree             func(uintptr)
	chromaEmbeddedHeartbeat        func(uintptr, *uint64) int32
	chromaEmbeddedGetMaxBatchSize  func(uintptr, *uint32) int32
	chromaEmbeddedCreateTenant     func(uintptr, *byte) int32
	chromaEmbeddedGetTenant        func(uintptr, *byte) *byte
	chromaEmbeddedUpdateTenant     func(uintptr, *byte) int32
	chromaEmbeddedReset            func(uintptr) int32
	chromaEmbeddedCreateDatabase   func(uintptr, *byte) int32
	chromaEmbeddedListDatabases    func(uintptr, *byte) *byte
	chromaEmbeddedGetDatabase      func(uintptr, *byte) *byte
	chromaEmbeddedDeleteDatabase   func(uintptr, *byte) int32
	chromaEmbeddedListCollections  func(uintptr, *byte) *byte
	chromaEmbeddedGetCollection    func(uintptr, *byte) *byte
	chromaEmbeddedCountCollections func(uintptr, *byte, *uint32) int32
	chromaEmbeddedUpdateCollection func(uintptr, *byte) int32
	chromaEmbeddedDeleteCollection func(uintptr, *byte) int32
	chromaEmbeddedForkCollection   func(uintptr, *byte) *byte
	chromaEmbeddedCount            func(uintptr, *byte, *uint32) int32
	chromaEmbeddedGet              func(uintptr, *byte) *byte
	chromaEmbeddedUpdate           func(uintptr, *byte) int32
	chromaEmbeddedUpsert           func(uintptr, *byte) int32
	chromaEmbeddedDeleteRecords    func(uintptr, *byte) int32
	chromaEmbeddedCreateCollection func(uintptr, *byte) *byte
	chromaEmbeddedAdd              func(uintptr, *byte) int32
	chromaEmbeddedQuery            func(uintptr, *byte) *byte
	chromaEmbeddedIndexingStatus   func(uintptr, *byte) *byte
	chromaEmbeddedHealthcheck      func(uintptr) *byte
	chromaStringFree               func(*byte)
	chromaGetLastError             func() *byte
	chromaVersion                  func() *byte
)

// Init initializes the Chroma library. Must be called before any other functions.
// If libPath is empty, it will look for CHROMA_LIB_PATH environment variable.
func Init(libPath string) error {
	libOnce.Do(func() {
		libHandle, libErr = loadLibrary(libPath)
		if libErr != nil {
			return
		}
		libErr = registerFunctions()
	})
	return libErr
}

func registerFunctions() error {
	purego.RegisterLibFunc(&chromaServerStart, libHandle, "chroma_server_start")
	purego.RegisterLibFunc(&chromaServerStartFromString, libHandle, "chroma_server_start_from_string")
	purego.RegisterLibFunc(&chromaServerPort, libHandle, "chroma_server_port")
	purego.RegisterLibFunc(&chromaServerAddress, libHandle, "chroma_server_address")
	purego.RegisterLibFunc(&chromaServerStop, libHandle, "chroma_server_stop")
	purego.RegisterLibFunc(&chromaServerFree, libHandle, "chroma_server_free")
	purego.RegisterLibFunc(&chromaEmbeddedStart, libHandle, "chroma_embedded_start")
	purego.RegisterLibFunc(&chromaEmbeddedStartFromString, libHandle, "chroma_embedded_start_from_string")
	purego.RegisterLibFunc(&chromaEmbeddedFree, libHandle, "chroma_embedded_free")
	purego.RegisterLibFunc(&chromaEmbeddedHeartbeat, libHandle, "chroma_embedded_heartbeat")
	purego.RegisterLibFunc(&chromaEmbeddedGetMaxBatchSize, libHandle, "chroma_embedded_get_max_batch_size")
	purego.RegisterLibFunc(&chromaEmbeddedCreateTenant, libHandle, "chroma_embedded_create_tenant")
	purego.RegisterLibFunc(&chromaEmbeddedGetTenant, libHandle, "chroma_embedded_get_tenant")
	purego.RegisterLibFunc(&chromaEmbeddedUpdateTenant, libHandle, "chroma_embedded_update_tenant")
	purego.RegisterLibFunc(&chromaEmbeddedReset, libHandle, "chroma_embedded_reset")
	purego.RegisterLibFunc(&chromaEmbeddedCreateDatabase, libHandle, "chroma_embedded_create_database")
	purego.RegisterLibFunc(&chromaEmbeddedListDatabases, libHandle, "chroma_embedded_list_databases")
	purego.RegisterLibFunc(&chromaEmbeddedGetDatabase, libHandle, "chroma_embedded_get_database")
	purego.RegisterLibFunc(&chromaEmbeddedDeleteDatabase, libHandle, "chroma_embedded_delete_database")
	purego.RegisterLibFunc(&chromaEmbeddedListCollections, libHandle, "chroma_embedded_list_collections")
	purego.RegisterLibFunc(&chromaEmbeddedGetCollection, libHandle, "chroma_embedded_get_collection")
	purego.RegisterLibFunc(&chromaEmbeddedCountCollections, libHandle, "chroma_embedded_count_collections")
	purego.RegisterLibFunc(&chromaEmbeddedUpdateCollection, libHandle, "chroma_embedded_update_collection")
	purego.RegisterLibFunc(&chromaEmbeddedDeleteCollection, libHandle, "chroma_embedded_delete_collection")
	purego.RegisterLibFunc(&chromaEmbeddedForkCollection, libHandle, "chroma_embedded_fork_collection")
	purego.RegisterLibFunc(&chromaEmbeddedCount, libHandle, "chroma_embedded_count")
	purego.RegisterLibFunc(&chromaEmbeddedGet, libHandle, "chroma_embedded_get")
	purego.RegisterLibFunc(&chromaEmbeddedUpdate, libHandle, "chroma_embedded_update")
	purego.RegisterLibFunc(&chromaEmbeddedUpsert, libHandle, "chroma_embedded_upsert")
	purego.RegisterLibFunc(&chromaEmbeddedDeleteRecords, libHandle, "chroma_embedded_delete_records")
	purego.RegisterLibFunc(&chromaEmbeddedCreateCollection, libHandle, "chroma_embedded_create_collection")
	purego.RegisterLibFunc(&chromaEmbeddedAdd, libHandle, "chroma_embedded_add")
	purego.RegisterLibFunc(&chromaEmbeddedQuery, libHandle, "chroma_embedded_query")
	purego.RegisterLibFunc(&chromaEmbeddedIndexingStatus, libHandle, "chroma_embedded_indexing_status")
	purego.RegisterLibFunc(&chromaEmbeddedHealthcheck, libHandle, "chroma_embedded_healthcheck")
	purego.RegisterLibFunc(&chromaStringFree, libHandle, "chroma_string_free")
	purego.RegisterLibFunc(&chromaGetLastError, libHandle, "chroma_get_last_error")
	purego.RegisterLibFunc(&chromaVersion, libHandle, "chroma_version")
	return nil
}

func getLastError() string {
	ptr := chromaGetLastError()
	if ptr == nil {
		return ""
	}
	return goStringFromPtr(ptr)
}

func goStringFromPtr(ptr *byte) string {
	if ptr == nil {
		return ""
	}
	var n uintptr
	q := unsafe.Pointer(ptr)
	for *(*byte)(unsafe.Add(q, n)) != 0 {
		n++
	}
	return string(unsafe.Slice(ptr, n))
}

func cStringFromGo(s string) []byte {
	return append([]byte(s), 0)
}

// Server represents a running Chroma server instance.
type Server struct {
	handle uintptr
	port   int
	addr   string
}

// StartServerConfig contains configuration options for starting a server.
type StartServerConfig struct {
	ConfigPath   string // Path to YAML config file
	ConfigString string // YAML config string (used if ConfigPath is empty)
}

// StartServer starts a new Chroma server with the given configuration.
func StartServer(config StartServerConfig) (*Server, error) {
	if libHandle == 0 {
		return nil, ErrLibraryNotLoaded
	}

	var handle uintptr
	switch {
	case config.ConfigPath != "":
		pathBytes := cStringFromGo(config.ConfigPath)
		handle = chromaServerStart(&pathBytes[0])
	case config.ConfigString != "":
		yamlBytes := cStringFromGo(config.ConfigString)
		handle = chromaServerStartFromString(&yamlBytes[0])
	default:
		return nil, errors.New("either ConfigPath or ConfigString must be provided")
	}

	if handle == 0 {
		return nil, errors.Wrap(ErrNullPointer, getLastError())
	}

	port := chromaServerPort(handle)
	addrPtr := chromaServerAddress(handle)
	addr := ""
	if addrPtr != nil {
		addr = goStringFromPtr(addrPtr)
	}

	server := &Server{
		handle: handle,
		port:   int(port),
		addr:   addr,
	}

	runtime.SetFinalizer(server, func(s *Server) {
		_ = s.Close()
	})

	return server, nil
}

// Port returns the port the server is listening on.
func (s *Server) Port() int {
	return s.port
}

// Address returns the address the server is listening on.
func (s *Server) Address() string {
	return s.addr
}

// URL returns the full URL of the server (e.g., "http://127.0.0.1:8000").
func (s *Server) URL() string {
	return fmt.Sprintf("http://%s:%d", s.addr, s.port)
}

// Stop gracefully stops the server.
func (s *Server) Stop() error {
	if s.handle == 0 {
		return ErrServerNotStarted
	}

	rc := chromaServerStop(s.handle)
	if rc != Success {
		return errorFromCode(rc, getLastError())
	}
	return nil
}

// Close stops the server and frees resources.
func (s *Server) Close() error {
	if s.handle == 0 {
		return nil
	}

	_ = s.Stop() // Ignore error, server might already be stopped
	chromaServerFree(s.handle)
	s.handle = 0
	return nil
}

// Version returns the version of the Chroma shim library.
func Version() string {
	if libHandle == 0 {
		return ""
	}
	ptr := chromaVersion()
	if ptr == nil {
		return ""
	}
	return goStringFromPtr(ptr)
}
