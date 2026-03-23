package tech.amikos.chroma.local.core;

import java.util.concurrent.locks.ReentrantLock;
import java.util.function.LongSupplier;

public abstract class AbstractChromaRuntime implements ChromaRuntime {

    private static final ReentrantLock FFI_LOCK = new ReentrantLock();

    protected abstract String readBorrowedString(long address);

    protected abstract String readOwnedString(long address);

    protected abstract String readLastError();

    protected long callFfiHandle(LongSupplier ffiCall) {
        FFI_LOCK.lock();
        try {
            long result = ffiCall.getAsLong();
            if (result == 0L) {
                String error = readLastError();
                throw new ChromaException(error != null ? error : "FFI call returned null handle");
            }
            readLastError(); // drain stale errors
            return result;
        } finally {
            FFI_LOCK.unlock();
        }
    }

    protected <T> T callFfiJson(LongSupplier ffiCall, Class<T> type) {
        FFI_LOCK.lock();
        try {
            long ptr = ffiCall.getAsLong();
            if (ptr == 0L) {
                String error = readLastError();
                throw new ChromaException(error != null ? error : "FFI call returned null pointer");
            }
            readLastError(); // drain stale errors
            String json = readOwnedString(ptr);
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
        } finally {
            FFI_LOCK.unlock();
        }
    }

    protected void callFfiVoid(Runnable ffiCall) {
        FFI_LOCK.lock();
        try {
            ffiCall.run();
            String error = readLastError();
            if (error != null && !error.isEmpty()) {
                throw new ChromaException("FFI call failed: " + error);
            }
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
            readLastError(); // drain stale errors
            return readBorrowedString(ptr);
        } finally {
            FFI_LOCK.unlock();
        }
    }
}
