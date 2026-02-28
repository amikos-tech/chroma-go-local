package tech.amikos.chroma.local.core;

import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.LongConsumer;

public final class EmbeddedSession implements AutoCloseable {
    private final long handle;
    private final LongConsumer closeAction;
    private final AtomicBoolean closed;

    public EmbeddedSession(long handle, LongConsumer closeAction) {
        if (handle == 0L) {
            throw new IllegalArgumentException("embedded handle must be non-zero");
        }
        this.handle = handle;
        this.closeAction = closeAction;
        this.closed = new AtomicBoolean(false);
    }

    public long handle() {
        return handle;
    }

    @Override
    public void close() {
        if (closed.compareAndSet(false, true)) {
            closeAction.accept(handle);
        }
    }
}
