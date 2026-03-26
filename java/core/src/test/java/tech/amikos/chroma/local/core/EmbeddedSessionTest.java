package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.atomic.AtomicReference;
import java.util.function.BiFunction;
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

    private static EmbeddedSession create(long handle, LongConsumer closeAction) {
        return new EmbeddedSession(handle, closeAction,
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL);
    }

    // --- Constructor null-rejection tests ---

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
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL));
    }

    @Test
    void constructorRejectsNullCompactCollectionAction() {
        assertThrows(IllegalArgumentException.class, () -> new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, null, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL));
    }

    @Test
    void constructorRejectsNullCompactAllAction() {
        assertThrows(IllegalArgumentException.class, () -> new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, STUB_COMPACT_COLLECTION, null,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL));
    }

    @Test
    void constructorRejectsNullPruneWalCollectionAction() {
        assertThrows(IllegalArgumentException.class, () -> new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                null, STUB_PRUNE_WAL_ALL));
    }

    @Test
    void constructorRejectsNullPruneWalAllAction() {
        assertThrows(IllegalArgumentException.class, () -> new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, null));
    }

    // --- Close lifecycle tests ---

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

    // --- Input validation tests ---

    @Test
    void rebuildCollectionRejectsNullOptions() {
        EmbeddedSession session = create(42L, ignored -> {});
        assertThrows(IllegalArgumentException.class, () -> session.rebuildCollection((RebuildOptions) null));
    }

    @Test
    void compactCollectionRejectsNullRequest() {
        EmbeddedSession session = create(42L, ignored -> {});
        assertThrows(IllegalArgumentException.class, () -> session.compactCollection(null));
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

    // --- Convenience overload delegation tests ---

    @Test
    void rebuildCollectionConvenienceOverloadDelegates() {
        AtomicLong capturedHandle = new AtomicLong();
        AtomicReference<String> capturedJson = new AtomicReference<>();
        RebuildCollectionResult fakeResult = new RebuildCollectionResult();

        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {},
                (h, json) -> { capturedHandle.set(h); capturedJson.set(json); return fakeResult; },
                STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_WAL_COLLECTION, STUB_PRUNE_WAL_ALL);

        RebuildCollectionResult result = session.rebuildCollection("myCollection");

        assertEquals(fakeResult, result);
        assertEquals(42L, capturedHandle.get());
        assertNotNull(capturedJson.get());
        assertTrue(capturedJson.get().contains("myCollection"));
    }

    @Test
    void pruneCollectionWalConvenienceOverloadDelegates() {
        AtomicLong capturedHandle = new AtomicLong();
        AtomicReference<String> capturedJson = new AtomicReference<>();
        WALPruneResult fakeResult = new WALPruneResult();

        EmbeddedSession session = new EmbeddedSession(42L, ignored -> {},
                STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                (h, json) -> { capturedHandle.set(h); capturedJson.set(json); return fakeResult; },
                STUB_PRUNE_WAL_ALL);

        WALPruneResult result = session.pruneCollectionWAL("myCollection");

        assertEquals(fakeResult, result);
        assertEquals(42L, capturedHandle.get());
        assertNotNull(capturedJson.get());
        assertTrue(capturedJson.get().contains("myCollection"));
    }
}
