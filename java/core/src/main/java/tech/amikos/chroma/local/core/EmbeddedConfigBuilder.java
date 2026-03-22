package tech.amikos.chroma.local.core;

import java.util.LinkedHashMap;
import java.util.Map;

import org.yaml.snakeyaml.DumperOptions;
import org.yaml.snakeyaml.Yaml;

public final class EmbeddedConfigBuilder {

    private String persistPath = "./chroma";
    private String sqliteFilename = "chroma.sqlite3";
    private boolean allowReset = false;
    private String rawYaml;

    public EmbeddedConfigBuilder persistPath(String persistPath) {
        this.persistPath = persistPath;
        return this;
    }

    public EmbeddedConfigBuilder sqliteFilename(String sqliteFilename) {
        this.sqliteFilename = sqliteFilename;
        return this;
    }

    public EmbeddedConfigBuilder allowReset(boolean allowReset) {
        this.allowReset = allowReset;
        return this;
    }

    public EmbeddedConfigBuilder rawYaml(String rawYaml) {
        this.rawYaml = rawYaml;
        return this;
    }

    public String build() {
        if (rawYaml != null) {
            return rawYaml;
        }
        validate();
        return toYaml();
    }

    private void validate() {
        if (persistPath == null || persistPath.isBlank()) {
            throw new IllegalArgumentException("persistPath must be set");
        }
    }

    private String toYaml() {
        DumperOptions options = new DumperOptions();
        options.setDefaultFlowStyle(DumperOptions.FlowStyle.BLOCK);

        Map<String, Object> map = new LinkedHashMap<>();
        map.put("persist_path", persistPath);
        map.put("sqlite_filename", sqliteFilename);
        map.put("allow_reset", allowReset);

        Yaml yaml = new Yaml(options);
        return yaml.dump(map);
    }
}
