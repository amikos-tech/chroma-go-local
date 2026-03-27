package tech.amikos.chroma.local.core;

import java.util.Objects;

public final class BackupResult<S> {
    private final BackupManifest manifest;
    private final S session;

    public BackupResult(BackupManifest manifest, S session) {
        this.manifest = Objects.requireNonNull(manifest, "manifest");
        this.session = session;
    }

    public BackupManifest manifest() { return manifest; }

    /** Returns the new session after backup, or null if left closed/stopped. */
    public S session() { return session; }
}
