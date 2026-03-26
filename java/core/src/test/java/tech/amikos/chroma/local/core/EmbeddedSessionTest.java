package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.util.concurrent.atomic.AtomicInteger;
import org.junit.jupiter.api.Test;

class EmbeddedSessionTest {
    @Test
    void constructorRejectsZeroHandle() {
        assertThrows(IllegalArgumentException.class, () -> new EmbeddedSession(0L, ignored -> {}));
    }

    @Test
    void constructorRejectsNullCloseAction() {
        assertThrows(IllegalArgumentException.class, () -> new EmbeddedSession(42L, null));
    }

    @Test
    void closeInvokesActionOnce() {
        AtomicInteger closeCalls = new AtomicInteger();
        EmbeddedSession session = new EmbeddedSession(42L, ignored -> closeCalls.incrementAndGet());

        session.close();
        session.close();

        assertEquals(1, closeCalls.get());
    }

    @Test
    void close_retriableAfterFailure() {
        AtomicInteger closeCalls = new AtomicInteger();
        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {
            if (closeCalls.incrementAndGet() == 1) {
                throw new RuntimeException("close failed");
            }
        });

        assertThrows(RuntimeException.class, session::close);
        assertEquals(42L, session.handle(), "handle accessible after failed close");
        session.close();
        assertEquals(2, closeCalls.get(), "closeAction retried on second close");
        assertThrows(IllegalStateException.class, session::handle);
    }

    @Test
    void handle_returnsValue_whenOpen() {
        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {});
        assertEquals(42L, session.handle());
    }

    @Test
    void handleThrowsAfterClose() {
        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {});

        session.close();

        assertThrows(IllegalStateException.class, session::handle);
    }
}
