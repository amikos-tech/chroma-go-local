package tech.amikos.chroma.local.jna;

import com.sun.jna.Library;
import com.sun.jna.Native;
import com.sun.jna.Pointer;
import java.nio.file.Path;
import tech.amikos.chroma.local.core.ChromaException;
import tech.amikos.chroma.local.core.ChromaRuntime;
import tech.amikos.chroma.local.core.EmbeddedSession;

public final class JnaChromaRuntime implements ChromaRuntime {
    private final JnaBindings bindings;

    private interface JnaBindings extends Library {
        Pointer chroma_version();

        Pointer chroma_get_last_error();

        void chroma_string_free(Pointer value);

        Pointer chroma_embedded_start_from_string(String configYaml);

        void chroma_embedded_free(Pointer handle);
    }

    private JnaChromaRuntime(JnaBindings bindings) {
        this.bindings = bindings;
    }

    public static JnaChromaRuntime init(String libraryPath) {
        if (libraryPath == null || libraryPath.trim().isEmpty()) {
            throw new IllegalArgumentException("libraryPath must be set");
        }
        Path normalized = Path.of(libraryPath).toAbsolutePath().normalize();
        try {
            JnaBindings bindings = Native.load(normalized.toString(), JnaBindings.class);
            return new JnaChromaRuntime(bindings);
        } catch (UnsatisfiedLinkError | RuntimeException e) {
            throw new ChromaException("failed to initialize JNA runtime from " + normalized, e);
        }
    }

    @Override
    public String version() {
        try {
            // chroma_version returns a static C string owned by the runtime.
            // Do not call chroma_string_free on this pointer.
            Pointer ptr = bindings.chroma_version();
            if (ptr == null || Pointer.nativeValue(ptr) == 0L) {
                throw new ChromaException("chroma_version returned NULL");
            }
            return ptr.getString(0);
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof UnsatisfiedLinkError e) {
                throw new ChromaException("failed to read chroma_version", e);
            }
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to read chroma_version", t);
        }
    }

    @Override
    public EmbeddedSession startEmbedded(String configYaml) {
        if (configYaml == null || configYaml.isBlank()) {
            throw new IllegalArgumentException("configYaml must be set");
        }
        try {
            Pointer handle = bindings.chroma_embedded_start_from_string(configYaml);
            if (handle == null || Pointer.nativeValue(handle) == 0L) {
                throw new ChromaException(lastError("embedded startup failed"));
            }
            return new EmbeddedSession(Pointer.nativeValue(handle), this::embeddedFree);
        } catch (ChromaException e) {
            throw e;
        } catch (Throwable t) {
            if (t instanceof UnsatisfiedLinkError e) {
                throw new ChromaException("failed to start embedded runtime", e);
            }
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to start embedded runtime", t);
        }
    }

    private void embeddedFree(long handle) {
        if (handle == 0L) {
            return;
        }
        try {
            bindings.chroma_embedded_free(new Pointer(handle));
        } catch (Throwable t) {
            if (t instanceof UnsatisfiedLinkError e) {
                throw new ChromaException("failed to free embedded handle", e);
            }
            if (t instanceof Error error) {
                throw error;
            }
            throw new ChromaException("failed to free embedded handle", t);
        }
    }

    private String lastError(String fallback) {
        try {
            Pointer ptr = bindings.chroma_get_last_error();
            if (ptr == null || Pointer.nativeValue(ptr) == 0L) {
                return fallback;
            }
            String message;
            try {
                message = ptr.getString(0);
            } finally {
                bindings.chroma_string_free(ptr);
            }
            if (message == null || message.isBlank()) {
                return fallback;
            }
            return message;
        } catch (Throwable t) {
            if (t instanceof Error error && !(error instanceof UnsatisfiedLinkError)) {
                throw error;
            }
            String detail = t.getMessage();
            if (detail == null || detail.isBlank()) {
                return fallback + " (failed to retrieve native error details)";
            }
            return fallback + " (failed to retrieve native error details: " + detail + ")";
        }
    }

    @Override
    public void close() {
        // JNA runtime doesn't own shared native resources beyond embedded sessions.
    }
}
