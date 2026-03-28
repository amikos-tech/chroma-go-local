package tech.amikos.chroma.local.panama;

import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.IOException;
import java.net.ServerSocket;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.file.Path;
import java.time.Duration;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.CleanupMode;
import org.junit.jupiter.api.io.TempDir;
import tech.amikos.chroma.local.core.CompactAllRequest;
import tech.amikos.chroma.local.core.CompactCollectionRequest;
import tech.amikos.chroma.local.core.CompactionResult;
import tech.amikos.chroma.local.core.MaintenanceResult;
import tech.amikos.chroma.local.core.RebuildCollectionResult;
import tech.amikos.chroma.local.core.RebuildOptions;
import tech.amikos.chroma.local.core.ServerConfigBuilder;
import tech.amikos.chroma.local.core.ServerSession;
import tech.amikos.chroma.local.core.WALPruneOptions;
import tech.amikos.chroma.local.core.WALPruneResult;

class PanamaServerMaintenanceTest {

    private static final HttpClient HTTP = HttpClient.newHttpClient();

    @Test
    void serverRebuildCollection(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port)
                .listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            waitForReady(session.url(), Duration.ofSeconds(15));
            createCollection(session.url(), "rebuild_test");

            MaintenanceResult<RebuildCollectionResult, ServerSession> result =
                    session.rebuildCollection("rebuild_test");

