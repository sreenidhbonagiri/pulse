-- Store one HTTP check outcome per row.
-- Deleting a monitor also deletes its check results.

CREATE TABLE check_results (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status_code INTEGER,
    response_time_ms INTEGER NOT NULL,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    checked_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_check_results_monitor_id
    ON check_results (monitor_id);

CREATE INDEX idx_check_results_monitor_id_checked_at
    ON check_results (monitor_id, checked_at DESC);
