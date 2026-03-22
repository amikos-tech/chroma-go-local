package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.*;

import java.util.List;
import java.util.Map;

import org.junit.jupiter.api.Test;
import org.yaml.snakeyaml.Yaml;

class ServerConfigBuilderTest {

    @SuppressWarnings("unchecked")
    private Map<String, Object> parseYaml(String yaml) {
        return new Yaml().load(yaml);
    }

    @Test
    void defaultBuildMatchesGoDefaults() {
        String yaml = new ServerConfigBuilder().build();
        Map<String, Object> map = parseYaml(yaml);
        assertEquals(8000, map.get("port"));
        assertEquals("127.0.0.1", map.get("listen_address"));
        assertEquals(41943040, map.get("max_payload_size_bytes"));
        assertEquals("./chroma", map.get("persist_path"));
        assertEquals("chroma.sqlite3", map.get("sqlite_filename"));
        assertEquals(false, map.get("allow_reset"));
        assertNull(map.get("cors_allow_origins"));
        assertNull(map.get("open_telemetry"));
    }

    @Test
    void customBuild() {
        String yaml = new ServerConfigBuilder()
                .port(9090)
                .listenAddress("0.0.0.0")
                .persistPath("/data/chroma")
                .allowReset(true)
                .build();
        Map<String, Object> map = parseYaml(yaml);
        assertEquals(9090, map.get("port"));
        assertEquals("0.0.0.0", map.get("listen_address"));
        assertEquals("/data/chroma", map.get("persist_path"));
        assertEquals(true, map.get("allow_reset"));
    }

    @SuppressWarnings("unchecked")
    @Test
    void corsAllowOrigins() {
        String yaml = new ServerConfigBuilder()
                .corsAllowOrigins(List.of("http://localhost:3000", "https://example.com"))
                .build();
        Map<String, Object> map = parseYaml(yaml);
        List<String> origins = (List<String>) map.get("cors_allow_origins");
        assertNotNull(origins);
        assertEquals(2, origins.size());
        assertEquals("http://localhost:3000", origins.get(0));
        assertEquals("https://example.com", origins.get(1));
    }

    @SuppressWarnings("unchecked")
    @Test
    void openTelemetry() {
        String yaml = new ServerConfigBuilder()
                .otelEndpoint("http://otel:4317")
                .otelServiceName("chroma-dev")
                .build();
        Map<String, Object> map = parseYaml(yaml);
        Map<String, Object> otel = (Map<String, Object>) map.get("open_telemetry");
        assertNotNull(otel);
        assertEquals("http://otel:4317", otel.get("endpoint"));
        assertEquals("chroma-dev", otel.get("service_name"));
    }

    @Test
    void rawYamlOverridesEverything() {
        String yaml = new ServerConfigBuilder()
                .port(9999)
                .rawYaml("custom: true")
                .build();
        assertEquals("custom: true", yaml);
    }

    @Test
    void portNegativeThrows() {
        assertThrows(IllegalArgumentException.class, () ->
                new ServerConfigBuilder().port(-1).build());
    }

    @Test
    void portTooHighThrows() {
        assertThrows(IllegalArgumentException.class, () ->
                new ServerConfigBuilder().port(70000).build());
    }

    @Test
    void persistPathNullThrows() {
        assertThrows(IllegalArgumentException.class, () ->
                new ServerConfigBuilder().persistPath(null).build());
    }

    @Test
    void persistPathBlankThrows() {
        assertThrows(IllegalArgumentException.class, () ->
                new ServerConfigBuilder().persistPath("").build());
    }

    @Test
    void defaultBuildContainsAllRequiredKeys() {
        String yaml = new ServerConfigBuilder().build();
        Map<String, Object> map = parseYaml(yaml);
        assertTrue(map.containsKey("port"));
        assertTrue(map.containsKey("listen_address"));
        assertTrue(map.containsKey("max_payload_size_bytes"));
        assertTrue(map.containsKey("persist_path"));
        assertTrue(map.containsKey("sqlite_filename"));
        assertTrue(map.containsKey("allow_reset"));
    }
}
