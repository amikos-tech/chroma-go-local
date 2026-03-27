package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.atomic.AtomicReference;
import java.util.function.BiFunction;
import java.util.function.Function;
import java.util.function.LongConsumer;
import org.junit.jupiter.api.Test;

class EmbeddedSessionTest {

    private static final BiFunction<Long, String, RebuildCollectionResult> STUB_REBUILD =
            (h, json) -> { throw new UnsupportedOperationException("stub"); };
    private static final BiFunction<Long, String, CompactionResult> STUB_COMPACT_COLLECTION =
            (h, json) -> { throw new UnsupportedOperationException("stub"); };
    private static final BiFunction<Long, String, CompactionResult> STUB_COMPACT_ALL =
            (h, json) -> { throw new UnsupportedOperationException("stub"); };
    private static final BiFunction<Long, String, WALPruneResult> STUB_PRUNE_WAL_COLLECTION =
            (h, json) -> { throw new UnsupportedOperationException("stub"); };
    private static final BiFunction<Long, String, WALPruneResult> STUB_PRUNE_WAL_ALL =
            (h, json) -> { throw new UnsupportedOperationException("stub"); };
    private static final Function<BackupOptions, BackupResult<EmbeddedSession>> STUB_BACKUP =
            opts -> { throw new UnsupportedOperationException("stub"); };

    private static EmbeddedSession create(long handle, LongConsumer closeAction) {
        return new EmbeddedSession(handle, closeAction,
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL, STUB_BACKUP);
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
    void constructorRejectsNullRebuildAction() {
        assertThrows(IllegalArgumentException.class, () -> new EmbeddedSession(42L, ignored -> {},
                null, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL, STUB_BACKUP));
    }

    @Test
    void constructorRejectsNullCompactCollectionAction() {
        assertThrows(IllegalArgumentException.class, () -> new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, null, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL, STUB_BACKUP));
    }

