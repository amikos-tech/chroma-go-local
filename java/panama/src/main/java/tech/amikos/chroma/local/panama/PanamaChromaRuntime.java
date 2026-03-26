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
import tech.amikos.chroma.local.core.AbstractChromaRuntime;
import tech.amikos.chroma.local.core.ChromaException;
import tech.amikos.chroma.local.core.EmbeddedSession;
import tech.amikos.chroma.local.core.ServerSession;

public final class PanamaChromaRuntime extends AbstractChromaRuntime {
    private static final long MAX_C_STRING_LEN = 1L << 20;
    private static final boolean WINDOWS_OS = System.getProperty("os.name", "")
            .toLowerCase(Locale.ROOT)
            .contains("win");

    private record Ffi(
            MethodHandle version,
            MethodHandle getLastError,
            MethodHandle stringFree,
            MethodHandle embeddedStart,
            MethodHandle embeddedFree,
            MethodHandle serverStart,
            MethodHandle serverStop,
            MethodHandle serverFree,
            MethodHandle serverPort,
            MethodHandle serverAddress,
            MethodHandle serverPersistPath) {}

    private final Arena arena;
    private final Ffi ffi;

    private PanamaChromaRuntime(Arena arena, Ffi ffi) {
        this.arena = arena;
        this.ffi = ffi;
    }

