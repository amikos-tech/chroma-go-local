package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.util.List;
import org.junit.jupiter.api.Test;

class BackupResultTest {

    private static BackupManifest testManifest() {
        return new BackupManifest("v1", "embedded", "now", "java",
                List.of("/src"), "/dst", "/dst/persist", "/dst/manifest.json",
                false, 0, 0, null);
    }

    @Test
    void manifestAndSessionAccessors() {
        BackupManifest manifest = testManifest();
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
        BackupManifest manifest = testManifest();
        BackupResult<String> result = new BackupResult<>(manifest, null);
        assertEquals(manifest, result.manifest());
        assertNull(result.session());
    }
}
