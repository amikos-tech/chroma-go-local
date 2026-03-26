package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import com.google.gson.JsonParseException;

import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicReference;
import org.junit.jupiter.api.Test;

class AbstractChromaRuntimeTest {

    static class TestChromaRuntime extends AbstractChromaRuntime {
        String lastErrorValue;
        final Map<Long, String> stringStore = new HashMap<>();
        RuntimeException doCloseError;

        @Override
        protected String readBorrowedString(long address) {
            return stringStore.get(address);
        }

        @Override
        protected String readOwnedString(long address) {
            return stringStore.remove(address);
        }

        @Override
        protected String readLastError() {
            String err = lastErrorValue;
            lastErrorValue = null;
            return err;
        }

        @Override
        protected String doVersion() {
            return "test";
        }

        @Override
        protected EmbeddedSession doStartEmbedded(String configYaml) {
            return null;
        }

        @Override
        protected ServerSession doStartServer(String configYaml) {
            return null;
        }

        @Override
        protected void doClose() {
            if (doCloseError != null) throw doCloseError;
        }
    }

    @Test
    void callFfiHandle_returnsHandle_whenNonZero() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        long result = runtime.callFfiHandle(() -> 42L);
        assertEquals(42L, result);
    }

    @Test
    void callFfiHandle_throwsChromaException_whenZero() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.lastErrorValue = "test error";
        ChromaException ex = assertThrows(ChromaException.class, () -> runtime.callFfiHandle(() -> 0L));
        assertEquals("test error", ex.getMessage());
    }

    @Test
    void callFfiHandle_throwsWithDefaultMessage_whenZeroAndNoError() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        ChromaException ex = assertThrows(ChromaException.class, () -> runtime.callFfiHandle(() -> 0L));
        assertEquals("FFI call returned null handle", ex.getMessage());
    }

    @Test
    void callFfiInt_returnsValue_whenNonNegative() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        assertEquals(0, runtime.callFfiInt(() -> 0));
        assertEquals(8000, runtime.callFfiInt(() -> 8000));
        assertEquals(65535, runtime.callFfiInt(() -> 65535));
    }

    @Test
    void callFfiInt_throwsWithNativeError_whenNegative() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.lastErrorValue = "port not bound";
        ChromaException ex = assertThrows(ChromaException.class, () -> runtime.callFfiInt(() -> -1));
        assertEquals("port not bound", ex.getMessage());
    }

    @Test
    void callFfiInt_throwsWithDefaultMessage_whenNegativeAndNoError() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        ChromaException ex = assertThrows(ChromaException.class, () -> runtime.callFfiInt(() -> -42));
        assertTrue(ex.getMessage().contains("-42"));
    }

    @Test
    void callFfiJson_deserializesResult() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.stringStore.put(100L, "{\"collection_id\":\"abc\",\"name\":\"test\",\"rebuilt\":true,\"records_scanned\":42,\"vectors_reindexed\":10,\"duration_ms\":100}");
        RebuildCollectionResult result = runtime.callFfiJson(() -> 100L, RebuildCollectionResult.class);
        assertEquals("abc", result.collectionId());
        assertEquals("test", result.name());
        assertTrue(result.rebuilt());
        assertEquals(42L, result.recordsScanned());
    }

    @Test
    void callFfiJson_throwsChromaException_whenZero() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.lastErrorValue = "json error";
        ChromaException ex = assertThrows(ChromaException.class,
                () -> runtime.callFfiJson(() -> 0L, RebuildCollectionResult.class));
        assertEquals("json error", ex.getMessage());
    }

    @Test
    void callFfiVoid_doesNotThrow_whenSuccessful() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.callFfiVoid(() -> {});
    }

    @Test
    void callFfiVoid_throwsOnLastError() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.lastErrorValue = "void operation failed";
        ChromaException ex = assertThrows(ChromaException.class,
                () -> runtime.callFfiVoid(() -> {}));
        assertTrue(ex.getMessage().contains("void operation failed"));
    }

    @Test
    void callFfiVoid_propagatesExceptionFromRunnable() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        ChromaException ex = assertThrows(ChromaException.class,
                () -> runtime.callFfiVoid(() -> { throw new ChromaException("inner failure"); }));
        assertEquals("inner failure", ex.getMessage());
    }

    @Test
    void callFfiJson_throwsOnNullJson() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        ChromaException ex = assertThrows(ChromaException.class,
                () -> runtime.callFfiJson(() -> 300L, RebuildCollectionResult.class));
        assertTrue(ex.getMessage().contains("null/empty response"));
    }

    @Test
    void callFfiJson_throwsOnEmptyJson() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.stringStore.put(301L, "");
        ChromaException ex = assertThrows(ChromaException.class,
                () -> runtime.callFfiJson(() -> 301L, RebuildCollectionResult.class));
        assertTrue(ex.getMessage().contains("null/empty response"));
    }

    @Test
    void callFfiJson_throwsOnMalformedJson() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.stringStore.put(302L, "{not valid json!!!");
        ChromaException ex = assertThrows(ChromaException.class,
                () -> runtime.callFfiJson(() -> 302L, RebuildCollectionResult.class));
        assertTrue(ex.getMessage().contains("Failed to deserialize"));
    }

    @Test
    void callFfiBorrowedString_returnsString() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.stringStore.put(200L, "borrowed-value");
        String result = runtime.callFfiBorrowedString(() -> 200L);
        assertEquals("borrowed-value", result);
        assertEquals("borrowed-value", runtime.stringStore.get(200L));
    }

    @Test
    void callFfiBorrowedString_throwsChromaException_whenZero() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.lastErrorValue = "borrowed error";
        ChromaException ex = assertThrows(ChromaException.class,
                () -> runtime.callFfiBorrowedString(() -> 0L));
        assertEquals("borrowed error", ex.getMessage());
    }

    @Test
    void callFfiVoid_succeedsWithEmptyStringError() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.lastErrorValue = "";
        assertDoesNotThrow(() -> runtime.callFfiVoid(() -> {}));
    }

    @Test
    void callFfiJson_throwsOnLiteralNullJson() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.stringStore.put(400L, "null");
        ChromaException ex = assertThrows(ChromaException.class,
                () -> runtime.callFfiJson(() -> 400L, RebuildCollectionResult.class));
        assertTrue(ex.getMessage().contains("null"));
    }

    @Test
    void callFfiVoid_notAffectedByStaleError_whenProtocolFollowed() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.lastErrorValue = "old error";
        assertThrows(ChromaException.class, () -> runtime.callFfiHandle(() -> 0L));
        assertDoesNotThrow(() -> runtime.callFfiVoid(() -> {}));
    }

    @Test
    void callFfiJson_preservesExceptionCause() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.stringStore.put(500L, "{not valid json!!!");
        ChromaException ex = assertThrows(ChromaException.class,
                () -> runtime.callFfiJson(() -> 500L, RebuildCollectionResult.class));
        assertTrue(ex.getMessage().contains("Failed to deserialize"));
        assertInstanceOf(JsonParseException.class, ex.getCause());
    }

    // --- Lifecycle tests ---

    @Test
    void ensureOpen_throwsAfterClose() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.close();
        assertThrows(IllegalStateException.class, runtime::version);
    }

    @Test
    void close_isIdempotent() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.close();
        assertDoesNotThrow(runtime::close);
    }

    @Test
    void close_rollsBackOnDoCloseFailure() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.doCloseError = new ChromaException("arena still in use");
        assertThrows(ChromaException.class, runtime::close);
        // Rolled back — runtime is still open
        assertDoesNotThrow(runtime::version);
    }

    @Test
    void close_succeedsAfterDoCloseErrorIsResolved() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.doCloseError = new ChromaException("arena still in use");
        assertThrows(ChromaException.class, runtime::close);
        // Fix the error
        runtime.doCloseError = null;
        assertDoesNotThrow(runtime::close);
        assertThrows(IllegalStateException.class, runtime::version);
    }

    @Test
    void close_rejectsAllOperationsAfterSuccess() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        runtime.close();
        assertThrows(IllegalStateException.class, runtime::version);
        assertThrows(IllegalStateException.class, () -> runtime.startEmbedded("yaml"));
        assertThrows(IllegalStateException.class, () -> runtime.startServer("yaml"));
    }

    @Test
    void callFfiHandle_serializesAccess() throws Exception {
        TestChromaRuntime runtime = new TestChromaRuntime();
        AtomicBoolean insideCriticalSection = new AtomicBoolean(false);
        AtomicBoolean overlapDetected = new AtomicBoolean(false);
        CountDownLatch started = new CountDownLatch(1);
        CountDownLatch proceed = new CountDownLatch(1);

        Thread t1 = new Thread(() -> {
            runtime.callFfiHandle(() -> {
                insideCriticalSection.set(true);
                started.countDown();
                try {
                    proceed.await();
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                }
                insideCriticalSection.set(false);
                return 1L;
            });
        });

        AtomicReference<Long> t2Result = new AtomicReference<>();
        Thread t2 = new Thread(() -> {
            try {
                started.await();
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
            t2Result.set(runtime.callFfiHandle(() -> {
                if (insideCriticalSection.get()) {
                    overlapDetected.set(true);
                }
                return 2L;
            }));
        });

        t1.start();
        t2.start();

        started.await();
        Thread.sleep(50);
        proceed.countDown();

        t1.join(5000);
        t2.join(5000);

        assertEquals(false, overlapDetected.get(), "Two concurrent calls should not overlap");
        assertEquals(2L, t2Result.get());
    }
}
