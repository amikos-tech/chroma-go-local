package tech.amikos.chroma.local.core;

import java.util.Collections;
import java.util.List;

public final class RebuildCollectionResult {
    private final String collectionId;
    private final String name;
    private final String tenantId;
    private final String databaseName;
    private final boolean precheck;
    private final boolean wouldRebuild;
    private final boolean rebuilt;
    private final long recordsScanned;
    private final long vectorsReindexed;
    private final long durationMs;
    private final String backupPath;
    private final List<String> warnings;

    RebuildCollectionResult() {
        this.collectionId = null;
        this.name = null;
        this.tenantId = null;
        this.databaseName = null;
        this.precheck = false;
        this.wouldRebuild = false;
        this.rebuilt = false;
        this.recordsScanned = 0;
        this.vectorsReindexed = 0;
        this.durationMs = 0;
        this.backupPath = null;
        this.warnings = null;
    }

    public String collectionId() { return collectionId; }
    public String name() { return name; }
    public String tenantId() { return tenantId; }
    public String databaseName() { return databaseName; }
    public boolean precheck() { return precheck; }
    public boolean wouldRebuild() { return wouldRebuild; }
    public boolean rebuilt() { return rebuilt; }
    public long recordsScanned() { return recordsScanned; }
    public long vectorsReindexed() { return vectorsReindexed; }
    public long durationMs() { return durationMs; }
    public String backupPath() { return backupPath; }
    public List<String> warnings() { return warnings == null ? Collections.emptyList() : Collections.unmodifiableList(warnings); }
}
