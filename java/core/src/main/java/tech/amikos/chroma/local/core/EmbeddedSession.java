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
        if (closeAction == null) {
            throw new IllegalArgumentException("closeAction must be set");
        }
        this.handle = handle;
        this.closeAction = closeAction;
        this.closed = new AtomicBoolean(false);
    }

    public long handle() {
        if (closed.get()) {
            throw new IllegalStateException("session is closed");
        }
        return handle;
    }

    @Override
    public void close() {
        if (closed.compareAndSet(false, true)) {
            try {
                closeAction.accept(handle);
            } catch (RuntimeException | Error e) {
                closed.set(false);
                throw e;
            }
        }
    }
}
