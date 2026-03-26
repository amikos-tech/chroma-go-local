package tech.amikos.chroma.local.core;

public final class RebuildOptions {

    private final String name;
    private final String tenantId;
    private final String databaseName;
    private final boolean precheck;
    private final boolean keepBackup;

    private RebuildOptions(Builder builder) {
        this.name = builder.name;
        this.tenantId = builder.tenantId;
        this.databaseName = builder.databaseName;
        this.precheck = builder.precheck;
        this.keepBackup = builder.keepBackup;
    }

    public String name() { return name; }
    public String tenantId() { return tenantId; }
    public String databaseName() { return databaseName; }
    public boolean precheck() { return precheck; }
    public boolean keepBackup() { return keepBackup; }

    public String toJson() {
        return JsonUtil.toJson(this);
    }

    public static RebuildOptions defaults(String name) {
        return new Builder(name).build();
    }

    public static class Builder {
        private final String name;
        private String tenantId;
        private String databaseName;
        private boolean precheck;
        private boolean keepBackup = true;

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

        public Builder precheck(boolean precheck) {
            this.precheck = precheck;
            return this;
        }

        public Builder keepBackup(boolean keepBackup) {
            this.keepBackup = keepBackup;
            return this;
        }

        public RebuildOptions build() {
            Validation.validateRequiredName(name);
            Validation.validateTenantId(tenantId);
            Validation.validateDatabaseName(databaseName);
            return new RebuildOptions(this);
        }
    }
}
