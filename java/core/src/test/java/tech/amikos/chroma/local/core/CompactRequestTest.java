package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.*;

import org.junit.jupiter.api.Test;

class CompactRequestTest {

    @Test
    void compactCollectionRequestToJson() {
        CompactCollectionRequest req = new CompactCollectionRequest.Builder("my_collection")
                .tenantId("tenant_abc")
                .databaseName("db_name")
                .build();
        String json = req.toJson();
        assertTrue(json.contains("\"name\""));
        assertTrue(json.contains("\"my_collection\""));
        assertTrue(json.contains("\"tenant_id\""));
        assertTrue(json.contains("\"database_name\""));
    }

    @Test
    void compactCollectionRequestBuilderRejectsBlankName() {
        assertThrows(IllegalArgumentException.class, () ->
                new CompactCollectionRequest.Builder("").build());
    }

    @Test
    void compactCollectionRequestBuilderRejectsShortDatabaseName() {
        assertThrows(IllegalArgumentException.class, () ->
                new CompactCollectionRequest.Builder("col")
                        .databaseName("ab")
                        .build());
    }

    @Test
    void compactAllRequestToJson() {
        CompactAllRequest req = new CompactAllRequest.Builder()
                .tenantId("tenant_abc")
                .databaseName("db_name")
                .build();
        String json = req.toJson();
        assertTrue(json.contains("\"tenant_id\""));
        assertTrue(json.contains("\"database_name\""));
    }

    @Test
    void compactAllRequestToJsonOmitsNulls() {
        CompactAllRequest req = new CompactAllRequest.Builder().build();
        String json = req.toJson();
        assertFalse(json.contains("tenant_id"));
        assertFalse(json.contains("database_name"));
    }

    @Test
    void compactAllRequestBuilderRejectsShortDatabaseName() {
        assertThrows(IllegalArgumentException.class, () ->
                new CompactAllRequest.Builder()
                        .databaseName("ab")
                        .build());
    }

    @Test
    void compactCollectionRequestBuilderRejectsTooShortTenantId() {
        assertThrows(IllegalArgumentException.class, () ->
                new CompactCollectionRequest.Builder("coll").tenantId("ab").build());
    }

    @Test
    void compactAllRequestBuilderRejectsTooShortTenantId() {
        assertThrows(IllegalArgumentException.class, () ->
                new CompactAllRequest.Builder().tenantId("ab").build());
    }

    @Test
    void compactCollectionRequestBuilderRejectsNullName() {
        assertThrows(IllegalArgumentException.class, () ->
                new CompactCollectionRequest.Builder(null).build());
    }

    @Test
    void compactCollectionRequestMinimalJson() {
        CompactCollectionRequest req = new CompactCollectionRequest.Builder("col").build();
        String json = req.toJson();
        assertTrue(json.contains("\"name\""));
        assertTrue(json.contains("\"col\""));
    }
}
