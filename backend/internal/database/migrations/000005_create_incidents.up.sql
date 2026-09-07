-- Incidents record an outage for a monitor. Deleting a monitor
-- also deletes its incidents. A monitor may have only one open incident.

CREATE TABLE incidents (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    status TEXT NOT NULL,
    failure_count INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_incidents_monitor_id
    ON incidents (monitor_id);

CREATE INDEX idx_incidents_monitor_id_started_at
    ON incidents (monitor_id, started_at DESC);

CREATE UNIQUE INDEX idx_incidents_one_open_per_monitor
    ON incidents (monitor_id)
    WHERE status = 'open';
