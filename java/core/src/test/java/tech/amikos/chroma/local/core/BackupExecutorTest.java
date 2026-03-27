package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.MessageDigest;
import java.util.HexFormat;
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
                BackupMode.EMBEDDED, source.toString(), options,
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

        BackupManifest manifest = result.manifest();
        assertNotNull(manifest);
        assertEquals("v1", manifest.schemaVersion());
        assertEquals("embedded", manifest.mode());
        assertTrue(manifest.fileCount() >= 1);
        assertTrue(manifest.totalBytes() > 0);
        assertEquals(1, manifest.files().size());
    }

    @Test
    void leaveInactiveEmbeddedReturnsNullSession() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);

        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString())
                .leaveInactive(true)
                .build();

        BackupResult<String> result = BackupExecutor.execute(
                BackupMode.EMBEDDED, source.toString(), options,
                () -> {},
                () -> "should-not-be-called");

        assertNotNull(result.manifest());
        assertNull(result.session());
    }

    @Test
    void leaveInactiveServerReturnsNullSession() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);

        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString())
                .leaveInactive(true)
                .build();

        BackupResult<String> result = BackupExecutor.execute(
                BackupMode.SERVER, source.toString(), options,
                () -> {},
                () -> "should-not-be-called");

        assertNotNull(result.manifest());
        assertNull(result.session());
        assertEquals("server", result.manifest().mode());
    }

    @Test
    void rejectsDestinationInsideSource() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        Path dest = source.resolve("backups").resolve("today");

        BackupOptions options = new BackupOptions.Builder(dest.toString()).build();

        assertThrows(IllegalArgumentException.class, () ->
                BackupExecutor.execute(BackupMode.EMBEDDED, source.toString(), options, () -> {}, () -> "x"));
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
                BackupExecutor.execute(BackupMode.EMBEDDED, source.toString(), options, () -> {}, () -> "x"));
    }

    @Test
    void rejectsDestinationThatIsAFile() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        Path dest = tempDir.resolve("backup-file");
        Files.writeString(dest, "i am a file");

        BackupOptions options = new BackupOptions.Builder(dest.toString()).build();

        assertThrows(IllegalArgumentException.class, () ->
                BackupExecutor.execute(BackupMode.EMBEDDED, source.toString(), options, () -> {}, () -> "x"));
    }

    @Test
    void rejectsNonExistentSourcePath() {
        Path source = tempDir.resolve("does-not-exist");
        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString()).build();

        AtomicBoolean restarted = new AtomicBoolean(false);
        ChromaException ex = assertThrows(ChromaException.class, () ->
                BackupExecutor.execute(BackupMode.EMBEDDED, source.toString(), options,
                        () -> {},
                        () -> { restarted.set(true); return "new-session"; }));

        assertTrue(ex.getMessage().contains("backup source path does not exist"));
        assertTrue(restarted.get(), "restart must be attempted even after source-not-found error");
    }

    @Test
    void includeMetadataFalseProducesEmptyFilesList() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        Files.writeString(source.resolve("data.txt"), "content");

        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString())
                .includeMetadata(false)
                .build();

        BackupResult<String> result = BackupExecutor.execute(
                BackupMode.EMBEDDED, source.toString(), options,
                () -> {},
                () -> "session");

        BackupManifest manifest = result.manifest();
        assertTrue(manifest.fileCount() >= 1);
        assertTrue(manifest.totalBytes() > 0);
        assertTrue(manifest.files().isEmpty());
    }

    @Test
    void sha256HashIsCorrect() throws Exception {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        String content = "hello-sha256-test";
        Files.writeString(source.resolve("hashme.txt"), content);

        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString())
                .includeMetadata(true)
                .build();

        BackupResult<String> result = BackupExecutor.execute(
                BackupMode.EMBEDDED, source.toString(), options,
                () -> {},
                () -> "session");

        BackupFileMetadata meta = result.manifest().files().get(0);
        MessageDigest digest = MessageDigest.getInstance("SHA-256");
        String expected = HexFormat.of().formatHex(digest.digest(content.getBytes()));
        assertEquals(expected, meta.sha256());
    }

    @Test
    void nestedDirectoryStructureIsCopied() throws IOException {
        Path source = tempDir.resolve("source");
        Path subdir = source.resolve("subdir");
        Files.createDirectories(subdir);
        Files.writeString(subdir.resolve("nested.txt"), "nested-content");

        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString())
                .includeMetadata(true)
                .build();

        BackupResult<String> result = BackupExecutor.execute(
                BackupMode.EMBEDDED, source.toString(), options,
                () -> {},
                () -> "session");

        Path copiedNested = dest.resolve("persist").resolve("subdir").resolve("nested.txt");
        assertTrue(Files.exists(copiedNested));
        assertEquals("nested-content", Files.readString(copiedNested));

        BackupFileMetadata meta = result.manifest().files().get(0);
        assertEquals("subdir/nested.txt", meta.path());
    }

    @Test
    void symlinkInSourceIsRejected() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        Path target = tempDir.resolve("target-file");
        Files.writeString(target, "real-content");
        Files.createSymbolicLink(source.resolve("link.txt"), target);

        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString()).build();

        assertThrows(ChromaException.class, () ->
                BackupExecutor.execute(BackupMode.EMBEDDED, source.toString(), options,
                        () -> {},
                        () -> "session"));
    }

    @Test
    void closeActionFailureCleansUpDestination() {
        Path source = tempDir.resolve("source");
        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString()).build();

        assertThrows(RuntimeException.class, () ->
                BackupExecutor.execute(BackupMode.EMBEDDED, source.toString(), options,
                        () -> { throw new RuntimeException("close failed"); },
                        () -> "session"));

        assertTrue(!Files.exists(dest) || isEmptyDir(dest));
    }

    @Test
    void restartIsAttemptedOnCopyFailure() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        Path target = tempDir.resolve("target-file");
        Files.writeString(target, "real-content");
        Files.createSymbolicLink(source.resolve("link.txt"), target);

        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString()).build();

        AtomicBoolean restarted = new AtomicBoolean(false);
        ChromaException ex = assertThrows(ChromaException.class, () ->
                BackupExecutor.execute(BackupMode.EMBEDDED, source.toString(), options,
                        () -> {},
                        () -> { restarted.set(true); return "restarted"; }));

        assertTrue(restarted.get(), "restart must be attempted even after copy failure");
        assertTrue(ex.getCause() instanceof IOException, "original IOException must be preserved as cause");
    }

    @Test
    void copyFailureWithRestartFailureIncludesBothErrors() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        Path target = tempDir.resolve("target-file");
        Files.writeString(target, "real-content");
        Files.createSymbolicLink(source.resolve("link.txt"), target);

        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString()).build();

        ChromaException ex = assertThrows(ChromaException.class, () ->
                BackupExecutor.execute(BackupMode.EMBEDDED, source.toString(), options,
                        () -> {},
                        () -> { throw new RuntimeException("restart failed"); }));

        assertTrue(ex.getMessage().contains("restart also failed"));
        assertTrue(ex.getSuppressed().length >= 1, "backup error must be suppressed on restart failure");
    }

    @Test
    void copyFailureInLeaveInactivePathThrows() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        Path target = tempDir.resolve("target-file");
        Files.writeString(target, "real-content");
        Files.createSymbolicLink(source.resolve("link.txt"), target);

        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString())
                .leaveInactive(true)
                .build();

        ChromaException ex = assertThrows(ChromaException.class, () ->
                BackupExecutor.execute(BackupMode.EMBEDDED, source.toString(), options,
                        () -> {},
                        () -> "should-not-be-called"));

        assertNotNull(ex.getCause());
    }

    @Test
    void copyFailureInLeaveInactivePathCleansUpDestination() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        Path target = tempDir.resolve("target-file");
        Files.writeString(target, "real-content");
        Files.createSymbolicLink(source.resolve("link.txt"), target);

        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString())
                .leaveInactive(true)
                .build();

        assertThrows(RuntimeException.class, () ->
                BackupExecutor.execute(BackupMode.EMBEDDED, source.toString(), options,
                        () -> {},
                        () -> "should-not-be-called"));

        assertTrue(!Files.exists(dest) || isEmptyDir(dest),
                "partial backup must be cleaned up in leave-inactive path");
    }

    // --- Fix #11: null argument rejection ---

    @Test
    void executeRejectsNullMode() {
        BackupOptions options = new BackupOptions.Builder("/tmp/dest").build();
        assertThrows(NullPointerException.class, () ->
                BackupExecutor.execute(null, "/src", options, () -> {}, () -> "s"));
    }

    @Test
    void executeRejectsNullPersistPath() {
        BackupOptions options = new BackupOptions.Builder("/tmp/dest").build();
        assertThrows(NullPointerException.class, () ->
                BackupExecutor.execute(BackupMode.EMBEDDED, null, options, () -> {}, () -> "s"));
    }

    @Test
    void executeRejectsNullOptions() {
        assertThrows(NullPointerException.class, () ->
                BackupExecutor.execute(BackupMode.EMBEDDED, "/src", null, () -> {}, () -> "s"));
    }

    @Test
    void executeRejectsNullCloseAction() {
        BackupOptions options = new BackupOptions.Builder("/tmp/dest").build();
        assertThrows(NullPointerException.class, () ->
                BackupExecutor.execute(BackupMode.EMBEDDED, "/src", options, null, () -> "s"));
    }

    @Test
    void executeRejectsNullRestartAction() {
        BackupOptions options = new BackupOptions.Builder("/tmp/dest").build();
        assertThrows(NullPointerException.class, () ->
                BackupExecutor.execute(BackupMode.EMBEDDED, "/src", options, () -> {}, null));
    }

    // --- Fix #14: manifest list immutability ---

    @Test
    void manifestSourcePathsIsImmutable() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString()).build();
        BackupResult<String> result = BackupExecutor.execute(
                BackupMode.EMBEDDED, source.toString(), options, () -> {}, () -> "s");

        assertThrows(UnsupportedOperationException.class, () ->
                result.manifest().sourcePaths().add("evil"));
    }

    @Test
    void manifestFilesIsImmutable() throws IOException {
        Path source = tempDir.resolve("source");
        Files.createDirectories(source);
        Files.writeString(source.resolve("f.txt"), "data");
        Path dest = tempDir.resolve("backup");

        BackupOptions options = new BackupOptions.Builder(dest.toString())
                .includeMetadata(true)
                .build();
        BackupResult<String> result = BackupExecutor.execute(
                BackupMode.EMBEDDED, source.toString(), options, () -> {}, () -> "s");

        assertThrows(UnsupportedOperationException.class, () ->
                result.manifest().files().add(new BackupFileMetadata("x", 0, "0644", "abc", "now")));
    }

    // --- extractPersistPath ---

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

    @Test
    void extractPersistPathTopLevelTakesPrecedence() {
        String yaml = "persist_path: /top\nchroma:\n  persist_path: /nested\n";
        assertEquals("/top", BackupExecutor.extractPersistPath(yaml));
    }

    @Test
    void extractPersistPathThrowsForMissingKey() {
        assertThrows(IllegalArgumentException.class, () ->
                BackupExecutor.extractPersistPath("some_other_key: value\n"));
    }

    @Test
    void extractPersistPathThrowsForInvalidYaml() {
        assertThrows(IllegalArgumentException.class, () ->
                BackupExecutor.extractPersistPath("just-a-string"));
    }

    @Test
    void extractPersistPathThrowsForEmptyYaml() {
        assertThrows(IllegalArgumentException.class, () ->
                BackupExecutor.extractPersistPath(""));
    }

    @Test
    void extractPersistPathThrowsForMalformedYaml() {
        assertThrows(IllegalArgumentException.class, () ->
                BackupExecutor.extractPersistPath("{: invalid yaml ["));
    }

    private static boolean isEmptyDir(Path dir) {
        try (var entries = Files.list(dir)) {
            return entries.findFirst().isEmpty();
        } catch (IOException e) {
            return true;
        }
    }
}
