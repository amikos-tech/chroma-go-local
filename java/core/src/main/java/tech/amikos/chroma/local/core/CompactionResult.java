package tech.amikos.chroma.local.core;

import java.util.List;

public final class CompactionResult {
    private final int collectionCount;
    private final long durationMs;
    private final long pendingOpsBeforeTotal;
    private final long pendingOpsAfterTotal;
    private final List<CompactionCollectionResult> collections;

    CompactionResult() {
        this.collectionCount = 0;
        this.durationMs = 0;
        this.pendingOpsBeforeTotal = 0;
        this.pendingOpsAfterTotal = 0;
        this.collections = null;
    }

    public int collectionCount() { return collectionCount; }
    public long durationMs() { return durationMs; }
    public long pendingOpsBeforeTotal() { return pendingOpsBeforeTotal; }
    public long pendingOpsAfterTotal() { return pendingOpsAfterTotal; }
    public List<CompactionCollectionResult> collections() { return collections; }
}
