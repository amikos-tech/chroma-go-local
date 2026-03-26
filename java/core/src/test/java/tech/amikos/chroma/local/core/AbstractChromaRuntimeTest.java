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
        public String version() {
            return "test";
        }

        @Override
        public EmbeddedSession startEmbedded(String configYaml) {
            return null;
        }

        @Override
        public ServerSession startServer(String configYaml) {
            return null;
        }

        @Override
        public void close() {}
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
    void callFfiJson_throwsOnNullJson() {
        TestChromaRuntime runtime = new TestChromaRuntime();
        // pointer 300 maps to no entry, so readOwnedString returns null
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
        // borrowed string is not removed from store
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
        // Simulate: a previous call failed and drained the error via readLastError
        runtime.lastErrorValue = "old error";
        assertThrows(ChromaException.class, () -> runtime.callFfiHandle(() -> 0L));
        // Error was consumed by the failure path. Next void call should succeed.
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
            // t1 should still be in the critical section at this point
            t2Result.set(runtime.callFfiHandle(() -> {
                if (insideCriticalSection.get()) {
                    overlapDetected.set(true);
                }
                return 2L;
            }));
        });

        t1.start();
        t2.start();

        // Wait for t1 to enter critical section
        started.await();
        // Give t2 time to attempt lock acquisition
        Thread.sleep(50);
        // Release t1
        proceed.countDown();

        t1.join(5000);
        t2.join(5000);

        assertEquals(false, overlapDetected.get(), "Two concurrent calls should not overlap");
        assertEquals(2L, t2Result.get());
    }
}
