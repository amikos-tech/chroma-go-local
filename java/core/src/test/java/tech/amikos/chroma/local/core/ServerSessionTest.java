package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.atomic.AtomicReference;
import java.util.function.Function;
import org.junit.jupiter.api.Test;

class ServerSessionTest {

    private static final Function<BackupOptions, BackupResult<ServerSession>> STUB_BACKUP =
            opts -> { throw new UnsupportedOperationException("stub"); };

    private ServerSession createSession(long handle) {
        return new ServerSession(
                handle,
                h -> {},
                h -> {},
                h -> 8000,
                h -> "localhost",
                h -> "/data",
                STUB_BACKUP
        );
    }

    @Test
    void constructorRejectsZeroHandle() {
        assertThrows(IllegalArgumentException.class,
                () -> new ServerSession(0L, h -> {}, h -> {}, h -> 0, h -> "", h -> "", STUB_BACKUP));
    }

    @Test
    void constructorRejectsNullStopAction() {
        assertThrows(IllegalArgumentException.class,
                () -> new ServerSession(1L, null, h -> {}, h -> 0, h -> "", h -> "", STUB_BACKUP));
    }

    @Test
    void constructorRejectsNullFreeAction() {
        assertThrows(IllegalArgumentException.class,
                () -> new ServerSession(1L, h -> {}, null, h -> 0, h -> "", h -> "", STUB_BACKUP));
    }

    @Test
    void constructorRejectsNullPortAccessor() {
        assertThrows(IllegalArgumentException.class,
                () -> new ServerSession(1L, h -> {}, h -> {}, null, h -> "", h -> "", STUB_BACKUP));
    }

    @Test
    void constructorRejectsNullAddressAccessor() {
        assertThrows(IllegalArgumentException.class,
                () -> new ServerSession(1L, h -> {}, h -> {}, h -> 0, null, h -> "", STUB_BACKUP));
    }

    @Test
    void constructorRejectsNullPersistPathAccessor() {
        assertThrows(IllegalArgumentException.class,
                () -> new ServerSession(1L, h -> {}, h -> {}, h -> 0, h -> "", null, STUB_BACKUP));
    }

