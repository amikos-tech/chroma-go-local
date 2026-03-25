package tech.amikos.chroma.local.core;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class RebuildCollectionResultTest {

    @Test
    void deserializesFullJson() {
        String json = """
                {
                  "collection_id": "abc-123",
                  "name": "my_collection",
                  "tenant_id": "tenant_1",
                  "database_name": "default_database",
                  "precheck": false,
                  "would_rebuild": true,
                  "rebuilt": true,
                  "records_scanned": 5000,
                  "vectors_reindexed": 4500,
                  "duration_ms": 1234,
                  "backup_path": "/tmp/backup",
                  "warnings": ["warn1", "warn2"]
                }
                """;
        RebuildCollectionResult r = JsonUtil.fromJson(json, RebuildCollectionResult.class);
        assertEquals("abc-123", r.collectionId());
        assertEquals("my_collection", r.name());
        assertEquals("tenant_1", r.tenantId());
        assertEquals("default_database", r.databaseName());
        assertFalse(r.precheck());
        assertTrue(r.wouldRebuild());
        assertTrue(r.rebuilt());
        assertEquals(5000L, r.recordsScanned());
        assertEquals(4500L, r.vectorsReindexed());
        assertEquals(1234L, r.durationMs());
        assertEquals("/tmp/backup", r.backupPath());
        assertEquals(2, r.warnings().size());
        assertEquals("warn1", r.warnings().get(0));
    }

    @Test
    void deserializesJsonWithMissingOptionalFields() {
        String json = """
                {
                  "collection_id": "abc-123",
                  "name": "my_collection",
                  "tenant_id": "t1",
                  "database_name": "db1",
                  "precheck": true,
                  "would_rebuild": false,
                  "rebuilt": false,
                  "records_scanned": 100,
                  "vectors_reindexed": 0,
                  "duration_ms": 50
                }
                """;
        RebuildCollectionResult r = JsonUtil.fromJson(json, RebuildCollectionResult.class);
        assertEquals("abc-123", r.collectionId());
        assertTrue(r.precheck());
        assertFalse(r.wouldRebuild());
        assertEquals(100L, r.recordsScanned());
        assertNull(r.backupPath());
        assertTrue(r.warnings().isEmpty());
    }

    @Test
    void collectionIdMapsFromSnakeCase() {
        String json = """
                {"collection_id": "id-42", "name": "n", "tenant_id": "t", "database_name": "d",
                 "precheck": false, "would_rebuild": false, "rebuilt": false,
                 "records_scanned": 0, "vectors_reindexed": 0, "duration_ms": 0}
                """;
        RebuildCollectionResult r = JsonUtil.fromJson(json, RebuildCollectionResult.class);
        assertEquals("id-42", r.collectionId());
    }

    @Test
    void warnings_returnsUnmodifiableList() {
        String json = """
                {
                  "collection_id": "abc-123", "name": "n", "tenant_id": "t", "database_name": "d",
                  "precheck": false, "would_rebuild": false, "rebuilt": false,
                  "records_scanned": 0, "vectors_reindexed": 0, "duration_ms": 0,
                  "warnings": ["warn1"]
                }
                """;
        RebuildCollectionResult r = JsonUtil.fromJson(json, RebuildCollectionResult.class);
        assertThrows(UnsupportedOperationException.class, () -> r.warnings().add("new"));
    }

    @Test
    void recordsScannedReturnsLong() {
        String json = """
                {"collection_id": "x", "name": "n", "tenant_id": "t", "database_name": "d",
                 "precheck": false, "would_rebuild": false, "rebuilt": false,
                 "records_scanned": 9999999999, "vectors_reindexed": 0, "duration_ms": 0}
                """;
        RebuildCollectionResult r = JsonUtil.fromJson(json, RebuildCollectionResult.class);
        assertEquals(9999999999L, r.recordsScanned());
    }
}
