package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;

import org.junit.jupiter.api.Test;

class BackupResultTest {

    @Test
    void manifestAndSessionAccessors() {
        BackupManifest manifest = new BackupManifest();
        String session = "test-session";
        BackupResult<String> result = new BackupResult<>(manifest, session);
        assertEquals(manifest, result.manifest());
        assertEquals("test-session", result.session());
    }

    @Test
    void rejectsNullManifest() {
        assertThrows(NullPointerException.class, () -> new BackupResult<>(null, "session"));
    }

    @Test
    void allowsNullSession() {
        BackupManifest manifest = new BackupManifest();
        BackupResult<String> result = new BackupResult<>(manifest, null);
        assertEquals(manifest, result.manifest());
        assertNull(result.session());
    }
}
