package tech.amikos.chroma.local.core;

public final class ChromaException extends RuntimeException {
    public ChromaException(String message) {
        super(message);
    }

    public ChromaException(String message, Throwable cause) {
        super(message, cause);
    }
}
