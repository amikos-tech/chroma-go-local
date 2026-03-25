package tech.amikos.chroma.local.panama;

import java.lang.foreign.Arena;
import java.lang.foreign.FunctionDescriptor;
import java.lang.foreign.Linker;
import java.lang.foreign.MemorySegment;
import java.lang.foreign.SymbolLookup;
import java.lang.foreign.ValueLayout;
import java.lang.invoke.MethodHandle;
import java.nio.file.Path;
import java.util.Locale;
import java.util.concurrent.atomic.AtomicBoolean;
import tech.amikos.chroma.local.core.ChromaException;
import tech.amikos.chroma.local.core.ChromaRuntime;
import tech.amikos.chroma.local.core.EmbeddedSession;
import tech.amikos.chroma.local.core.ServerSession;

// Not thread-safe: FFI calls are not serialized. Use from a single thread until Phase 8
// wires AbstractChromaRuntime's FFI lock.
public final class PanamaChromaRuntime implements ChromaRuntime {
    private static final long MAX_C_STRING_LEN = 1L << 20;
    private static final boolean WINDOWS_OS = System.getProperty("os.name", "")
            .toLowerCase(Locale.ROOT)
            .contains("win");

    private final Arena arena;
    private final MethodHandle chromaVersion;
    private final MethodHandle chromaGetLastError;
    private final MethodHandle chromaStringFree;
    private final MethodHandle chromaEmbeddedStartFromString;
    private final MethodHandle chromaEmbeddedFree;
    private final MethodHandle chromaServerStartFromString;
    private final MethodHandle chromaServerStop;
    private final MethodHandle chromaServerFree;
    private final MethodHandle chromaServerPort;
    private final MethodHandle chromaServerAddress;
    private final MethodHandle chromaServerPersistPath;
    private final AtomicBoolean closed;

    private PanamaChromaRuntime(
            Arena arena,
            MethodHandle chromaVersion,
            MethodHandle chromaGetLastError,
            MethodHandle chromaStringFree,
            MethodHandle chromaEmbeddedStartFromString,
            MethodHandle chromaEmbeddedFree,
            MethodHandle chromaServerStartFromString,
            MethodHandle chromaServerStop,
            MethodHandle chromaServerFree,
            MethodHandle chromaServerPort,
            MethodHandle chromaServerAddress,
            MethodHandle chromaServerPersistPath) {
        this.arena = arena;
        this.chromaVersion = chromaVersion;
        this.chromaGetLastError = chromaGetLastError;
        this.chromaStringFree = chromaStringFree;
        this.chromaEmbeddedStartFromString = chromaEmbeddedStartFromString;
        this.chromaEmbeddedFree = chromaEmbeddedFree;
        this.chromaServerStartFromString = chromaServerStartFromString;
        this.chromaServerStop = chromaServerStop;
        this.chromaServerFree = chromaServerFree;
        this.chromaServerPort = chromaServerPort;
        this.chromaServerAddress = chromaServerAddress;
        this.chromaServerPersistPath = chromaServerPersistPath;
        this.closed = new AtomicBoolean(false);
    }

