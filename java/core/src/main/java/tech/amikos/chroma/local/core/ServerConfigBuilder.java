package tech.amikos.chroma.local.core;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

import org.yaml.snakeyaml.DumperOptions;
import org.yaml.snakeyaml.Yaml;

public final class ServerConfigBuilder {

    private int port = 8000;
    private String listenAddress = "127.0.0.1";
    private int maxPayloadSizeBytes = 41943040;
    private String persistPath = "./chroma";
    private String sqliteFilename = "chroma.sqlite3";
    private boolean allowReset = false;
    private List<String> corsAllowOrigins;
    private String otelEndpoint;
    private String otelServiceName;
    private String rawYaml;

    public ServerConfigBuilder port(int port) {
        this.port = port;
        return this;
    }

    public ServerConfigBuilder listenAddress(String listenAddress) {
        this.listenAddress = listenAddress;
        return this;
    }

    public ServerConfigBuilder maxPayloadSizeBytes(int maxPayloadSizeBytes) {
        this.maxPayloadSizeBytes = maxPayloadSizeBytes;
        return this;
    }

    public ServerConfigBuilder persistPath(String persistPath) {
        this.persistPath = persistPath;
        return this;
    }

    public ServerConfigBuilder sqliteFilename(String sqliteFilename) {
        this.sqliteFilename = sqliteFilename;
        return this;
    }

    public ServerConfigBuilder allowReset(boolean allowReset) {
        this.allowReset = allowReset;
        return this;
    }

    public ServerConfigBuilder corsAllowOrigins(List<String> corsAllowOrigins) {
        this.corsAllowOrigins = corsAllowOrigins;
        return this;
    }

    public ServerConfigBuilder otelEndpoint(String otelEndpoint) {
        this.otelEndpoint = otelEndpoint;
        return this;
    }

    public ServerConfigBuilder otelServiceName(String otelServiceName) {
        this.otelServiceName = otelServiceName;
        return this;
    }

    public ServerConfigBuilder rawYaml(String rawYaml) {
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
        if (port < 1 || port > 65535) {
            throw new IllegalArgumentException("port must be 1-65535");
        }
        if (persistPath == null || persistPath.isBlank()) {
            throw new IllegalArgumentException("persistPath must be set");
        }
        if (listenAddress == null || listenAddress.isBlank()) {
            throw new IllegalArgumentException("listenAddress must be set");
        }
    }

    private String toYaml() {
        DumperOptions options = new DumperOptions();
        options.setDefaultFlowStyle(DumperOptions.FlowStyle.BLOCK);

        Map<String, Object> map = new LinkedHashMap<>();
        map.put("port", port);
        map.put("listen_address", listenAddress);
        map.put("max_payload_size_bytes", maxPayloadSizeBytes);
        map.put("persist_path", persistPath);
        map.put("sqlite_filename", sqliteFilename);
        map.put("allow_reset", allowReset);

        if (corsAllowOrigins != null && !corsAllowOrigins.isEmpty()) {
            map.put("cors_allow_origins", corsAllowOrigins);
        }

        if (otelEndpoint != null) {
            Map<String, Object> otel = new LinkedHashMap<>();
            otel.put("endpoint", otelEndpoint);
            if (otelServiceName != null) {
                otel.put("service_name", otelServiceName);
            }
            map.put("open_telemetry", otel);
        }

        Yaml yaml = new Yaml(options);
        return yaml.dump(map);
    }
}
