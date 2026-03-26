package tech.amikos.chroma.local.panama;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.nio.file.Path;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import tech.amikos.chroma.local.core.ChromaException;
import tech.amikos.chroma.local.core.CompactAllRequest;
import tech.amikos.chroma.local.core.CompactCollectionRequest;
import tech.amikos.chroma.local.core.CompactionResult;
import tech.amikos.chroma.local.core.EmbeddedConfigBuilder;
import tech.amikos.chroma.local.core.EmbeddedSession;
import tech.amikos.chroma.local.core.RebuildOptions;
import tech.amikos.chroma.local.core.WALPruneOptions;
import tech.amikos.chroma.local.core.WALPruneResult;

class PanamaEmbeddedMaintenanceTest {

    @Test
    void embeddedRebuildNonexistentCollectionThrows(@TempDir Path persistDir) {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        String yaml = new EmbeddedConfigBuilder()
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath);
             EmbeddedSession session = runtime.startEmbedded(yaml)) {
            assertThrows(ChromaException.class,
                    () -> session.rebuildCollection(RebuildOptions.defaults("nonexistent")));
        }
    }

    @Test
    void embeddedRebuildNullOptionsThrows(@TempDir Path persistDir) {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        String yaml = new EmbeddedConfigBuilder()
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath);
             EmbeddedSession session = runtime.startEmbedded(yaml)) {
            assertThrows(IllegalArgumentException.class,
                    () -> session.rebuildCollection((RebuildOptions) null));
        }
    }

    @Test
    void embeddedCompactAllSmoke(@TempDir Path persistDir) {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        String yaml = new EmbeddedConfigBuilder()
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath);
             EmbeddedSession session = runtime.startEmbedded(yaml)) {
            CompactionResult result = session.compactAll(new CompactAllRequest.Builder().build());
            assertNotNull(result);
            assertEquals(0, result.collectionCount());
        }
    }

    @Test
    void embeddedCompactCollectionNonexistentThrows(@TempDir Path persistDir) {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        String yaml = new EmbeddedConfigBuilder()
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath);
             EmbeddedSession session = runtime.startEmbedded(yaml)) {
            assertThrows(ChromaException.class,
                    () -> session.compactCollection(new CompactCollectionRequest.Builder("nonexistent").build()));
        }
    }

    @Test
    void embeddedPruneWalAllDryRunSmoke(@TempDir Path persistDir) {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        String yaml = new EmbeddedConfigBuilder()
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath);
             EmbeddedSession session = runtime.startEmbedded(yaml)) {
            WALPruneResult result = session.pruneAllWAL(new WALPruneOptions.Builder().dryRun(true).build());
            assertNotNull(result);
            assertTrue(result.dryRun());
            assertEquals(0, result.collectionCount());
        }
    }

    @Test
    void embeddedPruneWalCollectionNonexistentThrows(@TempDir Path persistDir) {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        String yaml = new EmbeddedConfigBuilder()
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath);
             EmbeddedSession session = runtime.startEmbedded(yaml)) {
            assertThrows(ChromaException.class,
                    () -> session.pruneCollectionWAL(WALPruneOptions.defaults("nonexistent")));
        }
    }

    @Test
    void embeddedMaintenanceOpsThrowAfterClose(@TempDir Path persistDir) {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        String yaml = new EmbeddedConfigBuilder()
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            EmbeddedSession session = runtime.startEmbedded(yaml);
            session.close();

            assertThrows(IllegalStateException.class,
                    () -> session.rebuildCollection(RebuildOptions.defaults("x")));
            assertThrows(IllegalStateException.class,
                    () -> session.compactCollection(new CompactCollectionRequest.Builder("x").build()));
            assertThrows(IllegalStateException.class,
                    () -> session.compactAll(new CompactAllRequest.Builder().build()));
            assertThrows(IllegalStateException.class,
                    () -> session.pruneCollectionWAL(WALPruneOptions.defaults("x")));
            assertThrows(IllegalStateException.class,
                    () -> session.pruneAllWAL(new WALPruneOptions.Builder().dryRun(true).build()));
        }
    }
}
