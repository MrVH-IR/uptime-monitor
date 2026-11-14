ALTER TABLE monitors
    ADD COLUMN locked_until TIMESTAMPTZ;

CREATE INDEX idx_monitors_locked_until
    ON monitors (locked_until)
    WHERE locked_until IS NOT NULL;