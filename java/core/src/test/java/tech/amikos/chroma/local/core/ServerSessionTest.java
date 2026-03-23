package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;
import org.junit.jupiter.api.Test;

class ServerSessionTest {

    private ServerSession createSession(long handle) {
        return new ServerSession(
                handle,
                h -> {},
                h -> {},
                h -> 8000,
                h -> "localhost",
                h -> "/data"
        );
    }

    @Test
    void constructorRejectsZeroHandle() {
        assertThrows(IllegalArgumentException.class,
                () -> new ServerSession(0L, h -> {}, h -> {}, h -> 0, h -> "", h -> ""));
    }

    @Test
    void constructorRejectsNullStopAction() {
        assertThrows(IllegalArgumentException.class,
                () -> new ServerSession(1L, null, h -> {}, h -> 0, h -> "", h -> ""));
    }

    @Test
    void constructorRejectsNullFreeAction() {
        assertThrows(IllegalArgumentException.class,
                () -> new ServerSession(1L, h -> {}, null, h -> 0, h -> "", h -> ""));
    }

    @Test
    void constructorRejectsNullPortAccessor() {
        assertThrows(IllegalArgumentException.class,
                () -> new ServerSession(1L, h -> {}, h -> {}, null, h -> "", h -> ""));
    }

    @Test
    void constructorRejectsNullAddressAccessor() {
        assertThrows(IllegalArgumentException.class,
                () -> new ServerSession(1L, h -> {}, h -> {}, h -> 0, null, h -> ""));
    }

    @Test
    void constructorRejectsNullPersistPathAccessor() {
        assertThrows(IllegalArgumentException.class,
                () -> new ServerSession(1L, h -> {}, h -> {}, h -> 0, h -> "", null));
    }

    @Test
    void handle_returnsValue_whenOpen() {
        ServerSession session = createSession(42L);
        assertEquals(42L, session.handle());
    }

    @Test
    void handle_throwsIllegalStateException_whenClosed() {
        ServerSession session = createSession(42L);
        session.close();
        assertThrows(IllegalStateException.class, session::handle);
    }

    @Test
    void port_callsPortAccessor() {
        AtomicLong receivedHandle = new AtomicLong();
        ServerSession session = new ServerSession(
                99L,
                h -> {}, h -> {},
                h -> { receivedHandle.set(h); return 9090; },
                h -> "host", h -> "/path"
        );
        assertEquals(9090, session.port());
        assertEquals(99L, receivedHandle.get());
    }

    @Test
    void address_callsAddressAccessor() {
        AtomicLong receivedHandle = new AtomicLong();
        ServerSession session = new ServerSession(
                88L,
                h -> {}, h -> {},
                h -> 8000,
                h -> { receivedHandle.set(h); return "0.0.0.0"; },
                h -> "/path"
        );
        assertEquals("0.0.0.0", session.address());
        assertEquals(88L, receivedHandle.get());
    }

    @Test
    void persistPath_callsPersistPathAccessor() {
        AtomicLong receivedHandle = new AtomicLong();
        ServerSession session = new ServerSession(
                77L,
                h -> {}, h -> {},
                h -> 8000, h -> "host",
                h -> { receivedHandle.set(h); return "/my/data"; }
        );
        assertEquals("/my/data", session.persistPath());
        assertEquals(77L, receivedHandle.get());
    }

    @Test
    void url_returnsHttpUrl() {
        ServerSession session = new ServerSession(
                1L,
                h -> {}, h -> {},
                h -> 8080,
                h -> "127.0.0.1",
                h -> "/data"
        );
        assertEquals("http://127.0.0.1:8080", session.url());
    }

    @Test
    void close_callsStopThenFree() {
        AtomicInteger stopCalls = new AtomicInteger();
        AtomicInteger freeCalls = new AtomicInteger();
        ServerSession session = new ServerSession(
                1L,
                h -> stopCalls.incrementAndGet(),
                h -> freeCalls.incrementAndGet(),
                h -> 8000, h -> "host", h -> "/path"
        );
        session.close();
        assertEquals(1, stopCalls.get());
        assertEquals(1, freeCalls.get());
    }

