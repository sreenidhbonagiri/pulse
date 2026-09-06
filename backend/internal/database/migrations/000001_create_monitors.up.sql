-- Create the monitors table.
-- This is an "up" migration: it applies the change.

CREATE TABLE monitors (
    id UUID PRIMARY KEY,
    user_id UUID,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    http_method TEXT NOT NULL,
    check_interval_seconds INTEGER NOT NULL CHECK (check_interval_seconds > 0),
    timeout_seconds INTEGER NOT NULL CHECK (timeout_seconds > 0),
    expected_status_code INTEGER NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
