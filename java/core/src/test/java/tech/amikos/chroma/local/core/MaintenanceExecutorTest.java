package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.*;

import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.BiFunction;
import java.util.function.Function;
import org.junit.jupiter.api.Test;

class MaintenanceExecutorTest {

    private static final Function<BackupOptions, BackupResult<EmbeddedSession>> STUB_EMB_BACKUP =
            opts -> { throw new UnsupportedOperationException("stub"); };

    private static EmbeddedSession fakeEmbedded() {
        BiFunction<Long, String, RebuildCollectionResult> stubRebuild = (h, j) -> { throw new UnsupportedOperationException(); };
        BiFunction<Long, String, CompactionResult> stubCompact = (h, j) -> { throw new UnsupportedOperationException(); };
        BiFunction<Long, String, WALPruneResult> stubPrune = (h, j) -> { throw new UnsupportedOperationException(); };
        return new EmbeddedSession(1L, h -> {}, stubRebuild, stubCompact, stubCompact, stubPrune, stubPrune, STUB_EMB_BACKUP);
    }

    private static final Function<BackupOptions, BackupResult<ServerSession>> STUB_BACKUP =
            opts -> { throw new UnsupportedOperationException("stub"); };
    private static final Function<RebuildOptions, MaintenanceResult<RebuildCollectionResult, ServerSession>> STUB_REBUILD =
            opts -> { throw new UnsupportedOperationException("stub"); };
    private static final Function<CompactCollectionRequest, MaintenanceResult<CompactionResult, ServerSession>> STUB_COMPACT_COLLECTION =
            req -> { throw new UnsupportedOperationException("stub"); };
    private static final Function<CompactAllRequest, MaintenanceResult<CompactionResult, ServerSession>> STUB_COMPACT_ALL =
            req -> { throw new UnsupportedOperationException("stub"); };
    private static final Function<WALPruneOptions, MaintenanceResult<WALPruneResult, ServerSession>> STUB_PRUNE_COLLECTION =
            opts -> { throw new UnsupportedOperationException("stub"); };
    private static final Function<WALPruneOptions, MaintenanceResult<WALPruneResult, ServerSession>> STUB_PRUNE_ALL =
            opts -> { throw new UnsupportedOperationException("stub"); };

    private static ServerSession fakeServer() {
        return new ServerSession(1L, h -> {}, h -> {}, h -> 8000, h -> "localhost", h -> "/data",
                STUB_BACKUP, STUB_REBUILD, STUB_COMPACT_COLLECTION, STUB_COMPACT_ALL,
                STUB_PRUNE_COLLECTION, STUB_PRUNE_ALL);
    }

    // --- Happy path ---

    @Test
    void happyPath_returnsResultAndSession() {
        MaintenanceResult<String, ServerSession> result = MaintenanceExecutor.execute(
                "yaml",
                () -> {},
                cfg -> fakeEmbedded(),
                cfg -> fakeServer(),
                emb -> "ok");

        assertEquals("ok", result.result());
        assertNotNull(result.session());
        assertNull(result.restartError());
    }

    // --- Step 1: closeServer failure ---

    @Test
    void closeServerFailure_throwsChromaException() {
        ChromaException ex = assertThrows(ChromaException.class, () ->
                MaintenanceExecutor.execute(
                        "yaml",
                        () -> { throw new RuntimeException("stop failed"); },
                        cfg -> fakeEmbedded(),
                        cfg -> fakeServer(),
                        emb -> "ok"));

        assertTrue(ex.getMessage().contains("stop/free server"));
    }

    // --- Step 2: embedded start failure ---

    @Test
    void embeddedStartFailure_throwsChromaException() {
        ChromaException ex = assertThrows(ChromaException.class, () ->
                MaintenanceExecutor.execute(
                        "yaml",
                        () -> {},
                        cfg -> { throw new RuntimeException("embed failed"); },
                        cfg -> fakeServer(),
                        emb -> "ok"));

        assertTrue(ex.getMessage().contains("embedded runtime"));
        assertNotNull(ex.getCause());
    }

    // --- opError only ---

    @Test
    void opError_rethrown() {
        RuntimeException ex = assertThrows(RuntimeException.class, () ->
                MaintenanceExecutor.execute(
                        "yaml",
                        () -> {},
                        cfg -> fakeEmbedded(),
                        cfg -> fakeServer(),
                        emb -> { throw new RuntimeException("op failed"); }));

        assertEquals("op failed", ex.getMessage());
        assertEquals(0, ex.getSuppressed().length);
    }

    // --- opError + closeError ---

