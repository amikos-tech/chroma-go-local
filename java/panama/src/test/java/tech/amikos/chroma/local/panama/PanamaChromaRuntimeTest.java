package tech.amikos.chroma.local.panama;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotNull;

import java.nio.file.Files;
import java.nio.file.Path;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.Test;
import tech.amikos.chroma.local.core.EmbeddedSession;

class PanamaChromaRuntimeTest {
    @Test
    void versionAndEmbeddedLifecycleSmokeTest() throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        Path persistDir = Files.createTempDirectory("chroma-panama-smoke-");
        String yaml = embeddedYaml(persistDir);

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            String version = runtime.version();
            assertNotNull(version);
            assertFalse(version.isBlank());

            try (EmbeddedSession ignored = runtime.startEmbedded(yaml)) {
                // Smoke test ensures startup and close work via Panama bindings.
            }
        }
    }

    private static String embeddedYaml(Path persistDir) {
        String escapedPath = persistDir.toAbsolutePath().toString().replace("\\", "\\\\");
        return "persist_path: \"" + escapedPath + "\"\n"
                + "sqlite_filename: \"chroma.sqlite3\"\n"
                + "allow_reset: true\n";
    }
}
