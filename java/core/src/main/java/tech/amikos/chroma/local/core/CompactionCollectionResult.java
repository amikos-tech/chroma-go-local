package tech.amikos.chroma.local.core;

public final class CompactionCollectionResult {
    private final String collectionId;
    private final String name;
    private final String tenantId;
    private final String databaseName;
    private final Long pendingOpsBefore;
    private final Long pendingOpsAfter;
    private final String pendingOpsBeforeError;
    private final String pendingOpsAfterError;
    private final String error;

    CompactionCollectionResult() {
        this.collectionId = null;
        this.name = null;
        this.tenantId = null;
        this.databaseName = null;
        this.pendingOpsBefore = null;
        this.pendingOpsAfter = null;
        this.pendingOpsBeforeError = null;
        this.pendingOpsAfterError = null;
        this.error = null;
    }

    public String collectionId() { return collectionId; }
    public String name() { return name; }
    public String tenantId() { return tenantId; }
    public String databaseName() { return databaseName; }
    public Long pendingOpsBefore() { return pendingOpsBefore; }
    public Long pendingOpsAfter() { return pendingOpsAfter; }
    public String pendingOpsBeforeError() { return pendingOpsBeforeError; }
    public String pendingOpsAfterError() { return pendingOpsAfterError; }
    public String error() { return error; }
}
