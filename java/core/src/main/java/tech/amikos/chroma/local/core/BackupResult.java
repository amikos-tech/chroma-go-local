package tech.amikos.chroma.local.core;

import java.util.Objects;

/** @param session the new session, or null when backup was configured to leave the system inactive */
public record BackupResult<S>(BackupManifest manifest, S session) {
    public BackupResult {
        Objects.requireNonNull(manifest, "manifest");
    }
}
