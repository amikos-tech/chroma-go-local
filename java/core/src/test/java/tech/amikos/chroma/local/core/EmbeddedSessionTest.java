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
    void close_remainsClosedAfterFailure() {
        AtomicInteger closeCalls = new AtomicInteger();
        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {
            closeCalls.incrementAndGet();
            throw new RuntimeException("close failed");
        });

        assertThrows(RuntimeException.class, session::close);
        assertThrows(IllegalStateException.class, session::handle);
        session.close();
        assertEquals(1, closeCalls.get(), "closeAction must not be called again on second close");
    }

    @Test
    void handleThrowsAfterClose() {
        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {});

        session.close();

        assertThrows(IllegalStateException.class, session::handle);
    }
}
