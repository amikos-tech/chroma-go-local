package tech.amikos.chroma.local.panama;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.nio.file.Files;
import java.nio.file.Path;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.CleanupMode;
import org.junit.jupiter.api.io.TempDir;
import tech.amikos.chroma.local.core.BackupOptions;
import tech.amikos.chroma.local.core.BackupResult;
import tech.amikos.chroma.local.core.EmbeddedConfigBuilder;
import tech.amikos.chroma.local.core.EmbeddedSession;

class PanamaEmbeddedBackupTest {

    @Test
    void embeddedBackupCreatesDirectoryWithManifest(
            @TempDir(cleanup = CleanupMode.NEVER) Path persistDir,
            @TempDir Path backupDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        Files.writeString(persistDir.resolve("sentinel.txt"), "backup-test-data");

        String yaml = new EmbeddedConfigBuilder()
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        Path dest = backupDir.resolve("output");

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            EmbeddedSession session = runtime.startEmbedded(yaml);
            BackupResult<EmbeddedSession> result = session.backup(
                    new BackupOptions.Builder(dest.toString()).includeMetadata(true).build());

            assertNotNull(result.manifest());
            assertEquals("v1", result.manifest().schemaVersion());
            assertEquals("embedded", result.manifest().mode());
            assertTrue(result.manifest().fileCount() >= 1);
            assertTrue(Files.exists(dest.resolve("persist").resolve("sentinel.txt")));
            assertEquals("backup-test-data",
                    Files.readString(dest.resolve("persist").resolve("sentinel.txt")));
            assertTrue(Files.exists(dest.resolve("backup_manifest.json")));
            assertNotNull(result.session());

            result.session().close();
        }
    }

    @Test
    void embeddedBackupWithLeaveClosed(
            @TempDir(cleanup = CleanupMode.NEVER) Path persistDir,
            @TempDir Path backupDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        Files.writeString(persistDir.resolve("sentinel.txt"), "leave-closed-test");

        String yaml = new EmbeddedConfigBuilder()
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        Path dest = backupDir.resolve("output");

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            EmbeddedSession session = runtime.startEmbedded(yaml);
            BackupResult<EmbeddedSession> result = session.backup(
                    new BackupOptions.Builder(dest.toString()).leaveClosed(true).build());

            assertNull(result.session());
            assertNotNull(result.manifest());
            assertTrue(Files.exists(dest.resolve("backup_manifest.json")));
        }
    }

    @Test
    void embeddedBackupRejectsLeaveStopped(
            @TempDir(cleanup = CleanupMode.NEVER) Path persistDir,
            @TempDir Path backupDir) {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        String yaml = new EmbeddedConfigBuilder()
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        Path dest = backupDir.resolve("output");

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath);
             EmbeddedSession session = runtime.startEmbedded(yaml)) {
            IllegalArgumentException ex = assertThrows(IllegalArgumentException.class,
                    () -> session.backup(
                            new BackupOptions.Builder(dest.toString()).leaveStopped(true).build()));
            assertTrue(ex.getMessage().contains("leaveStopped"));
        }
    }

    @Test
    void embeddedBackupRejectsNullOptions(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        String yaml = new EmbeddedConfigBuilder()
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath);
             EmbeddedSession session = runtime.startEmbedded(yaml)) {
            assertThrows(IllegalArgumentException.class, () -> session.backup(null));
        }
    }

    @Test
    void embeddedBackupThrowsAfterClose(
            @TempDir(cleanup = CleanupMode.NEVER) Path persistDir,
            @TempDir Path backupDir) {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        String yaml = new EmbeddedConfigBuilder()
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        Path dest = backupDir.resolve("output");

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            EmbeddedSession session = runtime.startEmbedded(yaml);
            session.close();

            assertThrows(IllegalStateException.class,
                    () -> session.backup(
                            new BackupOptions.Builder(dest.toString()).build()));
        }
    }
}
