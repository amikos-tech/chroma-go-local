package tech.amikos.chroma.local.core;

public final class BackupFileMetadata {
    private final String path;
    private final long sizeBytes;
    private final String mode;
    private final String sha256;
    private final String modifiedAt;

    BackupFileMetadata() {
        this.path = null;
        this.sizeBytes = 0;
        this.mode = null;
        this.sha256 = null;
        this.modifiedAt = null;
    }

    public String path() { return path; }
    public long sizeBytes() { return sizeBytes; }
    public String mode() { return mode; }
    public String sha256() { return sha256; }
    public String modifiedAt() { return modifiedAt; }
}
