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
    void closeAllowsRetryAfterFailure() {
        AtomicInteger closeCalls = new AtomicInteger();
        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {
            int call = closeCalls.incrementAndGet();
            if (call == 1) {
                throw new RuntimeException("first close failed");
            }
        });

        assertThrows(RuntimeException.class, session::close);
        session.close();

        assertEquals(2, closeCalls.get());
    }

    @Test
    void handleThrowsAfterClose() {
        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {});

        session.close();

        assertThrows(IllegalStateException.class, session::handle);
    }
}