    public static PanamaChromaRuntime init(String libraryPath) {
        Path normalized = validateLibraryPath(libraryPath);
        Arena arena = Arena.ofShared();
        boolean initialized = false;
        try {
            Linker linker = Linker.nativeLinker();
            SymbolLookup library = SymbolLookup.libraryLookup(normalized, arena);

            Ffi ffi = new Ffi(
                    linker.downcallHandle(
                            requireSymbol(library, "chroma_version"),
                            FunctionDescriptor.of(ValueLayout.ADDRESS)),
                    linker.downcallHandle(
                            requireSymbol(library, "chroma_get_last_error"),
                            FunctionDescriptor.of(ValueLayout.ADDRESS)),
                    linker.downcallHandle(
                            requireSymbol(library, "chroma_string_free"),
                            FunctionDescriptor.ofVoid(ValueLayout.ADDRESS)),
                    linker.downcallHandle(
                            requireSymbol(library, "chroma_embedded_start_from_string"),
                            FunctionDescriptor.of(ValueLayout.ADDRESS, ValueLayout.ADDRESS)),
                    linker.downcallHandle(
                            requireSymbol(library, "chroma_embedded_free"),
                            FunctionDescriptor.ofVoid(ValueLayout.ADDRESS)),
                    linker.downcallHandle(
                            requireSymbol(library, "chroma_server_start_from_string"),
                            FunctionDescriptor.of(ValueLayout.ADDRESS, ValueLayout.ADDRESS)),
                    linker.downcallHandle(
                            requireSymbol(library, "chroma_server_stop"),
                            FunctionDescriptor.of(ValueLayout.JAVA_INT, ValueLayout.ADDRESS)),
                    linker.downcallHandle(
                            requireSymbol(library, "chroma_server_free"),
                            FunctionDescriptor.ofVoid(ValueLayout.ADDRESS)),
                    linker.downcallHandle(
                            requireSymbol(library, "chroma_server_port"),
                            FunctionDescriptor.of(ValueLayout.JAVA_INT, ValueLayout.ADDRESS)),
                    linker.downcallHandle(
                            requireSymbol(library, "chroma_server_address"),
                            FunctionDescriptor.of(ValueLayout.ADDRESS, ValueLayout.ADDRESS)),
                    linker.downcallHandle(
                            requireSymbol(library, "chroma_server_persist_path"),
                            FunctionDescriptor.of(ValueLayout.ADDRESS, ValueLayout.ADDRESS)));

            PanamaChromaRuntime runtime = new PanamaChromaRuntime(arena, ffi);
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
    protected String readBorrowedString(long address) {
        return MemorySegment.ofAddress(address).reinterpret(MAX_C_STRING_LEN).getString(0);
    }

    @Override
    protected String readOwnedString(long address) {
        MemorySegment ptr = MemorySegment.ofAddress(address);
        Throwable readError = null;
        try {
            return ptr.reinterpret(MAX_C_STRING_LEN).getString(0);
        } catch (Throwable t) {
            readError = t;
            throw t;
        } finally {
            try {
                ffi.stringFree().invokeExact(ptr);
            } catch (Throwable t) {
                if (t instanceof Error error) {
                    if (readError != null) error.addSuppressed(readError);
                    throw error;
                }
                if (readError != null) {
                    readError.addSuppressed(t);
                } else {
                    throw new ChromaException("failed to free native string", t);
                }
            }
        }
    }

    @Override
    protected String readLastError() {
        try {
            MemorySegment ptr = (MemorySegment) ffi.getLastError().invokeExact();
            if (ptr.equals(MemorySegment.NULL)) return null;
            try {
                return ptr.reinterpret(MAX_C_STRING_LEN).getString(0);
            } finally {
                ffi.stringFree().invokeExact(ptr);
            }
        } catch (Throwable t) {
            if (t instanceof Error error) throw error;
            return null;
        }
    }

    @Override
    protected String doVersion() {
        return callFfiBorrowedString(() -> {
            try {
                MemorySegment ptr = (MemorySegment) ffi.version().invokeExact();
                return ptr.address();
            } catch (Throwable t) {
                if (t instanceof Error error) throw error;
                throw new ChromaException("failed to read chroma_version", t);
            }
        });
    }

    @Override
    protected EmbeddedSession doStartEmbedded(String configYaml) {
        long handle = callFfiHandle(() -> {
            try (Arena callArena = Arena.ofConfined()) {
                MemorySegment yaml = callArena.allocateFrom(configYaml);
                MemorySegment h = (MemorySegment) ffi.embeddedStart().invokeExact(yaml);
                return h.address();
            } catch (Throwable t) {
                if (t instanceof Error error) throw error;
                throw new ChromaException("failed to start embedded runtime", t);
            }
        });
        return new EmbeddedSession(handle, this::embeddedFree);
    }

    @Override
    protected ServerSession doStartServer(String configYaml) {
        long handle = callFfiHandle(() -> {
            try (Arena callArena = Arena.ofConfined()) {
                MemorySegment yaml = callArena.allocateFrom(configYaml);
                MemorySegment h = (MemorySegment) ffi.serverStart().invokeExact(yaml);
                return h.address();
            } catch (Throwable t) {
                if (t instanceof Error error) throw error;
                throw new ChromaException("failed to start server runtime", t);
            }
        });
        return new ServerSession(
                handle,
                this::serverStop,
                this::serverFree,
                this::serverPort,
                this::serverAddress,
                this::serverPersistPath);
    }

    private void serverStop(long handleAddress) {
        if (handleAddress == 0L) return;
        callFfiInt(() -> {
            try {
                return (int) ffi.serverStop().invokeExact(MemorySegment.ofAddress(handleAddress));
            } catch (Throwable t) {
                if (t instanceof Error error) throw error;
                throw new ChromaException("failed to stop server", t);
            }
        });
    }

    // Cannot use callFfiFree: invokeExact is signature-polymorphic and
    // generates incorrect bytecode when called inside a lambda body.
    private void serverFree(long handleAddress) {
        if (handleAddress == 0L) return;
        ffiLock();
        try {
            ffi.serverFree().invokeExact(MemorySegment.ofAddress(handleAddress));
        } catch (Throwable t) {
            if (t instanceof Error error) throw error;
            throw new ChromaException("failed to free server handle", t);
        } finally {
            ffiUnlock();
        }
    }

    private int serverPort(long handleAddress) {
        return callFfiInt(() -> {
            try {
                return (int) ffi.serverPort().invokeExact(MemorySegment.ofAddress(handleAddress));
            } catch (Throwable t) {
                if (t instanceof Error error) throw error;
                throw new ChromaException("failed to read server port", t);
            }
        });
    }

    private String serverAddress(long handleAddress) {
        return callFfiBorrowedString(() -> {
            try {
                MemorySegment ptr = (MemorySegment) ffi.serverAddress().invokeExact(
                        MemorySegment.ofAddress(handleAddress));
                return ptr.address();
            } catch (Throwable t) {
                if (t instanceof Error error) throw error;
                throw new ChromaException("failed to read server address", t);
            }
        });
    }

    private String serverPersistPath(long handleAddress) {
        return callFfiBorrowedString(() -> {
            try {
                MemorySegment ptr = (MemorySegment) ffi.serverPersistPath().invokeExact(
                        MemorySegment.ofAddress(handleAddress));
                return ptr.address();
            } catch (Throwable t) {
                if (t instanceof Error error) throw error;
                throw new ChromaException("failed to read server persist path", t);
            }
        });
    }

    // Same invokeExact constraint as serverFree — cannot use callFfiFree.
    private void embeddedFree(long handleAddress) {
        if (handleAddress == 0L) return;
        ffiLock();
        try {
            ffi.embeddedFree().invokeExact(MemorySegment.ofAddress(handleAddress));
        } catch (Throwable t) {
            if (t instanceof Error error) throw error;
            throw new ChromaException("failed to free embedded handle", t);
        } finally {
            ffiUnlock();
        }
    }

    private static MemorySegment requireSymbol(SymbolLookup library, String name) {
        return library
                .find(name)
                .orElseThrow(() -> new ChromaException("missing symbol: " + name));
    }

    @Override
    protected void doClose() {
        // Skip arena.close on Windows — JVM crashes when unloading Panama-linked DLLs on Temurin 22
        if (WINDOWS_OS) return;
        try {
            arena.close();
        } catch (IllegalStateException e) {
            throw new ChromaException(
                    "failed to close Panama runtime; ensure all sessions are closed first", e);
        }
    }
}
