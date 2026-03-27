package tech.amikos.chroma.local.core;

import com.google.gson.FieldNamingPolicy;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.nio.file.FileVisitResult;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.SimpleFileVisitor;
import java.nio.file.StandardOpenOption;
import java.nio.file.attribute.BasicFileAttributes;
import java.nio.file.attribute.PosixFilePermissions;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Instant;
import java.util.ArrayList;
import java.util.HexFormat;
import java.util.List;
import java.util.Objects;
import java.util.function.Supplier;

public final class BackupExecutor {

    static final String MANIFEST_FILENAME = "backup_manifest.json";
    static final String SNAPSHOT_DIRNAME = "persist";
    static final String SCHEMA_VERSION = "v1";

    private static final Gson PRETTY_GSON = new GsonBuilder()
            .setFieldNamingPolicy(FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES)
            .setPrettyPrinting()
            .create();

    private BackupExecutor() {}

    static final class PreValidationFailure extends RuntimeException {
        PreValidationFailure(RuntimeException cause) { super(cause); }
    }

    public static <S> BackupResult<S> execute(BackupMode mode, String persistPath, String wrapperVersion,
                                              BackupOptions options, Runnable closeAction,
                                              Supplier<S> restartAction) {
        Objects.requireNonNull(mode, "mode");
        Objects.requireNonNull(persistPath, "persistPath");
        Objects.requireNonNull(wrapperVersion, "wrapperVersion");
        Objects.requireNonNull(options, "options");
        Objects.requireNonNull(closeAction, "closeAction");
        Objects.requireNonNull(restartAction, "restartAction");

        Path dest;
        Path source;
        try {
            dest = Path.of(options.destinationPath()).toAbsolutePath().normalize();
            source = Path.of(persistPath).toAbsolutePath().normalize();

            if (isWithinPath(dest, source)) {
                throw new IllegalArgumentException(
                        "destination path cannot be inside source persist path: " + dest);
            }

            ensureEmptyDir(dest);
        } catch (RuntimeException e) {
            throw new PreValidationFailure(e);
        }

        try {
            closeAction.run();
        } catch (RuntimeException e) {
            IOException cleanupErr = deleteDirectoryQuietly(dest);
            if (cleanupErr != null) e.addSuppressed(cleanupErr);
            throw e;
        }

        IOException backupError = null;
        BackupManifest manifest = null;
        try {
            Path snapshotPath = dest.resolve(SNAPSHOT_DIRNAME);
            CopyResult copyResult;
            if (Files.isDirectory(source)) {
                copyResult = copyDirectory(source, snapshotPath, options.includeMetadata());
            } else {
                throw new IOException(
                        "backup source path does not exist or is not a directory: " + source);
            }

            Path manifestPath = dest.resolve(MANIFEST_FILENAME);
            manifest = new BackupManifest(
                    SCHEMA_VERSION,
                    mode.toWire(),
                    Instant.now().toString(),
                    wrapperVersion,
                    List.of(source.toString()),
                    dest.toString(),
                    snapshotPath.toString(),
                    manifestPath.toString(),
                    options.includeMetadata(),
                    copyResult.fileCount,
                    copyResult.totalBytes,
                    options.includeMetadata() ? copyResult.files : null);

            writeManifest(manifestPath, manifest);
        } catch (IOException e) {
            backupError = e;
        }

        if (options.leaveInactive()) {
            if (backupError != null) {
                IOException cleanupErr = deleteDirectoryQuietly(dest);
                if (cleanupErr != null) backupError.addSuppressed(cleanupErr);
                throw toRuntimeException(backupError);
            }
            return new BackupResult<>(manifest, null);
        }

        S newSession;
        try {
            newSession = restartAction.get();
        } catch (RuntimeException restartErr) {
            if (backupError != null) {
                IOException cleanupErr = deleteDirectoryQuietly(dest);
                if (cleanupErr != null) backupError.addSuppressed(cleanupErr);
            }
            ChromaException combined = new ChromaException(
                    "backup failed and restart also failed: " + restartErr.getMessage(), restartErr);
            if (backupError != null) {
                combined.addSuppressed(backupError);
            }
            throw combined;
        }

        if (backupError != null) {
            IOException cleanupErr = deleteDirectoryQuietly(dest);
            if (cleanupErr != null) backupError.addSuppressed(cleanupErr);
            throw toRuntimeException(backupError);
        }

        return new BackupResult<>(manifest, newSession);
    }

    private static RuntimeException toRuntimeException(Throwable t) {
        if (t instanceof RuntimeException re) return re;
        return new ChromaException("backup failed: " + t.getMessage(), t);
    }

    private static boolean isWithinPath(Path path, Path parent) {
        Path normalized = path.toAbsolutePath().normalize();
        Path normalizedParent = parent.toAbsolutePath().normalize();
        return normalized.startsWith(normalizedParent);
    }

