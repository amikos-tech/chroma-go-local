package tech.amikos.chroma.local.core;

public final class WALPruneOptions {

    private final String name;
    private final String tenantId;
    private final String databaseName;
    private final boolean dryRun;
    private final boolean vacuum;
    private final Long maxAgeSeconds;
    private final Long maxBytes;
    private final Long watermarkHighBytes;
    private final Long watermarkLowBytes;

    private WALPruneOptions(Builder builder) {
        this.name = builder.name;
        this.tenantId = builder.tenantId;
        this.databaseName = builder.databaseName;
        this.dryRun = builder.dryRun;
        this.vacuum = builder.vacuum;
        this.maxAgeSeconds = builder.maxAgeSeconds;
        this.maxBytes = builder.maxBytes;
        this.watermarkHighBytes = builder.watermarkHighBytes;
        this.watermarkLowBytes = builder.watermarkLowBytes;
    }

    public String name() { return name; }
    public String tenantId() { return tenantId; }
    public String databaseName() { return databaseName; }
    public boolean dryRun() { return dryRun; }
    public boolean vacuum() { return vacuum; }
    public Long maxAgeSeconds() { return maxAgeSeconds; }
    public Long maxBytes() { return maxBytes; }
    public Long watermarkHighBytes() { return watermarkHighBytes; }
    public Long watermarkLowBytes() { return watermarkLowBytes; }

    public String toJson() {
        return JsonUtil.toJson(this);
    }

    public static WALPruneOptions defaults(String name) {
        return new Builder(name).dryRun(true).build();
    }

    public static class Builder {
        private final String name;
        private String tenantId;
        private String databaseName;
        private boolean dryRun;
        private boolean vacuum;
        private Long maxAgeSeconds;
        private Long maxBytes;
        private Long watermarkHighBytes;
        private Long watermarkLowBytes;

        public Builder(String name) {
            this.name = name;
        }

        public Builder() {
            this.name = null;
        }

        public Builder tenantId(String tenantId) {
            this.tenantId = tenantId;
            return this;
        }

        public Builder databaseName(String databaseName) {
            this.databaseName = databaseName;
            return this;
        }

        public Builder dryRun(boolean dryRun) {
            this.dryRun = dryRun;
            return this;
        }

        public Builder vacuum(boolean vacuum) {
            this.vacuum = vacuum;
            return this;
        }

        public Builder maxAgeSeconds(long maxAgeSeconds) {
            this.maxAgeSeconds = maxAgeSeconds;
            return this;
        }

        public Builder maxBytes(long maxBytes) {
            this.maxBytes = maxBytes;
            return this;
        }

        public Builder watermark(long highBytes, long lowBytes) {
            this.watermarkHighBytes = highBytes;
            this.watermarkLowBytes = lowBytes;
            return this;
        }

        public WALPruneOptions build() {
            if (name != null && name.isBlank()) {
                throw new IllegalArgumentException("name must not be blank");
            }
            Validation.validateDatabaseName(databaseName);
            Validation.validateTenantId(tenantId);
            if (maxAgeSeconds != null && maxAgeSeconds <= 0) {
                throw new IllegalArgumentException("max_age_seconds must be greater than 0");
            }
            if (maxBytes != null && maxBytes <= 0) {
                throw new IllegalArgumentException("max_bytes must be greater than 0");
            }
            if ((watermarkHighBytes == null) != (watermarkLowBytes == null)) {
                throw new IllegalArgumentException("wal prune watermark requires both high and low bytes");
            }
            if (watermarkHighBytes != null && watermarkLowBytes != null
                    && watermarkLowBytes > watermarkHighBytes) {
                throw new IllegalArgumentException(
                        "wal prune watermark low bytes must be less than or equal to high bytes");
            }
            boolean hasPolicy = maxAgeSeconds != null || maxBytes != null
                    || (watermarkHighBytes != null && watermarkLowBytes != null);
            if (!dryRun && !hasPolicy) {
                throw new IllegalArgumentException(
                        "at least one WAL prune policy is required unless dry-run is enabled");
            }
            return new WALPruneOptions(this);
        }
    }
}
