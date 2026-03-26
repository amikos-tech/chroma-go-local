package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.*;

import org.junit.jupiter.api.Test;

class RebuildOptionsTest {

    @Test
    void defaults() {
        RebuildOptions opts = RebuildOptions.defaults("my_collection");
        assertEquals("my_collection", opts.name());
        assertTrue(opts.keepBackup());
        assertFalse(opts.precheck());
        assertNull(opts.tenantId());
        assertNull(opts.databaseName());
    }

    @Test
    void toJsonMatchesGoFormat() {
        RebuildOptions opts = new RebuildOptions.Builder("test_col")
                .tenantId("tenant_abc")
                .databaseName("db_name")
                .precheck(true)
                .keepBackup(false)
                .build();
        String json = opts.toJson();
        assertTrue(json.contains("\"name\""));
        assertTrue(json.contains("\"test_col\""));
        assertTrue(json.contains("\"keep_backup\""));
        assertTrue(json.contains("\"tenant_id\""));
        assertTrue(json.contains("\"tenant_abc\""));
        assertTrue(json.contains("\"database_name\""));
        assertTrue(json.contains("\"db_name\""));
        assertTrue(json.contains("\"precheck\""));
    }

    @Test
    void toJsonDefaultsKeepBackupTrue() {
        RebuildOptions opts = RebuildOptions.defaults("col");
        String json = opts.toJson();
        assertTrue(json.contains("\"keep_backup\":true") || json.contains("\"keep_backup\": true"));
    }

    @Test
    void builderRejectsBlankName() {
        assertThrows(IllegalArgumentException.class, () ->
                new RebuildOptions.Builder("  ").build());
    }

    @Test
    void builderRejectsNullName() {
        assertThrows(IllegalArgumentException.class, () ->
                new RebuildOptions.Builder(null).build());
    }

    @Test
    void builderRejectsShortDatabaseName() {
        assertThrows(IllegalArgumentException.class, () ->
                new RebuildOptions.Builder("col").databaseName("ab").build());
    }

    @Test
    void builderRejectsShortTenantId() {
        assertThrows(IllegalArgumentException.class, () ->
                new RebuildOptions.Builder("col").tenantId("ab").build());
    }

    @Test
    void builderAcceptsValidDatabaseName() {
        RebuildOptions opts = new RebuildOptions.Builder("col")
                .databaseName("abc")
                .build();
        assertEquals("abc", opts.databaseName());
    }
}
