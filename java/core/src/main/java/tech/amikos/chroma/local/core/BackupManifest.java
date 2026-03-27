package tech.amikos.chroma.local.core;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

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

    BackupManifest() {
        this.schemaVersion = null;
        this.mode = null;
        this.createdAt = null;
        this.wrapperVersion = null;
        this.sourcePaths = null;
        this.destinationPath = null;
        this.snapshotPath = null;
        this.manifestPath = null;
        this.includeMetadata = false;
        this.fileCount = 0;
        this.totalBytes = 0;
        this.files = null;
    }

    BackupManifest(String schemaVersion, String mode, String createdAt, String wrapperVersion,
                   List<String> sourcePaths, String destinationPath, String snapshotPath,
                   String manifestPath, boolean includeMetadata, int fileCount, long totalBytes,
                   List<BackupFileMetadata> files) {
        this.schemaVersion = schemaVersion;
        this.mode = mode;
        this.createdAt = createdAt;
        this.wrapperVersion = wrapperVersion;
        this.sourcePaths = sourcePaths == null ? null : new ArrayList<>(sourcePaths);
        this.destinationPath = destinationPath;
        this.snapshotPath = snapshotPath;
        this.manifestPath = manifestPath;
        this.includeMetadata = includeMetadata;
        this.fileCount = fileCount;
        this.totalBytes = totalBytes;
        this.files = files == null ? null : new ArrayList<>(files);
    }

    public String schemaVersion() { return schemaVersion; }
    public String mode() { return mode; }
    public String createdAt() { return createdAt; }
    public String wrapperVersion() { return wrapperVersion; }
    public List<String> sourcePaths() { return sourcePaths == null ? Collections.emptyList() : Collections.unmodifiableList(sourcePaths); }
    public String destinationPath() { return destinationPath; }
    public String snapshotPath() { return snapshotPath; }
    public String manifestPath() { return manifestPath; }
    public boolean includeMetadata() { return includeMetadata; }
    public int fileCount() { return fileCount; }
    public long totalBytes() { return totalBytes; }
    public List<BackupFileMetadata> files() { return files == null ? Collections.emptyList() : Collections.unmodifiableList(files); }
}
