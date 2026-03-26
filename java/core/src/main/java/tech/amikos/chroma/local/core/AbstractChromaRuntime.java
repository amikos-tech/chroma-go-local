package tech.amikos.chroma.local.core;

import java.nio.file.Path;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.locks.ReentrantLock;
import java.util.function.IntSupplier;
import java.util.function.LongSupplier;

public abstract class AbstractChromaRuntime implements ChromaRuntime {

    // Process-wide: native library uses a global Mutex-guarded error slot, so all FFI calls must be serialized.
    private static final ReentrantLock FFI_LOCK = new ReentrantLock();

    private final AtomicBoolean closed = new AtomicBoolean(false);

    protected abstract String readBorrowedString(long address);

    protected abstract String readOwnedString(long address);

    protected abstract String readLastError();

    protected abstract String doVersion();

    protected abstract EmbeddedSession doStartEmbedded(String configYaml);

    protected abstract ServerSession doStartServer(String configYaml);

    protected void ensureOpen() {
        if (closed.get()) {
            throw new IllegalStateException("runtime is closed");
        }
    }

    @Override
    public final void close() {
        if (closed.compareAndSet(false, true)) {
            try {
                doClose();
            } catch (RuntimeException | Error e) {
                closed.set(false);
                throw e;
            }
        }
    }

    protected void doClose() {}

    @Override
    public final String version() {
        ensureOpen();
        return doVersion();
    }

    @Override
    public final EmbeddedSession startEmbedded(String configYaml) {
        ensureOpen();
        requireNonBlankConfig(configYaml);
        return doStartEmbedded(configYaml);
    }

    @Override
    public final ServerSession startServer(String configYaml) {
        ensureOpen();
        requireNonBlankConfig(configYaml);
        return doStartServer(configYaml);
    }

    protected static Path validateLibraryPath(String libraryPath) {
        if (libraryPath == null || libraryPath.trim().isEmpty()) {
            throw new IllegalArgumentException("libraryPath must be set");
        }
        return Path.of(libraryPath).toAbsolutePath().normalize();
    }

    protected static void ffiLock() { FFI_LOCK.lock(); }

    protected static void ffiUnlock() { FFI_LOCK.unlock(); }

    @FunctionalInterface
    protected interface FfiAction {
        void run() throws Throwable;
    }

    protected void callFfiFree(long handle, FfiAction freeCall) {
        if (handle == 0L) return;
        FFI_LOCK.lock();
        try {
            freeCall.run();
        } catch (Throwable t) {
            if (t instanceof Error error) throw error;
            throw new ChromaException("failed to free native handle", t);
        } finally {
            FFI_LOCK.unlock();
        }
    }

    protected long callFfiHandle(LongSupplier ffiCall) {
        FFI_LOCK.lock();
        try {
            long result = ffiCall.getAsLong();
            if (result == 0L) {
                String error = readLastError();
                throw new ChromaException(error != null ? error : "FFI call returned null handle");
            }
            return result;
        } finally {
            FFI_LOCK.unlock();
        }
    }

    protected int callFfiInt(IntSupplier ffiCall) {
        FFI_LOCK.lock();
        try {
            int result = ffiCall.getAsInt();
            if (result < 0) {
                String error = readLastError();
                throw new ChromaException(
                        error != null ? error : "FFI call returned error code: " + result);
            }
            return result;
        } finally {
            FFI_LOCK.unlock();
        }
    }

    protected <T> T callFfiJson(LongSupplier ffiCall, Class<T> type) {
        String json;
        FFI_LOCK.lock();
        try {
            long ptr = ffiCall.getAsLong();
            if (ptr == 0L) {
                String error = readLastError();
                throw new ChromaException(error != null ? error : "FFI call returned null pointer");
            }
            json = readOwnedString(ptr);
        } finally {
            FFI_LOCK.unlock();
        }
        if (json == null || json.isEmpty()) {
            throw new ChromaException("FFI call returned null/empty response for " + type.getSimpleName());
        }
        try {
            T result = JsonUtil.fromJson(json, type);
            if (result == null) {
                throw new ChromaException("Deserialization returned null for " + type.getSimpleName());
            }
            return result;
        } catch (com.google.gson.JsonParseException e) {
            throw new ChromaException("Failed to deserialize as " + type.getSimpleName() + ": " + e.getMessage(), e);
        }
    }

    // Rust shim clears the error on read (slot.take()), so polling readLastError() after a
    // successful void call will return null — no stale-error risk as long as the shim contract holds.
    protected void callFfiVoid(Runnable ffiCall) {
        FFI_LOCK.lock();
        try {
            ffiCall.run();
            String error = readLastError();
            if (error != null && !error.isEmpty()) {
                throw new ChromaException("FFI call failed: " + error);
            }
        } catch (RuntimeException | Error e) {
            try { readLastError(); } catch (RuntimeException | Error drain) { e.addSuppressed(drain); }
            throw e;
        } finally {
            FFI_LOCK.unlock();
        }
    }

    protected String callFfiBorrowedString(LongSupplier ffiCall) {
        FFI_LOCK.lock();
        try {
            long ptr = ffiCall.getAsLong();
            if (ptr == 0L) {
                String error = readLastError();
                throw new ChromaException(error != null ? error : "FFI call returned null pointer");
            }
            return readBorrowedString(ptr);
        } finally {
            FFI_LOCK.unlock();
        }
    }

    private static void requireNonBlankConfig(String configYaml) {
        if (configYaml == null || configYaml.isBlank()) {
            throw new IllegalArgumentException("configYaml must be set");
        }
    }
}
