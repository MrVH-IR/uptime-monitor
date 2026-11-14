DROP INDEX IF EXISTS idx_monitors_locked_until;

ALTER TABLE monitors
    DROP COLUMN IF EXISTS locked_until;