package tech.amikos.chroma.local.core;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class WALPruneResultTest {

    @Test
    void deserializesFullResult() {
        String json = """
                {
                  "collection_count": 1,
                  "duration_ms": 800,
                  "dry_run": true,
                  "vacuum_requested": false,
                  "vacuum_executed": false,
                  "warnings": ["dry run only"],
                  "candidate_count_total": 500,
                  "candidate_bytes_total": 102400,
                  "pruned_count_total": 0,
                  "pruned_bytes_total": 0,
                  "collections": [
                    {
                      "collection_id": "wal-1",
                      "name": "events",
                      "tenant_id": "t1",
                      "database_name": "db1",
                      "safe_seq_cutoff": 100,
                      "candidate_seq_min": 1,
                      "candidate_seq_max": 90,
                      "pruned_seq_min": 1,
                      "pruned_seq_max": 50,
                      "candidate_count": 500,
                      "candidate_bytes": 102400,
                      "pruned_count": 0,
                      "pruned_bytes": 0
                    }
                  ]
                }
                """;
        WALPruneResult r = JsonUtil.fromJson(json, WALPruneResult.class);
        assertEquals(1, r.collectionCount());
        assertEquals(800L, r.durationMs());
        assertTrue(r.dryRun());
        assertFalse(r.vacuumRequested());
        assertFalse(r.vacuumExecuted());
        assertEquals(1, r.warnings().size());
        assertEquals("dry run only", r.warnings().get(0));
        assertEquals(500L, r.candidateCountTotal());
        assertEquals(102400L, r.candidateBytesTotal());
        assertEquals(0L, r.prunedCountTotal());
        assertEquals(0L, r.prunedBytesTotal());
        assertNotNull(r.collections());
        assertEquals(1, r.collections().size());

        WALPruneCollectionResult c = r.collections().get(0);
        assertEquals("wal-1", c.collectionId());
        assertEquals("events", c.name());
        assertEquals(100L, c.safeSeqCutoff());
        assertEquals(1L, c.candidateSeqMin());
        assertEquals(90L, c.candidateSeqMax());
        assertEquals(1L, c.prunedSeqMin());
        assertEquals(50L, c.prunedSeqMax());
        assertEquals(500L, c.candidateCount());
    }

    @Test
    void optionalLongFieldsAreNullWhenAbsent() {
        String json = """
                {
                  "collection_id": "wal-1",
                  "name": "events",
                  "tenant_id": "t1",
                  "database_name": "db1",
                  "candidate_count": 0,
                  "candidate_bytes": 0,
                  "pruned_count": 0,
                  "pruned_bytes": 0
                }
                """;
        WALPruneCollectionResult c = JsonUtil.fromJson(json, WALPruneCollectionResult.class);
        assertNull(c.safeSeqCutoff());
        assertNull(c.candidateSeqMin());
        assertNull(c.candidateSeqMax());
        assertNull(c.prunedSeqMin());
        assertNull(c.prunedSeqMax());
    }

    @Test
    void collections_returnsUnmodifiableList() {
        String json = """
                {
                  "collection_count": 1, "duration_ms": 0,
                  "dry_run": false, "vacuum_requested": false, "vacuum_executed": false,
                  "candidate_count_total": 0, "candidate_bytes_total": 0,
                  "pruned_count_total": 0, "pruned_bytes_total": 0,
                  "collections": [
                    {
                      "collection_id": "wal-1", "name": "events",
                      "tenant_id": "t1", "database_name": "db1",
                      "candidate_count": 0, "candidate_bytes": 0,
                      "pruned_count": 0, "pruned_bytes": 0
                    }
                  ]
                }
                """;
        WALPruneResult r = JsonUtil.fromJson(json, WALPruneResult.class);
        assertThrows(UnsupportedOperationException.class, () -> r.collections().add(null));
    }

    @Test
    void dryRunMapsFromSnakeCase() {
        String json = """
                {
                  "collection_count": 0, "duration_ms": 0,
                  "dry_run": true, "vacuum_requested": true, "vacuum_executed": false,
                  "candidate_count_total": 0, "candidate_bytes_total": 0,
                  "pruned_count_total": 0, "pruned_bytes_total": 0,
                  "collections": []
                }
                """;
        WALPruneResult r = JsonUtil.fromJson(json, WALPruneResult.class);
        assertTrue(r.dryRun());
        assertTrue(r.vacuumRequested());
        assertFalse(r.vacuumExecuted());
    }
}