            assertNotNull(result.result());
            assertNull(result.restartError());
            try (ServerSession newSession = result.session()) {
                assertNotNull(newSession);
                verifyServerResponds(newSession.url());
                verifyCollectionExists(newSession.url(), "rebuild_test");
            }
        }
    }

    @Test
    void serverCompactCollection(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port)
                .listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            waitForReady(session.url(), Duration.ofSeconds(15));
            createCollection(session.url(), "compact_test");

            MaintenanceResult<CompactionResult, ServerSession> result =
                    session.compactCollection("compact_test");

            assertNotNull(result.result());
            assertNull(result.restartError());
            try (ServerSession newSession = result.session()) {
                assertNotNull(newSession);
                verifyServerResponds(newSession.url());
                verifyCollectionExists(newSession.url(), "compact_test");
            }
        }
    }

    @Test
    void serverCompactAll(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port)
                .listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            waitForReady(session.url(), Duration.ofSeconds(15));
            createCollection(session.url(), "compact_all_test");

            MaintenanceResult<CompactionResult, ServerSession> result =
                    session.compactAll(new CompactAllRequest.Builder().build());

            assertNotNull(result.result());
            assertNull(result.restartError());
            try (ServerSession newSession = result.session()) {
                assertNotNull(newSession);
                verifyServerResponds(newSession.url());
            }
        }
    }

    @Test
    void serverPruneCollectionWAL(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port)
                .listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            waitForReady(session.url(), Duration.ofSeconds(15));
            createCollection(session.url(), "prune_test");

            MaintenanceResult<WALPruneResult, ServerSession> result =
                    session.pruneCollectionWAL("prune_test");

            assertNotNull(result.result());
            assertNull(result.restartError());
            try (ServerSession newSession = result.session()) {
                assertNotNull(newSession);
                verifyServerResponds(newSession.url());
                verifyCollectionExists(newSession.url(), "prune_test");
            }
        }
    }

    @Test
    void serverPruneAllWAL(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port)
                .listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            waitForReady(session.url(), Duration.ofSeconds(15));
            createCollection(session.url(), "prune_all_test");

            MaintenanceResult<WALPruneResult, ServerSession> result =
                    session.pruneAllWAL(WALPruneOptions.defaults("ignored"));

            assertNotNull(result.result());
            assertNull(result.restartError());
            try (ServerSession newSession = result.session()) {
                assertNotNull(newSession);
                verifyServerResponds(newSession.url());
            }
        }
    }

    @Test
    void serverMaintenanceThrowsAfterClose(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port)
                .listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            session.close();

            assertThrows(IllegalStateException.class, () -> session.rebuildCollection("x"));
            assertThrows(IllegalStateException.class, () -> session.compactCollection("x"));
            assertThrows(IllegalStateException.class, () -> session.pruneCollectionWAL("x"));
        }
    }

    @Test
    void serverRebuildRejectsNullOptions(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port).listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true).build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            assertThrows(IllegalArgumentException.class,
                    () -> session.rebuildCollection((RebuildOptions) null));
            assertDoesNotThrow(session::port, "session must remain usable after null-arg rejection");
            session.close();
        }
    }

    @Test
    void serverCompactCollectionRejectsNullOptions(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port).listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true).build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            assertThrows(IllegalArgumentException.class,
                    () -> session.compactCollection((CompactCollectionRequest) null));
            assertDoesNotThrow(session::port, "session must remain usable after null-arg rejection");
            session.close();
        }
    }

    @Test
    void serverCompactAllRejectsNullOptions(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port).listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true).build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            assertThrows(IllegalArgumentException.class,
                    () -> session.compactAll(null));
            assertDoesNotThrow(session::port, "session must remain usable after null-arg rejection");
            session.close();
        }
    }

    @Test
    void serverPruneWALRejectsNullOptions(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port).listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true).build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            assertThrows(IllegalArgumentException.class,
                    () -> session.pruneCollectionWAL((WALPruneOptions) null));
            assertDoesNotThrow(session::port, "session must remain usable after null-arg rejection");
            session.close();
        }
    }

    @Test
    void serverPruneAllWALRejectsNullOptions(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port).listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true).build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            assertThrows(IllegalArgumentException.class,
                    () -> session.pruneAllWAL(null));
            assertDoesNotThrow(session::port, "session must remain usable after null-arg rejection");
            session.close();
        }
    }

    @Test
    void serverRebuildNonexistentCollectionThrows(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port).listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true).build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            waitForReady(session.url(), Duration.ofSeconds(15));
            assertThrows(RuntimeException.class,
                    () -> session.rebuildCollection("nonexistent_collection"));
        }
    }

    private static void waitForReady(String baseUrl, Duration timeout) throws Exception {
        long deadline = System.currentTimeMillis() + timeout.toMillis();
        while (System.currentTimeMillis() < deadline) {
            try {
                HttpResponse<String> resp = HTTP.send(
                        HttpRequest.newBuilder(URI.create(baseUrl + "/api/v2/heartbeat"))
                                .GET().timeout(Duration.ofSeconds(2)).build(),
                        HttpResponse.BodyHandlers.ofString());
                if (resp.statusCode() == 200) return;
            } catch (Exception ignored) {
            }
            Thread.sleep(100);
        }
        throw new AssertionError("Server not ready within " + timeout);
    }

    private static void createCollection(String baseUrl, String name) throws Exception {
        String body = "{\"name\":\"" + name + "\",\"get_or_create\":true}";
        HttpResponse<String> resp = HTTP.send(
                HttpRequest.newBuilder(URI.create(
                                baseUrl + "/api/v2/tenants/default_tenant/databases/default_database/collections"))
                        .POST(HttpRequest.BodyPublishers.ofString(body))
                        .header("Content-Type", "application/json")
                        .timeout(Duration.ofSeconds(10))
                        .build(),
                HttpResponse.BodyHandlers.ofString());
        assertTrue(resp.statusCode() >= 200 && resp.statusCode() < 300,
                "Failed to create collection '" + name + "': HTTP " + resp.statusCode() + " " + resp.body());
    }

    private static void verifyServerResponds(String baseUrl) throws Exception {
        waitForReady(baseUrl, Duration.ofSeconds(15));
    }

    private static void verifyCollectionExists(String baseUrl, String collectionName) throws Exception {
        HttpResponse<String> resp = HTTP.send(
                HttpRequest.newBuilder(URI.create(
                                baseUrl + "/api/v2/tenants/default_tenant/databases/default_database/collections"))
                        .GET().timeout(Duration.ofSeconds(10)).build(),
                HttpResponse.BodyHandlers.ofString());
        assertTrue(resp.body().contains(collectionName),
                "Collection " + collectionName + " should exist after maintenance op");
    }

    private static int findFreePort() throws IOException {
        try (ServerSocket s = new ServerSocket(0)) {
            return s.getLocalPort();
        }
    }
}
