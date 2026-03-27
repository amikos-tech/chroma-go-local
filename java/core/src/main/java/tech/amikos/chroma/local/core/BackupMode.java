package tech.amikos.chroma.local.core;

public enum BackupMode {
    SERVER("server"),
    EMBEDDED("embedded");

    private final String wire;

    BackupMode(String wire) {
        this.wire = wire;
    }

    public String toWire() {
        return wire;
    }
}
