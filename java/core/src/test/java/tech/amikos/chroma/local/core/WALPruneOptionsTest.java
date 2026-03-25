package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.*;

import org.junit.jupiter.api.Test;

class WALPruneOptionsTest {

    @Test
    void toJsonMatchesGoFormat() {
        WALPruneOptions opts = new WALPruneOptions.Builder("test_col")
                .dryRun(true)
                .vacuum(true)
                .maxAgeSeconds(3600)
                .build();
        String json = opts.toJson();
        assertTrue(json.contains("\"name\""));
        assertTrue(json.contains("\"test_col\""));
        assertTrue(json.contains("\"dry_run\""));
        assertTrue(json.contains("\"vacuum\""));
        assertTrue(json.contains("\"max_age_seconds\""));
        assertTrue(json.contains("3600"));
    }

    @Test
    void toJsonOmitsNullFields() {
        WALPruneOptions opts = new WALPruneOptions.Builder("col")
                .dryRun(true)
                .build();
        String json = opts.toJson();
        assertFalse(json.contains("max_age_seconds"));
        assertFalse(json.contains("max_bytes"));
        assertFalse(json.contains("watermark_high_bytes"));
        assertFalse(json.contains("watermark_low_bytes"));
    }

    @Test
    void builderRejectsMissingPolicyWhenNotDryRun() {
        assertThrows(IllegalArgumentException.class, () ->
                new WALPruneOptions.Builder("col")
                        .dryRun(false)
                        .build());
    }

    @Test
    void builderAcceptsMissingPolicyWhenDryRun() {
        WALPruneOptions opts = new WALPruneOptions.Builder("col")
                .dryRun(true)
                .build();
        assertNotNull(opts);
        assertTrue(opts.dryRun());
    }

    @Test
    void builderAcceptsSinglePolicyMaxAge() {
        WALPruneOptions opts = new WALPruneOptions.Builder("col")
                .maxAgeSeconds(100)
                .build();
        assertEquals(Long.valueOf(100), opts.maxAgeSeconds());
        assertFalse(opts.dryRun());
    }

    @Test
    void builderRejectsWatermarkLowGreaterThanHigh() {
        assertThrows(IllegalArgumentException.class, () ->
                new WALPruneOptions.Builder("col")
                        .watermark(100, 200)
                        .build());
    }

    @Test
    void builderRejectsMaxAgeSecondsZero() {
        assertThrows(IllegalArgumentException.class, () ->
                new WALPruneOptions.Builder("col")
                        .maxAgeSeconds(0)
                        .build());
    }

    @Test
    void builderAcceptsAllPolicies() {
        WALPruneOptions opts = new WALPruneOptions.Builder("col")
                .maxAgeSeconds(3600)
                .maxBytes(1024)
                .watermark(2048, 1024)
                .build();
        assertEquals(Long.valueOf(3600), opts.maxAgeSeconds());
        assertEquals(Long.valueOf(1024), opts.maxBytes());
        assertEquals(Long.valueOf(2048), opts.watermarkHighBytes());
        assertEquals(Long.valueOf(1024), opts.watermarkLowBytes());
    }

    @Test
    void defaultsForCollection() {
        WALPruneOptions opts = WALPruneOptions.defaults("my_col");
        assertEquals("my_col", opts.name());
        assertTrue(opts.dryRun());
    }

    @Test
    void pruneAllBuilder() {
        WALPruneOptions opts = new WALPruneOptions.Builder()
                .dryRun(true)
                .tenantId("tenant_abc")
                .build();
        assertNull(opts.name());
        assertEquals("tenant_abc", opts.tenantId());
    }

    @Test
    void builderRejectsNegativeMaxAgeSeconds() {
        assertThrows(IllegalArgumentException.class, () ->
                new WALPruneOptions.Builder("col")
                        .maxAgeSeconds(-5)
                        .build());
    }

    @Test
    void builderRejectsNegativeMaxBytes() {
        assertThrows(IllegalArgumentException.class, () ->
                new WALPruneOptions.Builder("col")
                        .maxBytes(-100)
                        .build());
    }

    @Test
    void builderRejectsZeroMaxBytes() {
        assertThrows(IllegalArgumentException.class, () ->
                new WALPruneOptions.Builder("col")
                        .maxBytes(0)
                        .build());
    }

    @Test
    void builderRejectsShortDatabaseName() {
        assertThrows(IllegalArgumentException.class, () ->
                new WALPruneOptions.Builder("col")
                        .dryRun(true)
                        .databaseName("ab")
                        .build());
    }

    @Test
    void builderRejectsShortTenantId() {
        assertThrows(IllegalArgumentException.class, () ->
                new WALPruneOptions.Builder("col")
                        .dryRun(true)
                        .tenantId("ab")
                        .build());
    }

    @Test
    void builderRejectsBlankName() {
        assertThrows(IllegalArgumentException.class, () ->
                new WALPruneOptions.Builder("   ")
                        .dryRun(true)
                        .build());
    }
}