    private static void ensureEmptyDir(Path path) {
        try {
            if (Files.exists(path)) {
                if (!Files.isDirectory(path)) {
                    throw new IllegalArgumentException("destination path must be a directory: " + path);
                }
                try (var entries = Files.list(path)) {
                    if (entries.findFirst().isPresent()) {
                        throw new IllegalArgumentException("destination path must be empty: " + path);
                    }
                }
            } else {
                Files.createDirectories(path);
            }
        } catch (IOException e) {
            throw new ChromaException("failed to prepare destination directory: " + e.getMessage(), e);
        }
    }

    private static CopyResult copyDirectory(Path source, Path destination, boolean includeMetadata)
            throws IOException {
        Files.createDirectories(destination);
        List<BackupFileMetadata> files = new ArrayList<>();
        int[] fileCount = {0};
        long[] totalBytes = {0};

        Files.walkFileTree(source, new SimpleFileVisitor<>() {
            @Override
            public FileVisitResult preVisitDirectory(Path dir, BasicFileAttributes attrs)
                    throws IOException {
                Path target = destination.resolve(source.relativize(dir));
                Files.createDirectories(target);
                return FileVisitResult.CONTINUE;
            }

            @Override
            public FileVisitResult visitFile(Path file, BasicFileAttributes attrs)
                    throws IOException {
                if (Files.isSymbolicLink(file)) {
                    throw new IOException("backup does not support symbolic links: " + file);
                }
                Path target = destination.resolve(source.relativize(file));
                String sha256 = copyFileWithHash(file, target);
                fileCount[0]++;
                totalBytes[0] += attrs.size();
                if (includeMetadata) {
                    String relPath = source.relativize(file).toString().replace('\\', '/');
                    String mode = readFileMode(file);
                    String modifiedAt = attrs.lastModifiedTime().toInstant().toString();
                    files.add(new BackupFileMetadata(relPath, attrs.size(), mode, sha256, modifiedAt));
                }
                return FileVisitResult.CONTINUE;
            }
        });
        return new CopyResult(fileCount[0], totalBytes[0], files);
    }

    private static String copyFileWithHash(Path source, Path destination) throws IOException {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] buffer = new byte[8192];
            try (InputStream in = Files.newInputStream(source);
                 OutputStream out = Files.newOutputStream(destination,
                         StandardOpenOption.CREATE_NEW, StandardOpenOption.WRITE)) {
                int read;
                while ((read = in.read(buffer)) != -1) {
                    out.write(buffer, 0, read);
                    digest.update(buffer, 0, read);
                }
            }
            Files.setLastModifiedTime(destination, Files.getLastModifiedTime(source));
            return HexFormat.of().formatHex(digest.digest());
        } catch (NoSuchAlgorithmException e) {
            throw new IOException("SHA-256 not available", e);
        }
    }

    private static String readFileMode(Path file) throws IOException {
        try {
            var perms = Files.getPosixFilePermissions(file);
            String permStr = PosixFilePermissions.toString(perms);
            int mode = 0;
            if (permStr.charAt(0) == 'r') mode |= 0400;
            if (permStr.charAt(1) == 'w') mode |= 0200;
            if (permStr.charAt(2) == 'x') mode |= 0100;
            if (permStr.charAt(3) == 'r') mode |= 040;
            if (permStr.charAt(4) == 'w') mode |= 020;
            if (permStr.charAt(5) == 'x') mode |= 010;
            if (permStr.charAt(6) == 'r') mode |= 04;
            if (permStr.charAt(7) == 'w') mode |= 02;
            if (permStr.charAt(8) == 'x') mode |= 01;
            return String.format("0%o", mode);
        } catch (UnsupportedOperationException e) {
            return "0644";
        }
    }

    static IOException deleteDirectoryQuietly(Path dir) {
        try {
            if (!Files.exists(dir)) return null;
            Files.walkFileTree(dir, new SimpleFileVisitor<>() {
                @Override
                public FileVisitResult visitFile(Path file, BasicFileAttributes attrs) throws IOException {
                    Files.deleteIfExists(file);
                    return FileVisitResult.CONTINUE;
                }

                @Override
                public FileVisitResult postVisitDirectory(Path d, IOException exc) throws IOException {
                    if (exc != null) throw exc;
                    Files.deleteIfExists(d);
                    return FileVisitResult.CONTINUE;
                }
            });
            return null;
        } catch (IOException e) {
            return e;
        }
    }

    private static void writeManifest(Path manifestPath, BackupManifest manifest) throws IOException {
        String json = PRETTY_GSON.toJson(manifest) + "\n";
        Files.writeString(manifestPath, json, StandardOpenOption.CREATE_NEW);
    }

    private record CopyResult(int fileCount, long totalBytes, List<BackupFileMetadata> files) {}
}
