package tech.amikos.chroma.local.core;

public final class BackupOptions {

    private final String destinationPath;
    private final boolean includeMetadata;
    private final boolean leaveInactive;

    private BackupOptions(Builder builder) {
        this.destinationPath = builder.destinationPath;
        this.includeMetadata = builder.includeMetadata;
        this.leaveInactive = builder.leaveInactive;
    }

    public String destinationPath() { return destinationPath; }
    public boolean includeMetadata() { return includeMetadata; }
    public boolean leaveInactive() { return leaveInactive; }

    public static class Builder {
        private final String destinationPath;
        private boolean includeMetadata;
        private boolean leaveInactive;

        public Builder(String destinationPath) {
            this.destinationPath = destinationPath;
        }

        public Builder includeMetadata(boolean includeMetadata) {
            this.includeMetadata = includeMetadata;
            return this;
        }

        public Builder leaveInactive(boolean leaveInactive) {
            this.leaveInactive = leaveInactive;
            return this;
        }

        public BackupOptions build() {
            if (destinationPath == null || destinationPath.isBlank()) {
                throw new IllegalArgumentException("destinationPath is required");
            }
            return new BackupOptions(this);
        }
    }
}
