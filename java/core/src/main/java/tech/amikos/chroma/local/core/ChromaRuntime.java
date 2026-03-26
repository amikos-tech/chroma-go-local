package tech.amikos.chroma.local.core;

public interface ChromaRuntime extends AutoCloseable {
    String version();

    EmbeddedSession startEmbedded(String configYaml);

    ServerSession startServer(String configYaml);

    @Override
    void close();
}
