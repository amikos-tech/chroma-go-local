package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.*;

import org.junit.jupiter.api.Test;

class BackupOptionsTest {

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
                .leaveInactive(true)
                .build();
        assertTrue(opts.includeMetadata());
        assertTrue(opts.leaveInactive());
    }

    @Test
    void defaultsWithOnlyDestination() {
        BackupOptions opts = new BackupOptions.Builder("/tmp/backup").build();
        assertEquals("/tmp/backup", opts.destinationPath());
        assertFalse(opts.includeMetadata());
        assertFalse(opts.leaveInactive());
    }
}
