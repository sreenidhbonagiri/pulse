DROP INDEX IF EXISTS idx_monitors_due;

ALTER TABLE monitors
    DROP COLUMN IF EXISTS next_check_at;
