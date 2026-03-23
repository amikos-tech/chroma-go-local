package tech.amikos.chroma.local.core;

import static org.junit.jupiter.api.Assertions.*;

import java.util.Map;

import org.junit.jupiter.api.Test;
import org.yaml.snakeyaml.Yaml;

class EmbeddedConfigBuilderTest {

    @SuppressWarnings("unchecked")
    private Map<String, Object> parseYaml(String yaml) {
        return new Yaml().load(yaml);
    }

    @Test
    void defaultBuildMatchesGoDefaults() {
        String yaml = new EmbeddedConfigBuilder().build();
        Map<String, Object> map = parseYaml(yaml);
        assertEquals("./chroma", map.get("persist_path"));
        assertEquals("chroma.sqlite3", map.get("sqlite_filename"));
        assertEquals(false, map.get("allow_reset"));
    }

    @Test
    void customBuild() {
        String yaml = new EmbeddedConfigBuilder()
                .persistPath("/data/embedded")
                .allowReset(true)
                .build();
        Map<String, Object> map = parseYaml(yaml);
        assertEquals("/data/embedded", map.get("persist_path"));
        assertEquals(true, map.get("allow_reset"));
    }

    @Test
    void rawYamlOverridesEverything() {
        String yaml = new EmbeddedConfigBuilder()
                .persistPath("/data/other")
                .rawYaml("custom: true")
                .build();
        assertEquals("custom: true", yaml);
    }

    @Test
    void persistPathNullThrows() {
        assertThrows(IllegalArgumentException.class, () ->
                new EmbeddedConfigBuilder().persistPath(null).build());
    }

    @Test
    void persistPathBlankThrows() {
        assertThrows(IllegalArgumentException.class, () ->
                new EmbeddedConfigBuilder().persistPath("").build());
    }

    @Test
    void builderRejectsBlankRawYaml() {
        assertThrows(IllegalArgumentException.class, () ->
                new EmbeddedConfigBuilder().rawYaml("").build());
    }

    @Test
    void defaultBuildContainsOnlyThreeKeys() {
        String yaml = new EmbeddedConfigBuilder().build();
        Map<String, Object> map = parseYaml(yaml);
        assertEquals(3, map.size());
        assertTrue(map.containsKey("persist_path"));
        assertTrue(map.containsKey("sqlite_filename"));
        assertTrue(map.containsKey("allow_reset"));
    }
}
