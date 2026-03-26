package tech.amikos.chroma.local.core;

public final class WALPruneCollectionResult {
    private final String collectionId;
    private final String name;
    private final String tenantId;
    private final String databaseName;
    private final Long safeSeqCutoff;
    private final Long candidateSeqMin;
    private final Long candidateSeqMax;
    private final Long prunedSeqMin;
    private final Long prunedSeqMax;
    private final long candidateCount;
    private final long candidateBytes;
    private final long prunedCount;
    private final long prunedBytes;
    private final String error;

    WALPruneCollectionResult() {
        this.collectionId = null;
        this.name = null;
        this.tenantId = null;
        this.databaseName = null;
        this.safeSeqCutoff = null;
        this.candidateSeqMin = null;
        this.candidateSeqMax = null;
        this.prunedSeqMin = null;
        this.prunedSeqMax = null;
        this.candidateCount = 0;
        this.candidateBytes = 0;
        this.prunedCount = 0;
        this.prunedBytes = 0;
        this.error = null;
    }

    public String collectionId() { return collectionId; }
    public String name() { return name; }
    public String tenantId() { return tenantId; }
    public String databaseName() { return databaseName; }
    public Long safeSeqCutoff() { return safeSeqCutoff; }
    public Long candidateSeqMin() { return candidateSeqMin; }
    public Long candidateSeqMax() { return candidateSeqMax; }
    public Long prunedSeqMin() { return prunedSeqMin; }
    public Long prunedSeqMax() { return prunedSeqMax; }
    public long candidateCount() { return candidateCount; }
    public long candidateBytes() { return candidateBytes; }
    public long prunedCount() { return prunedCount; }
    public long prunedBytes() { return prunedBytes; }
    public String error() { return error; }
}
