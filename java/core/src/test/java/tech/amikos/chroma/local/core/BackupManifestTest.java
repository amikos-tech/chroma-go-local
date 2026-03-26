package tech.amikos.chroma.local.core;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class BackupManifestTest {

    @Test
    void deserializesFullManifest() {
        String json = """
                {
                  "schema_version": "v1",
                  "mode": "server",
                  "created_at": "2026-03-22T10:30:00Z",
                  "wrapper_version": "0.4.0",
                  "source_paths": ["/data/chroma", "/etc/chroma.yaml"],
                  "destination_path": "/backups/2026-03-22",
                  "snapshot_path": "/backups/2026-03-22/persist",
                  "manifest_path": "/backups/2026-03-22/backup_manifest.json",
                  "include_metadata": true,
                  "file_count": 3,
                  "total_bytes": 1048576,
                  "files": [
                    {
                      "path": "chroma.sqlite3",
                      "size_bytes": 524288,
                      "mode": "0644",
                      "sha256": "abcdef1234567890",
                      "modified_at": "2026-03-22T09:00:00Z"
                    },
                    {
                      "path": "wal/wal.sqlite3",
                      "size_bytes": 524288,
                      "mode": "0644",
                      "sha256": "1234567890abcdef",
                      "modified_at": "2026-03-22T09:15:00Z"
                    }
                  ]
                }
                """;
        BackupManifest m = JsonUtil.fromJson(json, BackupManifest.class);
        assertEquals("v1", m.schemaVersion());
        assertEquals("server", m.mode());
        assertEquals("2026-03-22T10:30:00Z", m.createdAt());
        assertEquals("0.4.0", m.wrapperVersion());
        assertEquals(2, m.sourcePaths().size());
        assertEquals("/data/chroma", m.sourcePaths().get(0));
        assertEquals("/backups/2026-03-22", m.destinationPath());
        assertEquals("/backups/2026-03-22/persist", m.snapshotPath());
        assertEquals("/backups/2026-03-22/backup_manifest.json", m.manifestPath());
        assertTrue(m.includeMetadata());
        assertEquals(3, m.fileCount());
        assertEquals(1048576L, m.totalBytes());
        assertNotNull(m.files());
        assertEquals(2, m.files().size());

        BackupFileMetadata f = m.files().get(0);
        assertEquals("chroma.sqlite3", f.path());
        assertEquals(524288L, f.sizeBytes());
        assertEquals("0644", f.mode());
        assertEquals("abcdef1234567890", f.sha256());
        assertEquals("2026-03-22T09:00:00Z", f.modifiedAt());
    }

    @Test
    void deserializesManifestWithEmptyFilesList() {
        String json = """
                {
                  "schema_version": "v1",
                  "mode": "embedded",
                  "created_at": "2026-03-22T10:30:00Z",
                  "wrapper_version": "0.4.0",
                  "source_paths": ["/data/chroma"],
                  "destination_path": "/backups/latest",
                  "snapshot_path": "/backups/latest/persist",
                  "manifest_path": "/backups/latest/backup_manifest.json",
                  "include_metadata": false,
                  "file_count": 0,
                  "total_bytes": 0
                }
                """;
        BackupManifest m = JsonUtil.fromJson(json, BackupManifest.class);
        assertEquals("v1", m.schemaVersion());
        assertEquals("embedded", m.mode());
        assertFalse(m.includeMetadata());
        assertEquals(0, m.fileCount());
        assertEquals(0L, m.totalBytes());
        assertTrue(m.files().isEmpty());
    }

    @Test
    void backupFileMetadataFieldsMapCorrectly() {
        String json = """
                {
                  "path": "subdir/data.bin",
                  "size_bytes": 999999,
                  "mode": "0755",
                  "sha256": "deadbeef",
                  "modified_at": "2026-01-01T00:00:00Z"
                }
                """;
        BackupFileMetadata f = JsonUtil.fromJson(json, BackupFileMetadata.class);
        assertEquals("subdir/data.bin", f.path());
        assertEquals(999999L, f.sizeBytes());
        assertEquals("0755", f.mode());
        assertEquals("deadbeef", f.sha256());
        assertEquals("2026-01-01T00:00:00Z", f.modifiedAt());
    }

    @Test
    void files_returnsUnmodifiableList() {
        String json = """
                {
                  "schema_version": "v1", "mode": "server",
                  "created_at": "2026-03-22T10:30:00Z", "wrapper_version": "0.4.0",
                  "source_paths": ["/data/chroma"],
                  "destination_path": "/backups/x", "snapshot_path": "/backups/x/persist",
                  "manifest_path": "/backups/x/manifest.json",
                  "include_metadata": false, "file_count": 1, "total_bytes": 100,
                  "files": [
                    {
                      "path": "data.bin", "size_bytes": 100, "mode": "0644",
                      "sha256": "abc", "modified_at": "2026-01-01T00:00:00Z"
                    }
                  ]
                }
                """;
        BackupManifest m = JsonUtil.fromJson(json, BackupManifest.class);
        assertThrows(UnsupportedOperationException.class, () -> m.files().add(null));
        assertThrows(UnsupportedOperationException.class, () -> m.sourcePaths().add("new"));
    }

    @Test
    void fileCountReturnsIntAndTotalBytesReturnsLong() {
        String json = """
                {
                  "schema_version": "v1",
                  "mode": "server",
                  "created_at": "2026-03-22T10:30:00Z",
                  "wrapper_version": "0.4.0",
                  "source_paths": [],
                  "destination_path": "/backups/x",
                  "snapshot_path": "/backups/x/persist",
                  "manifest_path": "/backups/x/manifest.json",
                  "include_metadata": false,
                  "file_count": 42,
                  "total_bytes": 9999999999
                }
                """;
        BackupManifest m = JsonUtil.fromJson(json, BackupManifest.class);
        assertEquals(42, m.fileCount());
        assertEquals(9999999999L, m.totalBytes());
    }
}