    public static PanamaChromaRuntime init(String libraryPath) {
        if (libraryPath == null || libraryPath.trim().isEmpty()) {
            throw new IllegalArgumentException("libraryPath must be set");
        }

        Path normalized = Path.of(libraryPath).toAbsolutePath().normalize();
        Arena arena = Arena.ofShared();
        boolean initialized = false;
        try {
            Linker linker = Linker.nativeLinker();
            SymbolLookup library = SymbolLookup.libraryLookup(normalized, arena);

            MethodHandle chromaVersion = linker.downcallHandle(
                    requireSymbol(library, "chroma_version"),
                    FunctionDescriptor.of(ValueLayout.ADDRESS));
            MethodHandle chromaGetLastError = linker.downcallHandle(
                    requireSymbol(library, "chroma_get_last_error"),
                    FunctionDescriptor.of(ValueLayout.ADDRESS));
            MethodHandle chromaStringFree = linker.downcallHandle(
                    requireSymbol(library, "chroma_string_free"),
                    FunctionDescriptor.ofVoid(ValueLayout.ADDRESS));
            MethodHandle chromaEmbeddedStartFromString = linker.downcallHandle(
                    requireSymbol(library, "chroma_embedded_start_from_string"),
                    FunctionDescriptor.of(ValueLayout.ADDRESS, ValueLayout.ADDRESS));
            MethodHandle chromaEmbeddedFree = linker.downcallHandle(
                    requireSymbol(library, "chroma_embedded_free"),
                    FunctionDescriptor.ofVoid(ValueLayout.ADDRESS));
            MethodHandle chromaServerStartFromString = linker.downcallHandle(
                    requireSymbol(library, "chroma_server_start_from_string"),
                    FunctionDescriptor.of(ValueLayout.ADDRESS, ValueLayout.ADDRESS));
            MethodHandle chromaServerStop = linker.downcallHandle(
                    requireSymbol(library, "chroma_server_stop"),
                    FunctionDescriptor.of(ValueLayout.JAVA_INT, ValueLayout.ADDRESS));
            MethodHandle chromaServerFree = linker.downcallHandle(
                    requireSymbol(library, "chroma_server_free"),
                    FunctionDescriptor.ofVoid(ValueLayout.ADDRESS));
            MethodHandle chromaServerPort = linker.downcallHandle(
                    requireSymbol(library, "chroma_server_port"),
                    FunctionDescriptor.of(ValueLayout.JAVA_INT, ValueLayout.ADDRESS));
            MethodHandle chromaServerAddress = linker.downcallHandle(
                    requireSymbol(library, "chroma_server_address"),
                    FunctionDescriptor.of(ValueLayout.ADDRESS, ValueLayout.ADDRESS));
            MethodHandle chromaServerPersistPath = linker.downcallHandle(
                    requireSymbol(library, "chroma_server_persist_path"),
                    FunctionDescriptor.of(ValueLayout.ADDRESS, ValueLayout.ADDRESS));

            PanamaChromaRuntime runtime = new PanamaChromaRuntime(
                    arena,
                    chromaVersion,
                    chromaGetLastError,
                    chromaStringFree,
                    chromaEmbeddedStartFromString,
                    chromaEmbeddedFree,
                    chromaServerStartFromString,
                    chromaServerStop,
                    chromaServerFree,
                    chromaServerPort,
                    chromaServerAddress,
                    chromaServerPersistPath);
            initialized = true;
            return runtime;
        } catch (UnsatisfiedLinkError | RuntimeException e) {
            throw new ChromaException("failed to initialize Panama runtime from " + normalized, e);
        } finally {
            if (!initialized) {
                arena.close();
            }
        }
    }

    @Override
    public String version() {
        ensureOpen();
        try {
            // chroma_version returns a pointer to static read-only data in the shared library.
            // Do not call chroma_string_free on this pointer.
            MemorySegment ptr = (MemorySegment) chromaVersion.invokeExact();
            if (ptr.equals(MemorySegment.NULL)) {
                throw new ChromaException("chroma_version returned NULL");
            }
            return ptr.reinterpret(MAX_C_STRING_LEN).getString(0);
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to read chroma_version", t);
        }
    }

    @Override
    public EmbeddedSession startEmbedded(String configYaml) {
        ensureOpen();
        if (configYaml == null || configYaml.isBlank()) {
            throw new IllegalArgumentException("configYaml must be set");
        }

        try (Arena callArena = Arena.ofConfined()) {
            MemorySegment yaml = callArena.allocateFrom(configYaml);
            MemorySegment handle = (MemorySegment) chromaEmbeddedStartFromString.invokeExact(yaml);
            if (handle.equals(MemorySegment.NULL)) {
                throw new ChromaException(lastError("embedded startup failed"));
            }
            return new EmbeddedSession(handle.address(), this::embeddedFree);
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to start embedded runtime", t);
        }
    }

    @Override
    public ServerSession startServer(String configYaml) {
        ensureOpen();
        if (configYaml == null || configYaml.isBlank()) {
            throw new IllegalArgumentException("configYaml must be set");
        }
        try (Arena callArena = Arena.ofConfined()) {
            MemorySegment yaml = callArena.allocateFrom(configYaml);
            MemorySegment handle = (MemorySegment) chromaServerStartFromString.invokeExact(yaml);
            if (handle.equals(MemorySegment.NULL)) {
                throw new ChromaException(lastError("server startup failed"));
            }
            return new ServerSession(
                    handle.address(),
                    this::serverStop,
                    this::serverFree,
                    this::serverPort,
                    this::serverAddress,
                    this::serverPersistPath);
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to start server runtime", t);
        }
    }

