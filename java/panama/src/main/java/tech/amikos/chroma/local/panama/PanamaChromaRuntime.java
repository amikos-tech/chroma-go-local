package tech.amikos.chroma.local.panama;

import java.lang.foreign.Arena;
import java.lang.foreign.FunctionDescriptor;
import java.lang.foreign.Linker;
import java.lang.foreign.MemorySegment;
import java.lang.foreign.SymbolLookup;
import java.lang.foreign.ValueLayout;
import java.lang.invoke.MethodHandle;
import java.nio.file.Path;
import java.util.concurrent.atomic.AtomicBoolean;
import tech.amikos.chroma.local.core.ChromaException;
import tech.amikos.chroma.local.core.ChromaRuntime;
import tech.amikos.chroma.local.core.EmbeddedSession;

public final class PanamaChromaRuntime implements ChromaRuntime {
    private static final long MAX_C_STRING_LEN = 1L << 20;

    private final Arena arena;
    private final MethodHandle chromaVersion;
    private final MethodHandle chromaGetLastError;
    private final MethodHandle chromaStringFree;
    private final MethodHandle chromaEmbeddedStartFromString;
    private final MethodHandle chromaEmbeddedFree;
    private final AtomicBoolean closed;

    private PanamaChromaRuntime(
            Arena arena,
            MethodHandle chromaVersion,
            MethodHandle chromaGetLastError,
            MethodHandle chromaStringFree,
            MethodHandle chromaEmbeddedStartFromString,
            MethodHandle chromaEmbeddedFree) {
        this.arena = arena;
        this.chromaVersion = chromaVersion;
        this.chromaGetLastError = chromaGetLastError;
        this.chromaStringFree = chromaStringFree;
        this.chromaEmbeddedStartFromString = chromaEmbeddedStartFromString;
        this.chromaEmbeddedFree = chromaEmbeddedFree;
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

            PanamaChromaRuntime runtime = new PanamaChromaRuntime(
                    arena,
                    chromaVersion,
                    chromaGetLastError,
                    chromaStringFree,
                    chromaEmbeddedStartFromString,
                    chromaEmbeddedFree);
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
        try {
            // chroma_version returns a static C string owned by the runtime.
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
}
