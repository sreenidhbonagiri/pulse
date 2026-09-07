-- Add scheduling metadata so the scheduler knows when each monitor is due.
-- Existing monitors become due immediately so they can be picked up.

ALTER TABLE monitors
    ADD COLUMN next_check_at TIMESTAMPTZ;

UPDATE monitors
SET next_check_at = NOW()
WHERE next_check_at IS NULL;

CREATE INDEX idx_monitors_due
    ON monitors (next_check_at)
    WHERE is_active = true;
