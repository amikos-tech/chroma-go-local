package tech.amikos.chroma.local.core;

public final class BackupOptions {

    private final String destinationPath;
    private final boolean includeMetadata;
    private final boolean leaveStopped;
    private final boolean leaveClosed;

    private BackupOptions(Builder builder) {
        this.destinationPath = builder.destinationPath;
        this.includeMetadata = builder.includeMetadata;
        this.leaveStopped = builder.leaveStopped;
        this.leaveClosed = builder.leaveClosed;
    }

    public String destinationPath() { return destinationPath; }
    public boolean includeMetadata() { return includeMetadata; }
    public boolean leaveStopped() { return leaveStopped; }
    public boolean leaveClosed() { return leaveClosed; }

    public String toJson() {
        return JsonUtil.toJson(this);
    }

    public static class Builder {
        private final String destinationPath;
        private boolean includeMetadata;
        private boolean leaveStopped;
        private boolean leaveClosed;

        public Builder(String destinationPath) {
            this.destinationPath = destinationPath;
        }

        public Builder includeMetadata(boolean includeMetadata) {
            this.includeMetadata = includeMetadata;
            return this;
        }

        public Builder leaveStopped(boolean leaveStopped) {
            this.leaveStopped = leaveStopped;
            return this;
        }

        public Builder leaveClosed(boolean leaveClosed) {
            this.leaveClosed = leaveClosed;
            return this;
        }

        public BackupOptions build() {
            if (destinationPath == null || destinationPath.isBlank()) {
                throw new IllegalArgumentException("destination_path is required");
            }
            return new BackupOptions(this);
        }
    }
}
