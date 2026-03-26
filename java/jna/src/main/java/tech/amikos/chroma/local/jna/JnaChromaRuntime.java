package tech.amikos.chroma.local.jna;

import com.sun.jna.Library;
import com.sun.jna.Native;
import com.sun.jna.Pointer;
import java.nio.file.Path;
import java.util.concurrent.atomic.AtomicBoolean;
import tech.amikos.chroma.local.core.AbstractChromaRuntime;
import tech.amikos.chroma.local.core.ChromaException;
import tech.amikos.chroma.local.core.EmbeddedSession;
import tech.amikos.chroma.local.core.ServerSession;

public final class JnaChromaRuntime extends AbstractChromaRuntime {
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
    protected String readBorrowedString(long address) {
        return new Pointer(address).getString(0);
    }

    @Override
    protected String readOwnedString(long address) {
        Pointer ptr = new Pointer(address);
        try {
            return ptr.getString(0);
        } finally {
            bindings.chroma_string_free(ptr);
        }
    }

    @Override
    protected String readLastError() {
        Pointer ptr = bindings.chroma_get_last_error();
        if (ptr == null || Pointer.nativeValue(ptr) == 0L) {
            return null;
        }
        try {
            return ptr.getString(0);
        } finally {
            bindings.chroma_string_free(ptr);
        }
    }

    @Override
    public String version() {
        ensureOpen();
        return callFfiBorrowedString(() -> Pointer.nativeValue(bindings.chroma_version()));
    }

    @Override
    public EmbeddedSession startEmbedded(String configYaml) {
        ensureOpen();
        if (configYaml == null || configYaml.isBlank()) {
            throw new IllegalArgumentException("configYaml must be set");
        }
        long handle = callFfiHandle(
                () -> Pointer.nativeValue(bindings.chroma_embedded_start_from_string(configYaml)));
        return new EmbeddedSession(handle, this::embeddedFree);
    }

    @Override
    public ServerSession startServer(String configYaml) {
        ensureOpen();
        if (configYaml == null || configYaml.isBlank()) {
            throw new IllegalArgumentException("configYaml must be set");
        }
        long handle = callFfiHandle(
                () -> Pointer.nativeValue(bindings.chroma_server_start_from_string(configYaml)));
        return new ServerSession(
                handle,
                this::serverStop,
                this::serverFree,
                this::serverPort,
                this::serverAddress,
                this::serverPersistPath);
    }

    private void serverStop(long handle) {
        if (handle == 0L) return;
        callFfiVoid(() -> {
            int rc = bindings.chroma_server_stop(new Pointer(handle));
            if (rc != 0) {
                throw new ChromaException("server stop failed (rc=" + rc + ")");
            }
        });
    }

    private void serverFree(long handle) {
        if (handle == 0L) return;
        bindings.chroma_server_free(new Pointer(handle));
    }

    private int serverPort(long handle) {
        return (int) callFfiHandle(() -> {
            int p = bindings.chroma_server_port(new Pointer(handle));
            if (p < 0) return 0L;
            return (long) p;
        });
    }

    private String serverAddress(long handle) {
        return callFfiBorrowedString(
                () -> Pointer.nativeValue(bindings.chroma_server_address(new Pointer(handle))));
    }

    private String serverPersistPath(long handle) {
        return callFfiBorrowedString(
                () -> Pointer.nativeValue(bindings.chroma_server_persist_path(new Pointer(handle))));
    }

    private void embeddedFree(long handle) {
        if (handle == 0L) return;
        bindings.chroma_embedded_free(new Pointer(handle));
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