    @Test
    void opErrorAndCloseError_opErrorWithSuppressed() {
        AtomicBoolean closeCalled = new AtomicBoolean();
        EmbeddedSession throwingEmbedded = new EmbeddedSession(1L,
                h -> { closeCalled.set(true); throw new RuntimeException("close failed"); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                STUB_EMB_BACKUP);

        RuntimeException ex = assertThrows(RuntimeException.class, () ->
                MaintenanceExecutor.execute(
                        "yaml",
                        () -> {},
                        cfg -> throwingEmbedded,
                        cfg -> fakeServer(),
                        emb -> { throw new RuntimeException("op failed"); }));

        assertEquals("op failed", ex.getMessage());
        assertTrue(closeCalled.get());
        assertEquals(1, ex.getSuppressed().length);
        assertEquals("close failed", ex.getSuppressed()[0].getMessage());
    }

    // --- opError + restartError ---

    @Test
    void opErrorAndRestartError_combinedChromaException() {
        ChromaException ex = assertThrows(ChromaException.class, () ->
                MaintenanceExecutor.execute(
                        "yaml",
                        () -> {},
                        cfg -> fakeEmbedded(),
                        cfg -> { throw new RuntimeException("restart failed"); },
                        emb -> { throw new RuntimeException("op failed"); }));

        assertTrue(ex.getMessage().contains("op failed"));
        assertTrue(ex.getMessage().contains("restart failed"));
        assertTrue(ex.getMessage().contains("server remains stopped"));
    }

    // --- opError + closeError + restartError ---

    @Test
    void opErrorAndCloseErrorAndRestartError_allChained() {
        EmbeddedSession throwingEmbedded = new EmbeddedSession(1L,
                h -> { throw new RuntimeException("close failed"); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                STUB_EMB_BACKUP);

        ChromaException ex = assertThrows(ChromaException.class, () ->
                MaintenanceExecutor.execute(
                        "yaml",
                        () -> {},
                        cfg -> throwingEmbedded,
                        cfg -> { throw new RuntimeException("restart failed"); },
                        emb -> { throw new RuntimeException("op failed"); }));

        assertTrue(ex.getMessage().contains("op failed"));
        assertTrue(ex.getMessage().contains("restart failed"));
        // closeError and restartError are suppressed
        assertTrue(ex.getSuppressed().length >= 1);
    }

    // --- closeError only ---

    @Test
    void closeErrorOnly_returnsResultWithError() {
        EmbeddedSession throwingEmbedded = new EmbeddedSession(1L,
                h -> { throw new RuntimeException("close failed"); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                STUB_EMB_BACKUP);

        MaintenanceResult<String, ServerSession> result = MaintenanceExecutor.execute(
                "yaml",
                () -> {},
                cfg -> throwingEmbedded,
                cfg -> fakeServer(),
                emb -> "ok");

        assertEquals("ok", result.result());
        assertNotNull(result.session());
        assertNotNull(result.restartError());
        assertTrue(result.restartError().getMessage().contains("close temporary embedded"));
    }

    // --- closeError + restartError ---

    @Test
    void closeErrorAndRestartError_nullSessionWithCombinedError() {
        EmbeddedSession throwingEmbedded = new EmbeddedSession(1L,
                h -> { throw new RuntimeException("close failed"); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                (h, j) -> { throw new UnsupportedOperationException(); },
                STUB_EMB_BACKUP);

        MaintenanceResult<String, ServerSession> result = MaintenanceExecutor.execute(
                "yaml",
                () -> {},
                cfg -> throwingEmbedded,
                cfg -> { throw new RuntimeException("restart failed"); },
                emb -> "ok");

        assertEquals("ok", result.result());
        assertNull(result.session());
        assertNotNull(result.restartError());
        assertTrue(result.restartError().getMessage().contains("server remains stopped"));
    }

    // --- restartError only ---

    @Test
    void restartErrorOnly_nullSessionWithError() {
        MaintenanceResult<String, ServerSession> result = MaintenanceExecutor.execute(
                "yaml",
                () -> {},
                cfg -> fakeEmbedded(),
                cfg -> { throw new RuntimeException("restart failed"); },
                emb -> "ok");

        assertEquals("ok", result.result());
        assertNull(result.session());
        assertEquals("restart failed", result.restartError().getMessage());
    }

    // --- Parameter validation ---

    @Test
    void rejectsNullConfigYaml() {
        assertThrows(NullPointerException.class, () ->
                MaintenanceExecutor.execute(null, () -> {}, cfg -> fakeEmbedded(),
                        cfg -> fakeServer(), emb -> "ok"));
    }

    @Test
    void rejectsNullCloseServerAction() {
        assertThrows(NullPointerException.class, () ->
                MaintenanceExecutor.execute("yaml", null, cfg -> fakeEmbedded(),
                        cfg -> fakeServer(), emb -> "ok"));
    }

    @Test
    void rejectsNullStartEmbeddedAction() {
        assertThrows(NullPointerException.class, () ->
                MaintenanceExecutor.execute("yaml", () -> {}, null,
                        cfg -> fakeServer(), emb -> "ok"));
    }

    @Test
    void rejectsNullStartServerAction() {
        assertThrows(NullPointerException.class, () ->
                MaintenanceExecutor.execute("yaml", () -> {}, cfg -> fakeEmbedded(),
                        null, emb -> "ok"));
    }

    @Test
    void rejectsNullOperation() {
        assertThrows(NullPointerException.class, () ->
                MaintenanceExecutor.execute("yaml", () -> {}, cfg -> fakeEmbedded(),
                        cfg -> fakeServer(), null));
    }
}
