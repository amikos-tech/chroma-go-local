package tech.amikos.chroma.local.core;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Objects;

public final class BackupManifest {
    private final String schemaVersion;
    private final String mode;
    private final String createdAt;
    private final String wrapperVersion;
    private final List<String> sourcePaths;
    private final String destinationPath;
    private final String snapshotPath;
    private final String manifestPath;
    private final boolean includeMetadata;
    private final int fileCount;
    private final long totalBytes;
    private final List<BackupFileMetadata> files;

    BackupManifest(String schemaVersion, String mode, String createdAt, String wrapperVersion,
                   List<String> sourcePaths, String destinationPath, String snapshotPath,
                   String manifestPath, boolean includeMetadata, int fileCount, long totalBytes,
                   List<BackupFileMetadata> files) {
        this.schemaVersion = Objects.requireNonNull(schemaVersion, "schemaVersion");
        this.mode = Objects.requireNonNull(mode, "mode");
        this.createdAt = Objects.requireNonNull(createdAt, "createdAt");
        this.wrapperVersion = Objects.requireNonNull(wrapperVersion, "wrapperVersion");
        this.sourcePaths = new ArrayList<>(Objects.requireNonNull(sourcePaths, "sourcePaths"));
        this.destinationPath = Objects.requireNonNull(destinationPath, "destinationPath");
        this.snapshotPath = Objects.requireNonNull(snapshotPath, "snapshotPath");
        this.manifestPath = Objects.requireNonNull(manifestPath, "manifestPath");
        this.includeMetadata = includeMetadata;
        if (fileCount < 0) throw new IllegalArgumentException("fileCount must be non-negative");
        this.fileCount = fileCount;
        if (totalBytes < 0) throw new IllegalArgumentException("totalBytes must be non-negative");
        this.totalBytes = totalBytes;
        this.files = files == null ? null : new ArrayList<>(files);
    }

    public String schemaVersion() { return schemaVersion; }
    public String mode() { return mode; }
    public String createdAt() { return createdAt; }
    public String wrapperVersion() { return wrapperVersion; }
    public List<String> sourcePaths() { return Collections.unmodifiableList(sourcePaths); }
    public String destinationPath() { return destinationPath; }
    public String snapshotPath() { return snapshotPath; }
    public String manifestPath() { return manifestPath; }
    public boolean includeMetadata() { return includeMetadata; }
    public int fileCount() { return fileCount; }
    public long totalBytes() { return totalBytes; }
    public List<BackupFileMetadata> files() { return files == null ? Collections.emptyList() : Collections.unmodifiableList(files); }
}
