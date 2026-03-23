package tech.amikos.chroma.local.core;

final class Validation {

    private Validation() {}

    static void validateTenantId(String tenantId) {
        if (tenantId != null && tenantId.length() < 3) {
            throw new IllegalArgumentException("tenant_id must be at least 3 characters");
        }
    }

    static void validateDatabaseName(String databaseName) {
        if (databaseName != null && databaseName.length() < 3) {
            throw new IllegalArgumentException("database_name must be at least 3 characters");
        }
    }
}
