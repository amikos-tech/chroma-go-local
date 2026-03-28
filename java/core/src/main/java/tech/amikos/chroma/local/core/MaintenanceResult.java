package tech.amikos.chroma.local.core;

import java.util.Objects;

public final class MaintenanceResult<R, S> {
    private final R result;
    private final S session;
    private final Exception restartError;

    MaintenanceResult(R result, S session, Exception restartError) {
        Objects.requireNonNull(result, "result");
        this.result = result;
        this.session = session;
        this.restartError = restartError;
    }

    public R result() { return result; }

    public S session() { return session; }

    public Exception restartError() { return restartError; }
}
