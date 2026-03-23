package tech.amikos.chroma.local.core;

import java.util.Collections;
import java.util.List;

public final class WALPruneResult {
    private final int collectionCount;
    private final long durationMs;
    private final boolean dryRun;
    private final boolean vacuumRequested;
    private final boolean vacuumExecuted;
    private final String warning;
    private final long candidateCountTotal;
    private final long candidateBytesTotal;
    private final long prunedCountTotal;
    private final long prunedBytesTotal;
    private final List<WALPruneCollectionResult> collections;

    WALPruneResult() {
        this.collectionCount = 0;
        this.durationMs = 0;
        this.dryRun = false;
        this.vacuumRequested = false;
        this.vacuumExecuted = false;
        this.warning = null;
        this.candidateCountTotal = 0;
        this.candidateBytesTotal = 0;
        this.prunedCountTotal = 0;
        this.prunedBytesTotal = 0;
        this.collections = null;
    }

    public int collectionCount() { return collectionCount; }
    public long durationMs() { return durationMs; }
    public boolean dryRun() { return dryRun; }
    public boolean vacuumRequested() { return vacuumRequested; }
    public boolean vacuumExecuted() { return vacuumExecuted; }
    public String warning() { return warning; }
    public long candidateCountTotal() { return candidateCountTotal; }
    public long candidateBytesTotal() { return candidateBytesTotal; }
    public long prunedCountTotal() { return prunedCountTotal; }
    public long prunedBytesTotal() { return prunedBytesTotal; }
    public List<WALPruneCollectionResult> collections() { return collections == null ? null : Collections.unmodifiableList(collections); }
}