    @Test
    void close_isIdempotent() {
        AtomicInteger stopCalls = new AtomicInteger();
        AtomicInteger freeCalls = new AtomicInteger();
        ServerSession session = new ServerSession(
                1L,
                h -> stopCalls.incrementAndGet(),
                h -> freeCalls.incrementAndGet(),
                h -> 8000, h -> "host", h -> "/path"
        );
        session.close();
        session.close();
        session.close();
        assertEquals(1, stopCalls.get());
        assertEquals(1, freeCalls.get());
    }

    @Test
    void close_freesEvenIfStopFails() {
        AtomicInteger freeCalls = new AtomicInteger();
        ServerSession session = new ServerSession(
                1L,
                h -> { throw new RuntimeException("stop failed"); },
                h -> freeCalls.incrementAndGet(),
                h -> 8000, h -> "host", h -> "/path"
        );
        assertThrows(RuntimeException.class, session::close);
        assertEquals(1, freeCalls.get(), "freeAction must run even if stopAction throws");
    }

    @Test
    void close_remainsClosedAfterStopFailure() {
        AtomicInteger stopCalls = new AtomicInteger();
        ServerSession session = new ServerSession(
                1L,
                h -> { stopCalls.incrementAndGet(); throw new RuntimeException("stop failed"); },
                h -> {},
                h -> 8000, h -> "host", h -> "/path"
        );
        assertThrows(RuntimeException.class, session::close);
        assertThrows(IllegalStateException.class, session::port);
        assertThrows(IllegalStateException.class, session::address);
        session.close();
        assertEquals(1, stopCalls.get(), "stopAction must not be called again on second close");
    }

    @Test
    void close_freeRunsEvenWhenBothFail() {
        ServerSession session = new ServerSession(
                1L,
                h -> { throw new RuntimeException("stop failed"); },
                h -> { throw new RuntimeException("free failed"); },
                h -> 8000, h -> "host", h -> "/path"
        );
        RuntimeException ex = assertThrows(RuntimeException.class, session::close);
        assertEquals("free failed", ex.getMessage(), "finally-block exception propagates");
    }

    @Test
    void port_throwsIllegalStateException_afterClose() {
        ServerSession session = createSession(1L);
        session.close();
        assertThrows(IllegalStateException.class, session::port);
    }

    @Test
    void address_throwsIllegalStateException_afterClose() {
        ServerSession session = createSession(1L);
        session.close();
        assertThrows(IllegalStateException.class, session::address);
    }

    @Test
    void url_throwsIllegalStateException_afterClose() {
        ServerSession session = createSession(1L);
        session.close();
        assertThrows(IllegalStateException.class, session::url);
    }

    @Test
    void rebuildCollection_throwsUnsupportedOperationException() {
        ServerSession session = createSession(1L);
        assertThrows(UnsupportedOperationException.class,
                () -> session.rebuildCollection("coll", RebuildOptions.defaults("coll")));
    }

    @Test
    void compactCollection_throwsUnsupportedOperationException() {
        ServerSession session = createSession(1L);
        assertThrows(UnsupportedOperationException.class,
                () -> session.compactCollection(null));
    }

    @Test
    void backup_throwsUnsupportedOperationException() {
        ServerSession session = createSession(1L);
        assertThrows(UnsupportedOperationException.class,
                () -> session.backup(null));
    }

    @Test
    void maintenanceMethods_throwIllegalStateException_afterClose() {
        ServerSession session = createSession(1L);
        session.close();
        assertThrows(IllegalStateException.class,
                () -> session.rebuildCollection("coll", RebuildOptions.defaults("coll")));
        assertThrows(IllegalStateException.class,
                () -> session.compactCollection(null));
        assertThrows(IllegalStateException.class,
                () -> session.compactAll(null));
        assertThrows(IllegalStateException.class,
                () -> session.pruneCollectionWAL("coll", null));
        assertThrows(IllegalStateException.class,
                () -> session.pruneAllWAL(null));
        assertThrows(IllegalStateException.class,
                () -> session.backup(null));
    }
}
