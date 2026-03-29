package tech.amikos.chroma.local.core;

import java.util.Objects;

/**
 * Result of a maintenance operation that stops and restarts the server.
 * Callers must check {@link #session()} and {@link #restartError()} after each call:
 * <ul>
 *   <li>{@code session != null, restartError == null} — success</li>
 *   <li>{@code session != null, restartError != null} — operation succeeded, non-fatal teardown error</li>
 *   <li>{@code session == null, restartError != null} — server failed to restart</li>
 * </ul>
 */
public final class MaintenanceResult<R, S> {
    private final R result;
    private final S session;
    private final RuntimeException restartError;

    MaintenanceResult(R result, S session, RuntimeException restartError) {
        Objects.requireNonNull(result, "result");
        if (session == null && restartError == null) {
            throw new IllegalArgumentException(
                    "session and restartError cannot both be null");
        }
        this.result = result;
        this.session = session;
        this.restartError = restartError;
    }

    public R result() { return result; }

    /** May be null when the server failed to restart — check {@link #restartError()}. */
    public S session() { return session; }

    /** Non-null when a non-fatal error occurred during teardown or server restart. */
    public RuntimeException restartError() { return restartError; }
}
