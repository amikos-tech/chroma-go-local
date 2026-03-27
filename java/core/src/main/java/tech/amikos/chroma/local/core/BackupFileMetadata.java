package tech.amikos.chroma.local.core;

import java.util.Objects;

public record BackupFileMetadata(String path, long sizeBytes, String mode, String sha256, String modifiedAt) {
    public BackupFileMetadata {
        Objects.requireNonNull(path, "path");
        if (sizeBytes < 0) throw new IllegalArgumentException("sizeBytes must be non-negative");
        Objects.requireNonNull(mode, "mode");
        Objects.requireNonNull(sha256, "sha256");
        Objects.requireNonNull(modifiedAt, "modifiedAt");
    }
}
