package tech.amikos.chroma.local.jna;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.IOException;
import java.net.ServerSocket;
import java.nio.file.Files;
import java.nio.file.Path;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.CleanupMode;
import org.junit.jupiter.api.io.TempDir;
import tech.amikos.chroma.local.core.BackupOptions;
import tech.amikos.chroma.local.core.BackupResult;
import tech.amikos.chroma.local.core.ServerConfigBuilder;
import tech.amikos.chroma.local.core.ServerSession;

class JnaServerBackupTest {

    @Test
    void serverBackupCreatesDirectoryWithManifest(
            @TempDir(cleanup = CleanupMode.NEVER) Path persistDir,
            @TempDir Path backupDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        Files.writeString(persistDir.resolve("sentinel.txt"), "server-backup-test");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port)
                .listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        Path dest = backupDir.resolve("output");

        try (JnaChromaRuntime runtime = JnaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            BackupResult<ServerSession> result = session.backup(
                    new BackupOptions.Builder(dest.toString()).includeMetadata(true).build());

            assertNotNull(result.manifest());
            assertEquals("v1", result.manifest().schemaVersion());
            assertEquals("server", result.manifest().mode());
            assertTrue(result.manifest().fileCount() >= 1);
            assertTrue(Files.exists(dest.resolve("persist").resolve("sentinel.txt")));
            assertTrue(Files.exists(dest.resolve("backup_manifest.json")));
            assertNotNull(result.session());
            assertTrue(result.session().port() > 0);

            result.session().close();
        }
    }

    @Test
    void serverBackupWithLeaveInactive(
            @TempDir(cleanup = CleanupMode.NEVER) Path persistDir,
            @TempDir Path backupDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        Files.writeString(persistDir.resolve("sentinel.txt"), "leave-inactive-test");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port)
                .listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        Path dest = backupDir.resolve("output");

        try (JnaChromaRuntime runtime = JnaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            BackupResult<ServerSession> result = session.backup(
                    new BackupOptions.Builder(dest.toString()).leaveInactive(true).build());

            assertNull(result.session());
            assertNotNull(result.manifest());
            assertTrue(Files.exists(dest.resolve("backup_manifest.json")));
        }
    }

    @Test
    void serverBackupRejectsNullOptions(
            @TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port)
                .listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (JnaChromaRuntime runtime = JnaChromaRuntime.init(libPath);
             ServerSession session = runtime.startServer(yaml)) {
            assertThrows(IllegalArgumentException.class, () -> session.backup(null));
        }
    }

    @Test
    void serverBackupThrowsAfterClose(
            @TempDir(cleanup = CleanupMode.NEVER) Path persistDir,
            @TempDir Path backupDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port)
                .listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        Path dest = backupDir.resolve("output");

        try (JnaChromaRuntime runtime = JnaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            session.close();

            assertThrows(IllegalStateException.class,
                    () -> session.backup(
                            new BackupOptions.Builder(dest.toString()).build()));
        }
    }

    private static int findFreePort() throws IOException {
        try (ServerSocket s = new ServerSocket(0)) {
            return s.getLocalPort();
        }
    }
}
