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
            String json = readOwnedString(ptr);
            return JsonUtil.fromJson(json, type);
        } finally {
            FFI_LOCK.unlock();
        }
    }

    protected void callFfiVoid(Runnable ffiCall) {
        FFI_LOCK.lock();
        try {
            ffiCall.run();
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
}
