package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.*;

import org.junit.jupiter.api.Test;

class BackupOptionsTest {

    @Test
    void toJsonProducesExpectedKeys() {
        BackupOptions opts = new BackupOptions.Builder("/tmp/backup")
                .includeMetadata(true)
                .build();
        String json = opts.toJson();
        assertTrue(json.contains("\"destination_path\""));
        assertTrue(json.contains("\"/tmp/backup\""));
        assertTrue(json.contains("\"include_metadata\""));
    }

    @Test
    void builderRejectsBlankDestination() {
        assertThrows(IllegalArgumentException.class, () ->
                new BackupOptions.Builder("  ").build());
    }

    @Test
    void builderRejectsNullDestination() {
        assertThrows(IllegalArgumentException.class, () ->
                new BackupOptions.Builder(null).build());
    }

    @Test
    void builderAcceptsFlags() {
        BackupOptions opts = new BackupOptions.Builder("/tmp/backup")
                .includeMetadata(true)
                .leaveStopped(true)
                .leaveClosed(true)
                .build();
        assertTrue(opts.includeMetadata());
        assertTrue(opts.leaveStopped());
        assertTrue(opts.leaveClosed());
    }

    @Test
    void defaultsWithOnlyDestination() {
        BackupOptions opts = new BackupOptions.Builder("/tmp/backup").build();
        assertEquals("/tmp/backup", opts.destinationPath());
        assertFalse(opts.includeMetadata());
        assertFalse(opts.leaveStopped());
        assertFalse(opts.leaveClosed());
    }
}
