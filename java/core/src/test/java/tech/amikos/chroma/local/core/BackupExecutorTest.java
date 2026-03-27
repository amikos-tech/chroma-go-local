package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.concurrent.atomic.AtomicBoolean;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

class BackupExecutorTest {

    @TempDir
    Path tempDir;

    @Test
    void executeCopiesFilesAndWritesManifest() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        Files.writeString(source.resolve("sentinel.txt"), "test-data");

        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString())
                .includeMetadata(true)
                .build();

        AtomicBoolean closed = new AtomicBoolean(false);
        BackupResult<String> result = BackupExecutor.execute(
                "embedded", source.toString(), options,
                () -> closed.set(true),
                () -> "new-session");

        assertTrue(closed.get());
        assertNotNull(result.manifest());
        assertEquals("new-session", result.session());

        Path sentinelCopy = dest.resolve("persist").resolve("sentinel.txt");
        assertTrue(Files.exists(sentinelCopy));
        assertEquals("test-data", Files.readString(sentinelCopy));

        Path manifestFile = dest.resolve("backup_manifest.json");
        assertTrue(Files.exists(manifestFile));

        BackupManifest manifest = JsonUtil.fromJson(Files.readString(manifestFile), BackupManifest.class);
        assertNotNull(manifest);
        assertEquals("v1", manifest.schemaVersion());
        assertEquals("embedded", manifest.mode());
        assertTrue(manifest.fileCount() >= 1);
        assertTrue(manifest.totalBytes() > 0);
        assertEquals(1, manifest.files().size());
    }

    @Test
    void executeLeaveClosedReturnsNullSession() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);

        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString())
                .leaveClosed(true)
                .build();

        BackupResult<String> result = BackupExecutor.execute(
                "embedded", source.toString(), options,
                () -> {},
                () -> "should-not-be-called");

        assertNotNull(result.manifest());
        assertNull(result.session());
    }

    @Test
    void rejectsLeaveStoppedForEmbeddedMode() {
        Path source = tempDir.resolve("source");
        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString())
                .leaveStopped(true)
                .build();

        assertThrows(IllegalArgumentException.class, () ->
                BackupExecutor.execute("embedded", source.toString(), options, () -> {}, () -> "x"));
    }

    @Test
    void rejectsLeaveClosedForServerMode() {
        Path source = tempDir.resolve("source");
        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString())
                .leaveClosed(true)
                .build();

        assertThrows(IllegalArgumentException.class, () ->
                BackupExecutor.execute("server", source.toString(), options, () -> {}, () -> "x"));
    }

    @Test
    void rejectsDestinationInsideSource() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        Path dest = source.resolve("backups").resolve("today");

        BackupOptions options = new BackupOptions.Builder(dest.toString()).build();

        assertThrows(IllegalArgumentException.class, () ->
                BackupExecutor.execute("embedded", source.toString(), options, () -> {}, () -> "x"));
    }

    @Test
    void rejectsNonEmptyDestination() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        Path dest = tempDir.resolve("backup");
        Files.createDirectories(dest);
        Files.writeString(dest.resolve("existing.txt"), "data");

        BackupOptions options = new BackupOptions.Builder(dest.toString()).build();

        assertThrows(IllegalArgumentException.class, () ->
                BackupExecutor.execute("embedded", source.toString(), options, () -> {}, () -> "x"));
    }

    @Test
    void handlesNonExistentSourcePath() {
        Path source = tempDir.resolve("does-not-exist");
        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString()).build();

        BackupResult<String> result = BackupExecutor.execute(
                "embedded", source.toString(), options,
                () -> {},
                () -> "new-session");

        assertNotNull(result.manifest());
        assertEquals(0, result.manifest().fileCount());
        assertEquals(0, result.manifest().totalBytes());
    }

    @Test
    void extractPersistPathFromTopLevel() {
        String yaml = "persist_path: /tmp/chroma\nsqlite_filename: chroma.sqlite3\n";
        assertEquals("/tmp/chroma", BackupExecutor.extractPersistPath(yaml));
    }

    @Test
    void extractPersistPathFromNestedChromaKey() {
        String yaml = "chroma:\n  persist_path: /data/chroma\n";
        assertEquals("/data/chroma", BackupExecutor.extractPersistPath(yaml));
    }
}
