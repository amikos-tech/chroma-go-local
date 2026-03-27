package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;
import static org.junit.jupiter.api.Assertions.assertThrows;

import org.junit.jupiter.api.Test;

class BackupFileMetadataTest {

    @Test
    void rejectsNullPath() {
        assertThrows(NullPointerException.class, () ->
                new BackupFileMetadata(null, 0, "0644", "abc", "now"));
    }

    @Test
    void rejectsNullSha256() {
        assertThrows(NullPointerException.class, () ->
                new BackupFileMetadata("f.txt", 0, "0644", null, "now"));
    }

    @Test
    void rejectsNullModifiedAt() {
        assertThrows(NullPointerException.class, () ->
                new BackupFileMetadata("f.txt", 0, "0644", "abc", null));
    }

    @Test
    void rejectsNegativeSizeBytes() {
        assertThrows(IllegalArgumentException.class, () ->
                new BackupFileMetadata("f.txt", -1, "0644", "abc", "now"));
    }

    @Test
    void allowsNullMode() {
        assertDoesNotThrow(() ->
                new BackupFileMetadata("f.txt", 0, null, "abc", "now"));
    }
}