    private void serverStop(long handleAddress) {
        if (handleAddress == 0L) return;
        try {
            int rc = (int) chromaServerStop.invokeExact(MemorySegment.ofAddress(handleAddress));
            if (rc != 0) {
                throw new ChromaException(lastError("server stop failed (rc=" + rc + ")"));
            }
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to stop server", t);
        }
    }

    private void serverFree(long handleAddress) {
        if (handleAddress == 0L) return;
        try {
            chromaServerFree.invokeExact(MemorySegment.ofAddress(handleAddress));
        } catch (Throwable t) {
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to free server handle", t);
        }
    }

    private int serverPort(long handleAddress) {
        try {
            int port = (int) chromaServerPort.invokeExact(MemorySegment.ofAddress(handleAddress));
            if (port < 0) {
                throw new ChromaException(lastError("chroma_server_port returned " + port));
            }
            return port;
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to read server port", t);
        }
    }

    private String serverAddress(long handleAddress) {
        try {
            MemorySegment ptr = (MemorySegment) chromaServerAddress.invokeExact(
                    MemorySegment.ofAddress(handleAddress));
            if (ptr.equals(MemorySegment.NULL)) {
                throw new ChromaException("chroma_server_address returned NULL");
            }
            return ptr.reinterpret(MAX_C_STRING_LEN).getString(0);
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to read server address", t);
        }
    }

    private String serverPersistPath(long handleAddress) {
        try {
            MemorySegment ptr = (MemorySegment) chromaServerPersistPath.invokeExact(
                    MemorySegment.ofAddress(handleAddress));
            if (ptr.equals(MemorySegment.NULL)) {
                throw new ChromaException("chroma_server_persist_path returned NULL");
            }
            return ptr.reinterpret(MAX_C_STRING_LEN).getString(0);
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to read server persist path", t);
        }
    }

    private void embeddedFree(long handleAddress) {
        if (handleAddress == 0L) {
            return;
        }
        try {
            chromaEmbeddedFree.invokeExact(MemorySegment.ofAddress(handleAddress));
        } catch (Throwable t) {
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to free embedded handle", t);
        }
    }

    private String lastError(String fallback) {
        try {
            MemorySegment ptr = (MemorySegment) chromaGetLastError.invokeExact();
            if (ptr.equals(MemorySegment.NULL)) {
                return fallback;
            }
            String message;
            try {
                message = ptr.reinterpret(MAX_C_STRING_LEN).getString(0);
            } finally {
                chromaStringFree.invokeExact(ptr);
            }
            if (message == null || message.isBlank()) {
                return fallback;
            }
            return message;
        } catch (Throwable t) {
            if (t instanceof Error error) {
                throw error;
            }
            String detail = t.getMessage();
            if (detail == null || detail.isBlank()) {
                return fallback + " (failed to retrieve native error details)";
            }
            return fallback + " (failed to retrieve native error details: " + detail + ")";
        }
    }

    private static MemorySegment requireSymbol(SymbolLookup library, String name) {
        return library
                .find(name)
                .orElseThrow(() -> new ChromaException("missing symbol: " + name));
    }

    @Override
    public void close() {
        if (closed.compareAndSet(false, true)) {
            // Workaround for unstable JVM crashes seen when unloading the Panama-linked DLL on
            // windows-latest CI runners (Temurin 22). The process exits shortly after tests, so
            // keeping the library loaded for process lifetime is acceptable.
            if (WINDOWS_OS) {
                return;
            }
            try {
                arena.close();
            } catch (IllegalStateException e) {
                closed.set(false);
                throw new ChromaException(
                        "failed to close Panama runtime; ensure all EmbeddedSession instances are closed first",
                        e);
            } catch (RuntimeException | Error e) {
                closed.set(false);
                throw e;
            }
        }
    }

    private void ensureOpen() {
        if (closed.get()) {
            throw new IllegalStateException("runtime is closed");
        }
    }
}
