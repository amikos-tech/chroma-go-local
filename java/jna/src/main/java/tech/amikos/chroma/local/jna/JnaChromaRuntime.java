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
        JnaBindings bindings = Native.load(normalized.toString(), JnaBindings.class);
        return new JnaChromaRuntime(bindings);
    }

    @Override
    public String version() {
        Pointer ptr = bindings.chroma_version();
        if (ptr == null || Pointer.nativeValue(ptr) == 0L) {
            throw new ChromaException("chroma_version returned NULL");
        }
        return ptr.getString(0);
    }

    @Override
    public EmbeddedSession startEmbedded(String configYaml) {
        if (configYaml == null || configYaml.isBlank()) {
            throw new IllegalArgumentException("configYaml must be set");
        }
        Pointer handle = bindings.chroma_embedded_start_from_string(configYaml);
        if (handle == null || Pointer.nativeValue(handle) == 0L) {
            throw new ChromaException(lastError("embedded startup failed"));
        }
        return new EmbeddedSession(Pointer.nativeValue(handle), this::embeddedFree);
    }

    private void embeddedFree(long handle) {
        if (handle == 0L) {
            return;
        }
        bindings.chroma_embedded_free(new Pointer(handle));
    }

    private String lastError(String fallback) {
        Pointer ptr = bindings.chroma_get_last_error();
        if (ptr == null || Pointer.nativeValue(ptr) == 0L) {
            return fallback;
        }
        String message = ptr.getString(0);
        bindings.chroma_string_free(ptr);
        if (message == null || message.isBlank()) {
            return fallback;
        }
        return message;
    }
}
