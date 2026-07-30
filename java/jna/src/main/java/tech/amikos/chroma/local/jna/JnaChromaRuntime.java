package tech.amikos.chroma.local.jna;

import com.sun.jna.Library;
import com.sun.jna.Native;
import com.sun.jna.Pointer;
import java.nio.file.Path;
import tech.amikos.chroma.local.core.AbstractChromaRuntime;
import tech.amikos.chroma.local.core.BackupExecutor;
import tech.amikos.chroma.local.core.BackupMode;
import tech.amikos.chroma.local.core.ChromaException;
import tech.amikos.chroma.local.core.MaintenanceExecutor;
import tech.amikos.chroma.local.core.CompactionResult;
import tech.amikos.chroma.local.core.EmbeddedSession;
import tech.amikos.chroma.local.core.RebuildCollectionResult;
import tech.amikos.chroma.local.core.ServerSession;
import tech.amikos.chroma.local.core.WALPruneResult;

public final class JnaChromaRuntime extends AbstractChromaRuntime {
    private final JnaBindings bindings;

    private interface JnaBindings extends Library {
        Pointer chroma_version();

        Pointer chroma_get_last_error();

        void chroma_string_free(Pointer value);

        Pointer chroma_embedded_start_from_string(String configYaml);

        void chroma_embedded_free(Pointer handle);

        Pointer chroma_embedded_rebuild_collection(Pointer handle, String requestJson);

        Pointer chroma_embedded_compact_collection(Pointer handle, String requestJson);

        Pointer chroma_embedded_compact_all(Pointer handle, String requestJson);

        Pointer chroma_embedded_prune_wal_collection(Pointer handle, String requestJson);

        Pointer chroma_embedded_prune_wal_all(Pointer handle, String requestJson);

        Pointer chroma_server_start_from_string(String configYaml);

        int chroma_server_stop(Pointer handle);

        void chroma_server_free(Pointer handle);

        int chroma_server_port(Pointer handle);

        Pointer chroma_server_address(Pointer handle);

        int chroma_server_tls_enabled(Pointer handle);

        Pointer chroma_embedded_persist_path(Pointer handle);

        Pointer chroma_server_persist_path(Pointer handle);
    }

    private JnaChromaRuntime(JnaBindings bindings) {
        this.bindings = bindings;
    }

    public static JnaChromaRuntime init(String libraryPath) {
        Path normalized = validateLibraryPath(libraryPath);
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
    protected String doVersion() {
        return callFfiBorrowedString(() -> Pointer.nativeValue(bindings.chroma_version()));
    }

    @Override
    protected EmbeddedSession doStartEmbedded(String configYaml) {
        long handle = callFfiHandle(
                () -> Pointer.nativeValue(bindings.chroma_embedded_start_from_string(configYaml)));
        String persistPath = embeddedPersistPath(handle);
        String version = doVersion();
        return new EmbeddedSession(
                handle,
                this::embeddedFree,
                (h, json) -> callFfiJson(
                        () -> Pointer.nativeValue(bindings.chroma_embedded_rebuild_collection(new Pointer(h), json)),
                        RebuildCollectionResult.class),
                (h, json) -> callFfiJson(
                        () -> Pointer.nativeValue(bindings.chroma_embedded_compact_collection(new Pointer(h), json)),
                        CompactionResult.class),
                (h, json) -> callFfiJson(
                        () -> Pointer.nativeValue(bindings.chroma_embedded_compact_all(new Pointer(h), json)),
                        CompactionResult.class),
                (h, json) -> callFfiJson(
                        () -> Pointer.nativeValue(bindings.chroma_embedded_prune_wal_collection(new Pointer(h), json)),
                        WALPruneResult.class),
                (h, json) -> callFfiJson(
                        () -> Pointer.nativeValue(bindings.chroma_embedded_prune_wal_all(new Pointer(h), json)),
                        WALPruneResult.class),
                opts -> BackupExecutor.execute(BackupMode.EMBEDDED, persistPath, version, opts,
                        () -> embeddedFree(handle), () -> doStartEmbedded(configYaml)));
    }

    @Override
    protected ServerSession doStartServer(String configYaml) {
        long handle = callFfiHandle(
                () -> Pointer.nativeValue(bindings.chroma_server_start_from_string(configYaml)));
        String persistPath = serverPersistPath(handle);
        String version = doVersion();
        return new ServerSession(
                handle,
                this::serverStop,
                this::serverFree,
                this::serverPort,
                this::serverAddress,
                this::serverPersistPath,
                () -> serverTlsEnabled(handle),
                opts -> BackupExecutor.execute(BackupMode.SERVER, persistPath, version, opts,
                        () -> { try { serverStop(handle); } finally { serverFree(handle); } },
                        () -> doStartServer(configYaml)),
                opts -> MaintenanceExecutor.execute(configYaml,
                        () -> { try { serverStop(handle); } finally { serverFree(handle); } },
                        this::doStartEmbedded, this::doStartServer,
                        emb -> emb.rebuildCollection(opts)),
                req -> MaintenanceExecutor.execute(configYaml,
                        () -> { try { serverStop(handle); } finally { serverFree(handle); } },
                        this::doStartEmbedded, this::doStartServer,
                        emb -> emb.compactCollection(req)),
                req -> MaintenanceExecutor.execute(configYaml,
                        () -> { try { serverStop(handle); } finally { serverFree(handle); } },
                        this::doStartEmbedded, this::doStartServer,
                        emb -> emb.compactAll(req)),
                opts -> MaintenanceExecutor.execute(configYaml,
                        () -> { try { serverStop(handle); } finally { serverFree(handle); } },
                        this::doStartEmbedded, this::doStartServer,
                        emb -> emb.pruneCollectionWAL(opts)),
                opts -> MaintenanceExecutor.execute(configYaml,
                        () -> { try { serverStop(handle); } finally { serverFree(handle); } },
                        this::doStartEmbedded, this::doStartServer,
                        emb -> emb.pruneAllWAL(opts)));
    }

    private void serverStop(long handle) {
        if (handle == 0L) return;
        callFfiInt(() -> bindings.chroma_server_stop(new Pointer(handle)));
    }

    private void serverFree(long handle) {
        callFfiFree(handle, () -> bindings.chroma_server_free(new Pointer(handle)));
    }

    private int serverPort(long handle) {
        return callFfiInt(() -> bindings.chroma_server_port(new Pointer(handle)));
    }

    private String serverAddress(long handle) {
        return callFfiBorrowedString(
                () -> Pointer.nativeValue(bindings.chroma_server_address(new Pointer(handle))));
    }

    private boolean serverTlsEnabled(long handle) {
        return callFfiInt(() -> bindings.chroma_server_tls_enabled(new Pointer(handle))) > 0;
    }

    private String embeddedPersistPath(long handle) {
        return callFfiBorrowedString(
                () -> Pointer.nativeValue(bindings.chroma_embedded_persist_path(new Pointer(handle))));
    }

    private String serverPersistPath(long handle) {
        return callFfiBorrowedString(
                () -> Pointer.nativeValue(bindings.chroma_server_persist_path(new Pointer(handle))));
    }

    private void embeddedFree(long handle) {
        callFfiFree(handle, () -> bindings.chroma_embedded_free(new Pointer(handle)));
    }
}
