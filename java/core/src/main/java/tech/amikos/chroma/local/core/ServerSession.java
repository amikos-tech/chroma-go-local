package tech.amikos.chroma.local.core;

import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.locks.ReentrantLock;
import java.util.function.Function;
import java.util.function.LongConsumer;
import java.util.function.LongFunction;
import java.util.function.LongToIntFunction;

public final class ServerSession implements AutoCloseable {
    private final long handle;
    private final AtomicBoolean closed;
    private final ReentrantLock backupLock = new ReentrantLock();
    private final LongConsumer stopAction;
    private final LongConsumer freeAction;
    private final LongToIntFunction portAccessor;
    private final LongFunction<String> addressAccessor;
    private final LongFunction<String> persistPathAccessor;
    private final Function<BackupOptions, BackupResult<ServerSession>> backupAction;

    public ServerSession(long handle, LongConsumer stopAction, LongConsumer freeAction,
                         LongToIntFunction portAccessor, LongFunction<String> addressAccessor,
                         LongFunction<String> persistPathAccessor,
                         Function<BackupOptions, BackupResult<ServerSession>> backupAction) {
        if (handle == 0L) throw new IllegalArgumentException("server handle must be non-zero");
        if (stopAction == null) throw new IllegalArgumentException("stopAction must be set");
        if (freeAction == null) throw new IllegalArgumentException("freeAction must be set");
        if (portAccessor == null) throw new IllegalArgumentException("portAccessor must be set");
        if (addressAccessor == null) throw new IllegalArgumentException("addressAccessor must be set");
        if (persistPathAccessor == null) throw new IllegalArgumentException("persistPathAccessor must be set");
        if (backupAction == null) throw new IllegalArgumentException("backupAction must be set");
        this.handle = handle;
        this.stopAction = stopAction;
        this.freeAction = freeAction;
        this.portAccessor = portAccessor;
        this.addressAccessor = addressAccessor;
        this.persistPathAccessor = persistPathAccessor;
        this.backupAction = backupAction;
        this.closed = new AtomicBoolean(false);
    }

    private void ensureOpen() {
        if (closed.get()) throw new IllegalStateException("session is closed");
    }

    long handle() { ensureOpen(); return handle; }

    public int port() { ensureOpen(); return portAccessor.applyAsInt(handle); }

    public String address() { ensureOpen(); return addressAccessor.apply(handle); }

    public String persistPath() { ensureOpen(); return persistPathAccessor.apply(handle); }

    // TLS not yet supported
    public String url() {
        return "http://" + address() + ":" + port();
    }

    @Override
    public void close() {
        backupLock.lock();
        try {
            if (closed.compareAndSet(false, true)) {
                Throwable stopError = null;
                try {
                    stopAction.accept(handle);
                } catch (Throwable t) {
                    stopError = t;
                }
                try {
                    freeAction.accept(handle);
                } catch (Throwable t) {
                    if (stopError != null) {
                        stopError.addSuppressed(t);
                        throw rethrow(stopError);
                    }
                    throw rethrow(t);
                }
                if (stopError != null) {
                    throw rethrow(stopError);
                }
            }
        } finally {
            backupLock.unlock();
        }
    }

    @SuppressWarnings("unchecked")
    private static <T extends Throwable> RuntimeException rethrow(Throwable t) throws T {
        throw (T) t;
    }

    public RebuildCollectionResult rebuildCollection(RebuildOptions options) {
        ensureOpen();
        throw new UnsupportedOperationException("rebuildCollection is not yet supported for server sessions");
    }

    public RebuildCollectionResult rebuildCollection(String name) {
        return rebuildCollection(RebuildOptions.defaults(name));
    }

    public CompactionResult compactCollection(CompactCollectionRequest request) {
        ensureOpen();
        throw new UnsupportedOperationException("compactCollection is not yet supported for server sessions");
    }

    public CompactionResult compactCollection(String name) {
        return compactCollection(new CompactCollectionRequest.Builder(name).build());
    }

    public CompactionResult compactAll(CompactAllRequest request) {
        ensureOpen();
        throw new UnsupportedOperationException("compactAll is not yet supported for server sessions");
    }

    public WALPruneResult pruneCollectionWAL(WALPruneOptions options) {
        ensureOpen();
        throw new UnsupportedOperationException("pruneCollectionWAL is not yet supported for server sessions");
    }

    public WALPruneResult pruneCollectionWAL(String name) {
        return pruneCollectionWAL(WALPruneOptions.defaults(name));
    }

    /** The {@code name} field in options is ignored. */
    public WALPruneResult pruneAllWAL(WALPruneOptions options) {
        ensureOpen();
        throw new UnsupportedOperationException("pruneAllWAL is not yet supported for server sessions");
    }

    public BackupResult<ServerSession> backup(BackupOptions options) {
        backupLock.lock();
        try {
            ensureOpen();
            if (options == null) throw new IllegalArgumentException("options is required");
            try {
                return backupAction.apply(options);
            } finally {
                closed.set(true);
            }
        } finally {
            backupLock.unlock();
        }
    }
}
