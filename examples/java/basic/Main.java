package tech.amikos.chroma.local.examples;

import tech.amikos.chroma.local.jna.JnaChromaRuntime;

public final class Main {
    private Main() {}

    public static void main(String[] args) {
        String libPath = System.getenv("CHROMA_LIB_PATH");
        if (libPath == null || libPath.isBlank()) {
            throw new IllegalStateException("CHROMA_LIB_PATH must be set");
        }

        JnaChromaRuntime runtime = JnaChromaRuntime.init(libPath);
        System.out.println("Chroma shim version: " + runtime.version());
    }
}