    @Test
    void constructorRejectsNullBackupAction() {
        assertThrows(IllegalArgumentException.class,
                () -> new ServerSession(1L, h -> {}, h -> {}, h -> 0, h -> "", h -> "", null));
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
                h -> "host", h -> "/path",
                STUB_BACKUP
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
                h -> "/path",
                STUB_BACKUP
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
                h -> { receivedHandle.set(h); return "/my/data"; },
                STUB_BACKUP
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
                h -> "/data",
                STUB_BACKUP
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
                h -> 8000, h -> "host", h -> "/path",
                STUB_BACKUP
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
                h -> 8000, h -> "host", h -> "/path",
                STUB_BACKUP
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
                h -> 8000, h -> "host", h -> "/path",
                STUB_BACKUP
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
                h -> 8000, h -> "host", h -> "/path",
                STUB_BACKUP
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
                h -> 8000, h -> "host", h -> "/path",
                STUB_BACKUP
        );
        RuntimeException ex = assertThrows(RuntimeException.class, session::close);
        assertEquals("stop failed", ex.getMessage(), "stop exception propagates as primary");
        assertEquals(1, ex.getSuppressed().length, "free exception is suppressed");
        assertEquals("free failed", ex.getSuppressed()[0].getMessage());
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
                () -> session.rebuildCollection(RebuildOptions.defaults("coll")));
    }

    @Test
    void compactCollection_throwsUnsupportedOperationException() {
        ServerSession session = createSession(1L);
        assertThrows(UnsupportedOperationException.class,
                () -> session.compactCollection((CompactCollectionRequest) null));
    }

    @Test
    void backupRejectsNullOptions() {
        ServerSession session = createSession(1L);
        assertThrows(IllegalArgumentException.class, () -> session.backup(null));
    }

    @Test
    void backupThrowsAfterClose() {
        ServerSession session = createSession(1L);
        session.close();
        BackupOptions opts = new BackupOptions.Builder("/tmp/dest").build();
        assertThrows(IllegalStateException.class, () -> session.backup(opts));
    }

    @Test
    void backupDelegatesAndInvalidatesSession() {
        BackupManifest manifest = new BackupManifest("v1", "server", "now", "java",
                java.util.List.of("/src"), "/dst", "/dst/persist", "/dst/manifest.json",
                false, 0, 0, null);
        BackupResult<ServerSession> fakeResult = new BackupResult<>(manifest, null);
        AtomicReference<BackupOptions> capturedOpts = new AtomicReference<>();

        ServerSession session = new ServerSession(
                42L,
                h -> {}, h -> {},
                h -> 8000, h -> "host", h -> "/path",
                opts -> { capturedOpts.set(opts); return fakeResult; }
        );

        BackupOptions options = new BackupOptions.Builder("/tmp/backup").build();
        BackupResult<ServerSession> result = session.backup(options);

        assertEquals(fakeResult, result);
        assertEquals(options, capturedOpts.get());
        assertThrows(IllegalStateException.class, session::port,
                "session must be invalidated after backup");
    }

    @Test
    void closeAfterBackupIsNoOp() {
        BackupManifest manifest = new BackupManifest("v1", "server", "now", "java",
                java.util.List.of("/src"), "/dst", "/dst/persist", "/dst/manifest.json",
                false, 0, 0, null);
        AtomicInteger stopCalls = new AtomicInteger();
        AtomicInteger freeCalls = new AtomicInteger();

        ServerSession session = new ServerSession(
                42L,
                h -> stopCalls.incrementAndGet(),
                h -> freeCalls.incrementAndGet(),
                h -> 8000, h -> "host", h -> "/path",
                opts -> new BackupResult<>(manifest, null)
        );

        session.backup(new BackupOptions.Builder("/tmp/backup").build());
        session.close();
        assertEquals(0, stopCalls.get(), "stopAction must not be called after backup invalidated the session");
        assertEquals(0, freeCalls.get(), "freeAction must not be called after backup invalidated the session");
    }

    @Test
    void backupFailureStillInvalidatesSession() {
        ServerSession session = new ServerSession(
                42L,
                h -> {}, h -> {},
                h -> 8000, h -> "host", h -> "/path",
                opts -> { throw new RuntimeException("backup failed"); }
        );

        BackupOptions options = new BackupOptions.Builder("/tmp/backup").build();
        assertThrows(RuntimeException.class, () -> session.backup(options));

        assertThrows(IllegalStateException.class, session::port,
                "session must be invalidated even when backup fails");
        session.close();
    }

    @Test
    void backupPreValidationFailureLeavesSessionOpen() {
        AtomicInteger stopCalls = new AtomicInteger();
        AtomicInteger freeCalls = new AtomicInteger();
        ServerSession session = new ServerSession(
                42L,
                h -> stopCalls.incrementAndGet(),
                h -> freeCalls.incrementAndGet(),
                h -> 8000, h -> "host", h -> "/path",
                opts -> { throw new BackupExecutor.PreValidationFailure(
                        new IllegalArgumentException("dest inside source")); }
        );

        BackupOptions options = new BackupOptions.Builder("/tmp/backup").build();
        IllegalArgumentException ex = assertThrows(IllegalArgumentException.class,
                () -> session.backup(options));
        assertEquals("dest inside source", ex.getMessage());

        assertEquals(8000, session.port(), "session must remain open after pre-validation failure");

        session.close();
        assertEquals(1, stopCalls.get(), "stopAction must run on explicit close");
        assertEquals(1, freeCalls.get(), "freeAction must run on explicit close");
    }

    @Test
    void maintenanceMethods_throwIllegalStateException_afterClose() {
        ServerSession session = createSession(1L);
        session.close();
        assertThrows(IllegalStateException.class,
                () -> session.rebuildCollection(RebuildOptions.defaults("coll")));
        assertThrows(IllegalStateException.class,
                () -> session.compactCollection((CompactCollectionRequest) null));
        assertThrows(IllegalStateException.class,
                () -> session.compactAll(null));
        assertThrows(IllegalStateException.class,
                () -> session.pruneCollectionWAL(WALPruneOptions.defaults("coll")));
        assertThrows(IllegalStateException.class,
                () -> session.pruneAllWAL(null));
        BackupOptions opts = new BackupOptions.Builder("/tmp/dest").build();
        assertThrows(IllegalStateException.class, () -> session.backup(opts));
    }
}
