package tech.amikos.chroma.local.panama;

import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.io.IOException;
import java.net.ServerSocket;
import java.nio.file.Path;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import tech.amikos.chroma.local.core.ChromaException;
import tech.amikos.chroma.local.core.ServerConfigBuilder;
import tech.amikos.chroma.local.core.ServerSession;

class PanamaServerLifecycleTest {

    @Test
    void serverStartAccessorsStopClose(@TempDir Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port)
                .listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            try (ServerSession session = runtime.startServer(yaml)) {
                assertEquals(port, session.port());
                assertEquals("127.0.0.1", session.address());
                assertEquals("http://127.0.0.1:" + port, session.url());
                assertNotNull(session.persistPath());
                assertFalse(session.persistPath().isBlank());
            }
        }
    }

    @Test
    void doubleCloseIsIdempotent(@TempDir Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port)
                .listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            session.close();
            assertDoesNotThrow(session::close);
        }
    }

    @Test
    void accessorsThrowAfterClose(@TempDir Path persistDir) throws Exception {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        int port = findFreePort();
        String yaml = new ServerConfigBuilder()
                .port(port)
                .listenAddress("127.0.0.1")
                .persistPath(persistDir.toAbsolutePath().toString())
                .allowReset(true)
                .build();

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            ServerSession session = runtime.startServer(yaml);
            session.close();

            assertThrows(IllegalStateException.class, session::port);
            assertThrows(IllegalStateException.class, session::address);
            assertThrows(IllegalStateException.class, session::url);
            assertThrows(IllegalStateException.class, session::persistPath);
        }
    }

    @Test
    void startServerRejectsNullConfig() {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            assertThrows(IllegalArgumentException.class, () -> runtime.startServer(null));
        }
    }

    @Test
    void startServerRejectsEmptyConfig() {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            assertThrows(IllegalArgumentException.class, () -> runtime.startServer(""));
            assertThrows(IllegalArgumentException.class, () -> runtime.startServer("   "));
        }
    }

    @Test
    void startServerRejectsMalformedConfig() {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        Assumptions.assumeTrue(libPath != null && !libPath.isBlank(), "CHROMA_LIB_PATH is required");

        try (PanamaChromaRuntime runtime = PanamaChromaRuntime.init(libPath)) {
            assertThrows(ChromaException.class, () -> runtime.startServer("not: valid: server: config:"));
        }
    }

    private static int findFreePort() throws IOException {
        try (ServerSocket s = new ServerSocket(0)) {
            return s.getLocalPort();
        }
    }
}
