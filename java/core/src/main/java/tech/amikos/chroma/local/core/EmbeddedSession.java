package tech.amikos.chroma.local.core;

import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.BiFunction;
import java.util.function.LongConsumer;

public final class EmbeddedSession implements AutoCloseable {
    private final long handle;
    private final LongConsumer closeAction;
    private final AtomicBoolean closed;
    private final BiFunction<Long, String, RebuildCollectionResult> rebuildAction;
    private final BiFunction<Long, String, CompactionResult> compactCollectionAction;
    private final BiFunction<Long, String, CompactionResult> compactAllAction;
    private final BiFunction<Long, String, WALPruneResult> pruneWalCollectionAction;
    private final BiFunction<Long, String, WALPruneResult> pruneWalAllAction;

    public EmbeddedSession(long handle, LongConsumer closeAction,
            BiFunction<Long, String, RebuildCollectionResult> rebuildAction,
            BiFunction<Long, String, CompactionResult> compactCollectionAction,
            BiFunction<Long, String, CompactionResult> compactAllAction,
            BiFunction<Long, String, WALPruneResult> pruneWalCollectionAction,
            BiFunction<Long, String, WALPruneResult> pruneWalAllAction) {
        if (handle == 0L) {
            throw new IllegalArgumentException("embedded handle must be non-zero");
        }
        if (closeAction == null) {
            throw new IllegalArgumentException("closeAction must be set");
        }
        if (rebuildAction == null) {
            throw new IllegalArgumentException("rebuildAction must be set");
        }
        if (compactCollectionAction == null) {
            throw new IllegalArgumentException("compactCollectionAction must be set");
        }
        if (compactAllAction == null) {
            throw new IllegalArgumentException("compactAllAction must be set");
        }
        if (pruneWalCollectionAction == null) {
            throw new IllegalArgumentException("pruneWalCollectionAction must be set");
        }
        if (pruneWalAllAction == null) {
            throw new IllegalArgumentException("pruneWalAllAction must be set");
        }
        this.handle = handle;
        this.closeAction = closeAction;
        this.closed = new AtomicBoolean(false);
        this.rebuildAction = rebuildAction;
        this.compactCollectionAction = compactCollectionAction;
        this.compactAllAction = compactAllAction;
        this.pruneWalCollectionAction = pruneWalCollectionAction;
        this.pruneWalAllAction = pruneWalAllAction;
    }

    private void ensureOpen() {
        if (closed.get()) throw new IllegalStateException("session is closed");
    }

    public long handle() {
        ensureOpen();
        return handle;
    }

    public RebuildCollectionResult rebuildCollection(String name, RebuildOptions options) {
        ensureOpen();
        if (name == null || name.isBlank()) {
            throw new IllegalArgumentException("name is required");
        }
        if (options == null) {
            throw new IllegalArgumentException("options is required");
        }
        return rebuildAction.apply(handle, options.toJson());
    }

    public RebuildCollectionResult rebuildCollection(String name) {
        return rebuildCollection(name, RebuildOptions.defaults(name));
    }

    public CompactionResult compactCollection(CompactCollectionRequest request) {
        ensureOpen();
        if (request == null) {
            throw new IllegalArgumentException("request is required");
        }
        return compactCollectionAction.apply(handle, request.toJson());
    }

    public CompactionResult compactAll(CompactAllRequest request) {
        ensureOpen();
        if (request == null) {
            throw new IllegalArgumentException("request is required");
        }
        return compactAllAction.apply(handle, request.toJson());
    }

    public WALPruneResult pruneCollectionWAL(String name, WALPruneOptions options) {
        ensureOpen();
        if (name == null || name.isBlank()) {
            throw new IllegalArgumentException("name is required");
        }
        if (options == null) {
            throw new IllegalArgumentException("options is required");
        }
        return pruneWalCollectionAction.apply(handle, options.toJson());
    }

    public WALPruneResult pruneAllWAL(WALPruneOptions options) {
        ensureOpen();
        if (options == null) {
            throw new IllegalArgumentException("options is required");
        }
        return pruneWalAllAction.apply(handle, options.toJson());
    }

    @Override
    public void close() {
        if (closed.compareAndSet(false, true)) {
            closeAction.accept(handle);
        }
    }
}
