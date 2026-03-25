package tech.amikos.chroma.local.core;

public final class CompactCollectionRequest {

    private final String name;
    private final String tenantId;
    private final String databaseName;

    private CompactCollectionRequest(Builder builder) {
        this.name = builder.name;
        this.tenantId = builder.tenantId;
        this.databaseName = builder.databaseName;
    }

    public String name() { return name; }
    public String tenantId() { return tenantId; }
    public String databaseName() { return databaseName; }

    public String toJson() {
        return JsonUtil.toJson(this);
    }

    public static class Builder {
        private final String name;
        private String tenantId;
        private String databaseName;

        public Builder(String name) {
            this.name = name;
        }

        public Builder tenantId(String tenantId) {
            this.tenantId = tenantId;
            return this;
        }

        public Builder databaseName(String databaseName) {
            this.databaseName = databaseName;
            return this;
        }

        public CompactCollectionRequest build() {
            Validation.validateRequiredName(name);
            Validation.validateTenantId(tenantId);
            Validation.validateDatabaseName(databaseName);
            return new CompactCollectionRequest(this);
        }
    }
}
