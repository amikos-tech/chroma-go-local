package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.util.concurrent.atomic.AtomicInteger;
import java.util.function.LongConsumer;
import org.junit.jupiter.api.Test;

class EmbeddedSessionTest {

    private static EmbeddedSession create(long handle, LongConsumer closeAction) {
        return new EmbeddedSession(handle, closeAction,
                (h, json) -> null, (h, json) -> null, (h, json) -> null,
                (h, json) -> null, (h, json) -> null);
    }

    @Test
    void constructorRejectsZeroHandle() {
        assertThrows(IllegalArgumentException.class, () -> create(0L, ignored -> {}));
    }

    @Test
    void constructorRejectsNullCloseAction() {
        assertThrows(IllegalArgumentException.class, () -> create(42L, null));
    }

    @Test
    void closeInvokesActionOnce() {
        AtomicInteger closeCalls = new AtomicInteger();
        EmbeddedSession session = create(42L, ignored -> closeCalls.incrementAndGet());

        session.close();
        session.close();

        assertEquals(1, closeCalls.get());
    }

    @Test
    void close_staysClosedAfterFailure() {
        AtomicInteger closeCalls = new AtomicInteger();
        EmbeddedSession session = create(42L, ignored -> {
            closeCalls.incrementAndGet();
            throw new RuntimeException("close failed");
        });

        assertThrows(RuntimeException.class, session::close);
        assertThrows(IllegalStateException.class, session::handle, "handle poisoned after failed close");
        session.close();
        assertEquals(1, closeCalls.get(), "closeAction must not be retried — native handle is in unknown state");
    }

    @Test
    void handle_returnsValue_whenOpen() {
        EmbeddedSession session = create(42L, ignored -> {});
        assertEquals(42L, session.handle());
    }

    @Test
    void handleThrowsAfterClose() {
        EmbeddedSession session = create(42L, ignored -> {});

        session.close();

        assertThrows(IllegalStateException.class, session::handle);
    }
}
