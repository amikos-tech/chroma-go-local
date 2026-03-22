package tech.amikos.chroma.local.core;

public final class CompactAllRequest {

    private final String tenantId;
    private final String databaseName;

    private CompactAllRequest(Builder builder) {
        this.tenantId = builder.tenantId;
        this.databaseName = builder.databaseName;
    }

    public String tenantId() { return tenantId; }
    public String databaseName() { return databaseName; }

    public String toJson() {
        return JsonUtil.toJson(this);
    }

    public static class Builder {
        private String tenantId;
        private String databaseName;

        public Builder() {}

        public Builder tenantId(String tenantId) {
            this.tenantId = tenantId;
            return this;
        }

        public Builder databaseName(String databaseName) {
            this.databaseName = databaseName;
            return this;
        }

        public CompactAllRequest build() {
            if (databaseName != null && databaseName.length() < 3) {
                throw new IllegalArgumentException("database_name must be at least 3 characters");
            }
            return new CompactAllRequest(this);
        }
    }
}