    @Test
    void constructorRejectsNullCompactAllAction() {
        assertThrows(IllegalArgumentException.class, () -> new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, STUB_COMPACT_COLLECTION, null,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL, STUB_BACKUP));
    }

    @Test
    void constructorRejectsNullPruneWalCollectionAction() {
        assertThrows(IllegalArgumentException.class, () -> new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                null, STUB_PRUNE_WAL_ALL, STUB_BACKUP));
    }

    @Test
    void constructorRejectsNullPruneWalAllAction() {
        assertThrows(IllegalArgumentException.class, () -> new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, null, STUB_BACKUP));
    }

    @Test
    void constructorRejectsNullBackupAction() {
        assertThrows(IllegalArgumentException.class, () -> new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL, null));
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
        assertEquals(1, closeCalls.get(), "closeAction must not be retried");
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

    @Test
    void rebuildCollectionRejectsNullOptions() {
        EmbeddedSession session = create(42L, ignored -> {});
        assertThrows(IllegalArgumentException.class, () -> session.rebuildCollection((RebuildOptions) null));
    }

    @Test
    void compactCollectionRejectsNullRequest() {
        EmbeddedSession session = create(42L, ignored -> {});
        assertThrows(IllegalArgumentException.class, () -> session.compactCollection((CompactCollectionRequest) null));
    }

    @Test
    void compactAllRejectsNullRequest() {
        EmbeddedSession session = create(42L, ignored -> {});
        assertThrows(IllegalArgumentException.class, () -> session.compactAll(null));
    }

    @Test
    void pruneCollectionWalRejectsNullOptions() {
        EmbeddedSession session = create(42L, ignored -> {});
        assertThrows(IllegalArgumentException.class, () -> session.pruneCollectionWAL((WALPruneOptions) null));
    }

    @Test
    void pruneAllWalRejectsNullOptions() {
        EmbeddedSession session = create(42L, ignored -> {});
        assertThrows(IllegalArgumentException.class, () -> session.pruneAllWAL(null));
    }

    @Test
    void backupRejectsNullOptions() {
        EmbeddedSession session = create(42L, ignored -> {});
        assertThrows(IllegalArgumentException.class, () -> session.backup(null));
    }

    @Test
    void backupThrowsAfterClose() {
        EmbeddedSession session = create(42L, ignored -> {});
        session.close();
        BackupOptions opts = new BackupOptions.Builder("/tmp/dest").build();
        assertThrows(IllegalStateException.class, () -> session.backup(opts));
    }

    @Test
    void rebuildCollectionConvenienceOverloadDelegates() {
        AtomicLong capturedHandle = new AtomicLong();
        AtomicReference<String> capturedJson = new AtomicReference<>();
        RebuildCollectionResult fakeResult = new RebuildCollectionResult();

        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {},
                (h, json) -> { capturedHandle.set(h); capturedJson.set(json); return fakeResult; },
                STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL, STUB_BACKUP);

        RebuildCollectionResult result = session.rebuildCollection("myCollection");

        assertEquals(fakeResult, result);
        assertEquals(42L, capturedHandle.get());
        assertNotNull(capturedJson.get());
        assertTrue(capturedJson.get().contains("myCollection"));
    }

    @Test
    void compactCollectionConvenienceOverloadDelegates() {
        AtomicLong capturedHandle = new AtomicLong();
        AtomicReference<String> capturedJson = new AtomicReference<>();
        CompactionResult fakeResult = new CompactionResult();

        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD,
                (h, json) -> { capturedHandle.set(h); capturedJson.set(json); return fakeResult; },
                STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL, STUB_BACKUP);

        CompactionResult result = session.compactCollection("myCollection");

        assertEquals(fakeResult, result);
        assertEquals(42L, capturedHandle.get());
        assertNotNull(capturedJson.get());
        assertTrue(capturedJson.get().contains("myCollection"));
    }

    @Test
    void backupDelegatesAndInvalidatesSession() {
        BackupManifest manifest = new BackupManifest("v1", "embedded", "now", "java",
                java.util.List.of("/src"), "/dst", "/dst/persist", "/dst/manifest.json",
                false, 0, 0, null);
        BackupResult<EmbeddedSession> fakeResult = new BackupResult<>(manifest, null);
        AtomicReference<BackupOptions> capturedOpts = new AtomicReference<>();

        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL,
                opts -> { capturedOpts.set(opts); return fakeResult; });

        BackupOptions options = new BackupOptions.Builder("/tmp/backup").build();
        BackupResult<EmbeddedSession> result = session.backup(options);

        assertEquals(fakeResult, result);
        assertEquals(options, capturedOpts.get());
        assertThrows(IllegalStateException.class, session::handle,
                "session must be invalidated after backup");
    }

    @Test
    void backupSetsClosedAfterActionNotBefore() {
        AtomicBoolean wasOpenDuringAction = new AtomicBoolean(false);
        BackupManifest manifest = new BackupManifest("v1", "embedded", "now", "java",
                java.util.List.of("/src"), "/dst", "/dst/persist", "/dst/manifest.json",
                false, 0, 0, null);

        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL,
                opts -> {
                    wasOpenDuringAction.set(true);
                    return new BackupResult<>(manifest, null);
                });

        session.backup(new BackupOptions.Builder("/tmp/backup").build());
        assertTrue(wasOpenDuringAction.get(), "backupAction must execute before session is marked closed");
        assertThrows(IllegalStateException.class, session::handle,
                "session must be closed after backup completes");
    }

    @Test
    void closeAfterBackupIsNoOp() {
        BackupManifest manifest = new BackupManifest("v1", "embedded", "now", "java",
                java.util.List.of("/src"), "/dst", "/dst/persist", "/dst/manifest.json",
                false, 0, 0, null);
        AtomicInteger closeCalls = new AtomicInteger();

        EmbeddedSession session = new EmbeddedSession(42L, ignored -> closeCalls.incrementAndGet(),
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL,
                opts -> new BackupResult<>(manifest, null));

        session.backup(new BackupOptions.Builder("/tmp/backup").build());
        session.close();
        assertEquals(0, closeCalls.get(), "closeAction must not be called after backup invalidated the session");
    }

    @Test
    void backupFailureStillInvalidatesSession() {
        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL,
                opts -> { throw new RuntimeException("backup failed"); });

        BackupOptions options = new BackupOptions.Builder("/tmp/backup").build();
        assertThrows(RuntimeException.class, () -> session.backup(options));

        assertThrows(IllegalStateException.class, session::handle,
                "session must be invalidated even when backup fails");
        session.close();
    }

    @Test
    void backupPreValidationFailureLeavesSessionOpen() {
        AtomicInteger closeCalls = new AtomicInteger();
        EmbeddedSession session = new EmbeddedSession(42L, ignored -> closeCalls.incrementAndGet(),
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL,
                opts -> { throw new BackupExecutor.PreValidationFailure(
                        new IllegalArgumentException("dest inside source")); });

        BackupOptions options = new BackupOptions.Builder("/tmp/backup").build();
        IllegalArgumentException ex = assertThrows(IllegalArgumentException.class,
                () -> session.backup(options));
        assertEquals("dest inside source", ex.getMessage());

        assertEquals(42L, session.handle(), "session must remain open after pre-validation failure");

        session.close();
        assertEquals(1, closeCalls.get(), "closeAction must run on explicit close");
    }

    @Test
    void pruneCollectionWalConvenienceOverloadDelegates() {
        AtomicLong capturedHandle = new AtomicLong();
        AtomicReference<String> capturedJson = new AtomicReference<>();
        WALPruneResult fakeResult = new WALPruneResult();

        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                (h, json) -> { capturedHandle.set(h); capturedJson.set(json); return fakeResult; },
                STUB_PRUNE_WAL_ALL, STUB_BACKUP);

        WALPruneResult result = session.pruneCollectionWAL("myCollection");

        assertEquals(fakeResult, result);
        assertEquals(42L, capturedHandle.get());
        assertNotNull(capturedJson.get());
        assertTrue(capturedJson.get().contains("myCollection"));
    }
}
