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
    private final Function<RebuildOptions, MaintenanceResult<RebuildCollectionResult, ServerSession>> rebuildAction;
    private final Function<CompactCollectionRequest, MaintenanceResult<CompactionResult, ServerSession>> compactCollectionAction;
    private final Function<CompactAllRequest, MaintenanceResult<CompactionResult, ServerSession>> compactAllAction;
    private final Function<WALPruneOptions, MaintenanceResult<WALPruneResult, ServerSession>> pruneWalCollectionAction;
    private final Function<WALPruneOptions, MaintenanceResult<WALPruneResult, ServerSession>> pruneWalAllAction;

    public ServerSession(long handle, LongConsumer stopAction, LongConsumer freeAction,
                         LongToIntFunction portAccessor, LongFunction<String> addressAccessor,
                         LongFunction<String> persistPathAccessor,
                         Function<BackupOptions, BackupResult<ServerSession>> backupAction,
                         Function<RebuildOptions, MaintenanceResult<RebuildCollectionResult, ServerSession>> rebuildAction,
                         Function<CompactCollectionRequest, MaintenanceResult<CompactionResult, ServerSession>> compactCollectionAction,
                         Function<CompactAllRequest, MaintenanceResult<CompactionResult, ServerSession>> compactAllAction,
                         Function<WALPruneOptions, MaintenanceResult<WALPruneResult, ServerSession>> pruneWalCollectionAction,
                         Function<WALPruneOptions, MaintenanceResult<WALPruneResult, ServerSession>> pruneWalAllAction) {
        if (handle == 0L) throw new IllegalArgumentException("server handle must be non-zero");
        if (stopAction == null) throw new IllegalArgumentException("stopAction must be set");
        if (freeAction == null) throw new IllegalArgumentException("freeAction must be set");
        if (portAccessor == null) throw new IllegalArgumentException("portAccessor must be set");
        if (addressAccessor == null) throw new IllegalArgumentException("addressAccessor must be set");
        if (persistPathAccessor == null) throw new IllegalArgumentException("persistPathAccessor must be set");
        if (backupAction == null) throw new IllegalArgumentException("backupAction must be set");
        if (rebuildAction == null) throw new IllegalArgumentException("rebuildAction must be set");
        if (compactCollectionAction == null) throw new IllegalArgumentException("compactCollectionAction must be set");
        if (compactAllAction == null) throw new IllegalArgumentException("compactAllAction must be set");
        if (pruneWalCollectionAction == null) throw new IllegalArgumentException("pruneWalCollectionAction must be set");
        if (pruneWalAllAction == null) throw new IllegalArgumentException("pruneWalAllAction must be set");
        this.handle = handle;
        this.stopAction = stopAction;
        this.freeAction = freeAction;
        this.portAccessor = portAccessor;
        this.addressAccessor = addressAccessor;
        this.persistPathAccessor = persistPathAccessor;
        this.backupAction = backupAction;
        this.rebuildAction = rebuildAction;
        this.compactCollectionAction = compactCollectionAction;
        this.compactAllAction = compactAllAction;
        this.pruneWalCollectionAction = pruneWalCollectionAction;
        this.pruneWalAllAction = pruneWalAllAction;
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

    public MaintenanceResult<RebuildCollectionResult, ServerSession> rebuildCollection(RebuildOptions options) {
        backupLock.lock();
        try {
            ensureOpen();
            if (options == null) throw new IllegalArgumentException("options is required");
            MaintenanceResult<RebuildCollectionResult, ServerSession> result = rebuildAction.apply(options);
            closed.set(true);
            return result;
        } catch (RuntimeException e) {
            closed.set(true);
            throw e;
        } finally {
            backupLock.unlock();
        }
    }

    public MaintenanceResult<RebuildCollectionResult, ServerSession> rebuildCollection(String name) {
        return rebuildCollection(RebuildOptions.defaults(name));
    }

    public MaintenanceResult<CompactionResult, ServerSession> compactCollection(CompactCollectionRequest request) {
        backupLock.lock();
        try {
            ensureOpen();
            if (request == null) throw new IllegalArgumentException("request is required");
            MaintenanceResult<CompactionResult, ServerSession> result = compactCollectionAction.apply(request);
            closed.set(true);
            return result;
        } catch (RuntimeException e) {
            closed.set(true);
            throw e;
        } finally {
            backupLock.unlock();
        }
    }

    public MaintenanceResult<CompactionResult, ServerSession> compactCollection(String name) {
        return compactCollection(new CompactCollectionRequest.Builder(name).build());
    }

    public MaintenanceResult<CompactionResult, ServerSession> compactAll(CompactAllRequest request) {
        backupLock.lock();
        try {
            ensureOpen();
            if (request == null) throw new IllegalArgumentException("request is required");
            MaintenanceResult<CompactionResult, ServerSession> result = compactAllAction.apply(request);
            closed.set(true);
            return result;
        } catch (RuntimeException e) {
            closed.set(true);
            throw e;
        } finally {
            backupLock.unlock();
        }
    }

    public MaintenanceResult<WALPruneResult, ServerSession> pruneCollectionWAL(WALPruneOptions options) {
        backupLock.lock();
        try {
            ensureOpen();
            if (options == null) throw new IllegalArgumentException("options is required");
            MaintenanceResult<WALPruneResult, ServerSession> result = pruneWalCollectionAction.apply(options);
            closed.set(true);
            return result;
        } catch (RuntimeException e) {
            closed.set(true);
            throw e;
        } finally {
            backupLock.unlock();
        }
    }

    public MaintenanceResult<WALPruneResult, ServerSession> pruneCollectionWAL(String name) {
        return pruneCollectionWAL(WALPruneOptions.defaults(name));
    }

    public MaintenanceResult<WALPruneResult, ServerSession> pruneAllWAL(WALPruneOptions options) {
        backupLock.lock();
        try {
            ensureOpen();
            if (options == null) throw new IllegalArgumentException("options is required");
            MaintenanceResult<WALPruneResult, ServerSession> result = pruneWalAllAction.apply(options);
            closed.set(true);
            return result;
        } catch (RuntimeException e) {
            closed.set(true);
            throw e;
        } finally {
            backupLock.unlock();
        }
    }

    public BackupResult<ServerSession> backup(BackupOptions options) {
        backupLock.lock();
        try {
            ensureOpen();
            if (options == null) throw new IllegalArgumentException("options is required");
            try {
                BackupResult<ServerSession> result = backupAction.apply(options);
                closed.set(true);
                return result;
            } catch (BackupExecutor.PreValidationFailure e) {
                throw (RuntimeException) e.getCause();
            } catch (RuntimeException e) {
                closed.set(true);
                throw e;
            }
        } finally {
            backupLock.unlock();
        }
    }
}
