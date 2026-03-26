package tech.amikos.chroma.local.jna;

import com.sun.jna.Library;
import com.sun.jna.Native;
import com.sun.jna.Pointer;
import java.nio.file.Path;
import java.util.concurrent.atomic.AtomicBoolean;
import tech.amikos.chroma.local.core.ChromaException;
import tech.amikos.chroma.local.core.ChromaRuntime;
import tech.amikos.chroma.local.core.EmbeddedSession;
import tech.amikos.chroma.local.core.ServerSession;

// Not thread-safe: FFI calls are not serialized. Use from a single thread until Phase 8
// wires AbstractChromaRuntime's FFI lock.
public final class JnaChromaRuntime implements ChromaRuntime {
    private final JnaBindings bindings;
    private final AtomicBoolean closed;

    private interface JnaBindings extends Library {
        Pointer chroma_version();

        Pointer chroma_get_last_error();

        void chroma_string_free(Pointer value);

        Pointer chroma_embedded_start_from_string(String configYaml);

        void chroma_embedded_free(Pointer handle);

        Pointer chroma_server_start_from_string(String configYaml);

        int chroma_server_stop(Pointer handle);

        void chroma_server_free(Pointer handle);

        int chroma_server_port(Pointer handle);

        Pointer chroma_server_address(Pointer handle);

        Pointer chroma_server_persist_path(Pointer handle);
    }

    private JnaChromaRuntime(JnaBindings bindings) {
        this.bindings = bindings;
        this.closed = new AtomicBoolean(false);
    }

    public static JnaChromaRuntime init(String libraryPath) {
        if (libraryPath == null || libraryPath.trim().isEmpty()) {
            throw new IllegalArgumentException("libraryPath must be set");
        }
        Path normalized = Path.of(libraryPath).toAbsolutePath().normalize();
        try {
            JnaBindings bindings = Native.load(normalized.toString(), JnaBindings.class);
            return new JnaChromaRuntime(bindings);
        } catch (UnsatisfiedLinkError | RuntimeException e) {
            throw new ChromaException("failed to initialize JNA runtime from " + normalized, e);
        }
    }

    @Override
    public String version() {
        ensureOpen();
        try {
            // chroma_version returns a pointer to static read-only data in the shared library.
            // Do not call chroma_string_free on this pointer.
            Pointer ptr = bindings.chroma_version();
            if (ptr == null || Pointer.nativeValue(ptr) == 0L) {
                throw new ChromaException("chroma_version returned NULL");
            }
            return ptr.getString(0);
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof UnsatisfiedLinkError e) {
                throw new ChromaException("failed to read chroma_version", e);
            }
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
        try {
            Pointer handle = bindings.chroma_embedded_start_from_string(configYaml);
            if (handle == null || Pointer.nativeValue(handle) == 0L) {
                throw new ChromaException(lastError("embedded startup failed"));
            }
            return new EmbeddedSession(Pointer.nativeValue(handle), this::embeddedFree);
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof UnsatisfiedLinkError e) {
                throw new ChromaException("failed to start embedded runtime", e);
            }
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
        try {
            Pointer handle = bindings.chroma_server_start_from_string(configYaml);
            if (handle == null || Pointer.nativeValue(handle) == 0L) {
                throw new ChromaException(lastError("server startup failed"));
            }
            return new ServerSession(
                    Pointer.nativeValue(handle),
                    this::serverStop,
                    this::serverFree,
                    this::serverPort,
                    this::serverAddress,
                    this::serverPersistPath);
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof UnsatisfiedLinkError e) {
                throw new ChromaException("failed to start server runtime", e);
            }
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to start server runtime", t);
        }
    }

    private void serverStop(long handle) {
        if (handle == 0L) return;
        try {
            int rc = bindings.chroma_server_stop(new Pointer(handle));
            if (rc != 0) {
                throw new ChromaException(lastError("server stop failed (rc=" + rc + ")"));
            }
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof UnsatisfiedLinkError e) {
                throw new ChromaException("failed to stop server", e);
            }
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to stop server", t);
        }
    }

    private void serverFree(long handle) {
        if (handle == 0L) return;
        try {
            bindings.chroma_server_free(new Pointer(handle));
        } catch (Throwable t) {
            if (t instanceof UnsatisfiedLinkError e) {
                throw new ChromaException("failed to free server handle", e);
            }
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to free server handle", t);
        }
    }

    private int serverPort(long handle) {
        try {
            int port = bindings.chroma_server_port(new Pointer(handle));
            if (port < 0) {
                throw new ChromaException(lastError("chroma_server_port returned " + port));
            }
            return port;
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof UnsatisfiedLinkError e) {
                throw new ChromaException("failed to read server port", e);
            }
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to read server port", t);
        }
    }

    private String serverAddress(long handle) {
        try {
            Pointer ptr = bindings.chroma_server_address(new Pointer(handle));
            if (ptr == null || Pointer.nativeValue(ptr) == 0L) {
                throw new ChromaException("chroma_server_address returned NULL");
            }
            return ptr.getString(0);
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof UnsatisfiedLinkError e) {
                throw new ChromaException("failed to read server address", e);
            }
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to read server address", t);
        }
    }

    private String serverPersistPath(long handle) {
        try {
            Pointer ptr = bindings.chroma_server_persist_path(new Pointer(handle));
            if (ptr == null || Pointer.nativeValue(ptr) == 0L) {
                throw new ChromaException("chroma_server_persist_path returned NULL");
            }
            return ptr.getString(0);
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof UnsatisfiedLinkError e) {
                throw new ChromaException("failed to read server persist path", e);
            }
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to read server persist path", t);
        }
    }

    private void embeddedFree(long handle) {
        if (handle == 0L) {
            return;
        }
        try {
            bindings.chroma_embedded_free(new Pointer(handle));
        } catch (Throwable t) {
            if (t instanceof UnsatisfiedLinkError e) {
                throw new ChromaException("failed to free embedded handle", e);
            }
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to free embedded handle", t);
        }
    }

    private String lastError(String fallback) {
        try {
            Pointer ptr = bindings.chroma_get_last_error();
            if (ptr == null || Pointer.nativeValue(ptr) == 0L) {
                return fallback;
            }
            String message;
            try {
                message = ptr.getString(0);
            } finally {
                bindings.chroma_string_free(ptr);
            }
            if (message == null || message.isBlank()) {
                return fallback;
            }
            return message;
        } catch (Throwable t) {
            if (t instanceof Error error && !(error instanceof UnsatisfiedLinkError)) {
                throw error;
            }
            String detail = t.getMessage();
            if (detail == null || detail.isBlank()) {
                return fallback + " (failed to retrieve native error details)";
            }
            return fallback + " (failed to retrieve native error details: " + detail + ")";
        }
    }

    @Override
    public void close() {
        closed.compareAndSet(false, true);
    }

    private void ensureOpen() {
        if (closed.get()) {
            throw new IllegalStateException("runtime is closed");
        }
    }
}
