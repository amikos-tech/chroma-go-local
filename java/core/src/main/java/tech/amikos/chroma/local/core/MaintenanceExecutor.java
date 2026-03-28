package tech.amikos.chroma.local.core;

import java.util.Objects;
import java.util.function.Function;

public final class MaintenanceExecutor {

    private MaintenanceExecutor() {}

    public static <R> MaintenanceResult<R, ServerSession> execute(
            String configYaml,
            Runnable closeServerAction,
            Function<String, EmbeddedSession> startEmbeddedAction,
            Function<String, ServerSession> startServerAction,
            Function<EmbeddedSession, R> operation) {

        Objects.requireNonNull(configYaml, "configYaml");
        Objects.requireNonNull(closeServerAction, "closeServerAction");
        Objects.requireNonNull(startEmbeddedAction, "startEmbeddedAction");
        Objects.requireNonNull(startServerAction, "startServerAction");
        Objects.requireNonNull(operation, "operation");

        // Step 1: Stop+free the current server handle
        try {
            closeServerAction.run();
        } catch (RuntimeException e) {
            throw new ChromaException("failed to stop/free server before maintenance", e);
        }

        // Step 2: Start temporary embedded session
        EmbeddedSession embedded;
        try {
            embedded = startEmbeddedAction.apply(configYaml);
        } catch (RuntimeException startErr) {
            throw new ChromaException(
                    "failed to start temporary embedded runtime for maintenance; server remains stopped",
                    startErr);
        }

        // Step 3: Run operation
        R result = null;
        RuntimeException opError = null;
        try {
            result = operation.apply(embedded);
        } catch (RuntimeException e) {
            opError = e;
        }

        // Step 4: Close embedded
        RuntimeException closeError = null;
        try {
            embedded.close();
        } catch (RuntimeException e) {
            closeError = e;
        }

        // Step 5: Restart server
        ServerSession newSession = null;
        RuntimeException restartError = null;
        try {
            newSession = startServerAction.apply(configYaml);
        } catch (RuntimeException e) {
            restartError = e;
        }

        // Step 6: Error matrix (matches Go rebuild.go error-handling strategy)
        if (opError != null) {
            if (closeError != null) {
                opError.addSuppressed(closeError);
            }
            if (restartError != null) {
                ChromaException combined = new ChromaException(
                        opError.getMessage() + "; restart failed: " + restartError.getMessage()
                                + "; server remains stopped",
                        opError);
                combined.addSuppressed(restartError);
                throw combined;
            }
            throw opError;
        }

        if (closeError != null) {
            if (restartError != null) {
                return new MaintenanceResult<>(result, null,
                        new ChromaException(
                                "close embedded failed: " + closeError.getMessage()
                                        + "; restart failed: " + restartError.getMessage()
                                        + "; server remains stopped",
                                closeError));
            }
            return new MaintenanceResult<>(result, newSession,
                    new ChromaException(
                            "maintenance completed but failed to close temporary embedded runtime",
                            closeError));
        }

        if (restartError != null) {
            return new MaintenanceResult<>(result, null, restartError);
        }

        return new MaintenanceResult<>(result, newSession, null);
    }
}
