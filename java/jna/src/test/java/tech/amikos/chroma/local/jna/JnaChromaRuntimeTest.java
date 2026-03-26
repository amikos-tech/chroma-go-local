package tech.amikos.chroma.local.jna;

import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.IOException;
import java.net.ServerSocket;
import java.nio.file.Path;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.CleanupMode;
import org.junit.jupiter.api.io.TempDir;
import tech.amikos.chroma.local.core.ChromaException;
import tech.amikos.chroma.local.core.EmbeddedSession;
import tech.amikos.chroma.local.core.ServerSession;

class JnaChromaRuntimeTest {
    @Test
    void initRejectsMissingLibraryPath() {
        assertThrows(IllegalArgumentException.class, () -> JnaChromaRuntime.init(null));
        assertThrows(IllegalArgumentException.class, () -> JnaChromaRuntime.init(""));
        assertThrows(IllegalArgumentException.class, () -> JnaChromaRuntime.init("   "));
    }

    @Test
    void initWrapsNativeLoadFailures() {
        assertThrows(ChromaException.class, () -> JnaChromaRuntime.init("/nonexistent/libchroma_shim.so"));
    }

    @Test
    void versionAndEmbeddedLifecycleSmokeTest(@TempDir Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        String yaml = embeddedYaml(persistDir);

        try (JnaChromaRuntime runtime = JnaChromaRuntime.init(libPath)) {
            String version = runtime.version();
            assertNotNull(version);
            assertFalse(version.isBlank());

            try (EmbeddedSession ignored = runtime.startEmbedded(yaml)) {
                // Smoke test ensures startup and close work via JNA bindings.
            }
        }
    }

    @Test
    void serverLifecycleSmokeTest(@TempDir(cleanup = CleanupMode.NEVER) Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = serverYaml(persistDir, port);

        try (JnaChromaRuntime runtime = JnaChromaRuntime.init(libPath);
             ServerSession session = runtime.startServer(yaml)) {
            assertTrue(session.port() > 0);
            assertNotNull(session.address());
            assertNotNull(session.persistPath());
        }
    }

    @Test
    void startEmbeddedRejectsMissingYaml() {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        try (JnaChromaRuntime runtime = JnaChromaRuntime.init(libPath)) {
            assertThrows(IllegalArgumentException.class, () -> runtime.startEmbedded(null));
            assertThrows(IllegalArgumentException.class, () -> runtime.startEmbedded(""));
            assertThrows(IllegalArgumentException.class, () -> runtime.startEmbedded("   "));
        }
    }

    @Test
    void rejectsOperationsAfterClose(@TempDir Path persistDir) {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        String yaml = embeddedYaml(persistDir);
        JnaChromaRuntime runtime = JnaChromaRuntime.init(libPath);
        runtime.close();

        assertThrows(IllegalStateException.class, runtime::version);
        assertThrows(IllegalStateException.class, () -> runtime.startEmbedded(yaml));
        assertThrows(IllegalStateException.class, () -> runtime.startServer(serverYaml(persistDir, 9999)));
        assertDoesNotThrow(runtime::close);
    }

    private static String embeddedYaml(Path persistDir) {
        String escapedPath = persistDir.toAbsolutePath().toString().replace("\\", "\\\\");
        return "persist_path: \"" + escapedPath + "\"\n"
                + "sqlite_filename: \"chroma.sqlite3\"\n"
                + "allow_reset: true\n";
    }

    private static String serverYaml(Path persistDir, int port) {
        String escapedPath = persistDir.toAbsolutePath().toString().replace("\\", "\\\\");
        return "port: " + port + "\n"
                + "listen_address: \"127.0.0.1\"\n"
                + "persist_path: \"" + escapedPath + "\"\n"
                + "sqlite_filename: \"chroma.sqlite3\"\n"
                + "allow_reset: true\n";
    }

    private static int findFreePort() throws IOException {
        try (ServerSocket s = new ServerSocket(0)) {
            return s.getLocalPort();
        }
    }
}
