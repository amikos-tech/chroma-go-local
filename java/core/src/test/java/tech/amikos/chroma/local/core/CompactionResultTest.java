package tech.amikos.chroma.local.core;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class CompactionResultTest {

    @Test
    void deserializesFullCompactionResult() {
        String json = """
                {
                  "collection_count": 2,
                  "duration_ms": 5000,
                  "pending_ops_before_total": 100,
                  "pending_ops_after_total": 10,
                  "collections": [
                    {
                      "collection_id": "coll-1",
                      "name": "first",
                      "tenant_id": "t1",
                      "database_name": "db1",
                      "pending_ops_before": 50,
                      "pending_ops_after": 5,
                      "pending_ops_before_error": "",
                      "pending_ops_after_error": "",
                      "error": ""
                    },
                    {
                      "collection_id": "coll-2",
                      "name": "second",
                      "tenant_id": "t1",
                      "database_name": "db1",
                      "pending_ops_before": 50,
                      "pending_ops_after": 5
                    }
                  ]
                }
                """;
        CompactionResult r = JsonUtil.fromJson(json, CompactionResult.class);
        assertEquals(2, r.collectionCount());
        assertEquals(5000L, r.durationMs());
        assertEquals(100L, r.pendingOpsBeforeTotal());
        assertEquals(10L, r.pendingOpsAfterTotal());
        assertNotNull(r.collections());
        assertEquals(2, r.collections().size());

        CompactionCollectionResult c1 = r.collections().get(0);
        assertEquals("coll-1", c1.collectionId());
        assertEquals("first", c1.name());
        assertEquals(50L, c1.pendingOpsBefore());
        assertEquals(5L, c1.pendingOpsAfter());
    }

    @Test
    void optionalLongFieldsAreNullWhenAbsent() {
        String json = """
                {
                  "collection_id": "coll-1",
                  "name": "first",
                  "tenant_id": "t1",
                  "database_name": "db1",
                  "pending_ops_before_error": "not available",
                  "pending_ops_after_error": "not available"
                }
                """;
        CompactionCollectionResult c = JsonUtil.fromJson(json, CompactionCollectionResult.class);
        assertNull(c.pendingOpsBefore());
        assertNull(c.pendingOpsAfter());
        assertEquals("not available", c.pendingOpsBeforeError());
    }

    @Test
    void optionalLongFieldsHaveCorrectValueWhenPresent() {
        String json = """
                {
                  "collection_id": "coll-1",
                  "name": "first",
                  "tenant_id": "t1",
                  "database_name": "db1",
                  "pending_ops_before": 42,
                  "pending_ops_after": 7
                }
                """;
        CompactionCollectionResult c = JsonUtil.fromJson(json, CompactionCollectionResult.class);
        assertEquals(42L, c.pendingOpsBefore());
        assertEquals(7L, c.pendingOpsAfter());
    }

    @Test
    void collections_returnsUnmodifiableList() {
        String json = """
                {
                  "collection_count": 1,
                  "duration_ms": 0,
                  "pending_ops_before_total": 0,
                  "pending_ops_after_total": 0,
                  "collections": [
                    {
                      "collection_id": "coll-1", "name": "first",
                      "tenant_id": "t1", "database_name": "db1"
                    }
                  ]
                }
                """;
        CompactionResult r = JsonUtil.fromJson(json, CompactionResult.class);
        assertThrows(UnsupportedOperationException.class, () -> r.collections().add(null));
    }

    @Test
    void collectionCountReturnsInt() {
        String json = """
                {
                  "collection_count": 3,
                  "duration_ms": 0,
                  "pending_ops_before_total": 0,
                  "pending_ops_after_total": 0,
                  "collections": []
                }
                """;
        CompactionResult r = JsonUtil.fromJson(json, CompactionResult.class);
        assertEquals(3, r.collectionCount());
    }
}
