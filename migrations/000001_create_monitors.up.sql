CREATE TABLE monitors
(
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name             VARCHAR(150) NOT NULL,
    url              TEXT         NOT NULL,
    http_method      VARCHAR(10)  NOT NULL DEFAULT 'GET',
    interval_seconds INTEGER      NOT NULL DEFAULT 60,
    timeout_seconds  INTEGER      NOT NULL DEFAULT 10,
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    next_check_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_checked_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_monitors_interval
        CHECK (interval_seconds >= 10),

    CONSTRAINT chk_monitors_timeout
        CHECK (timeout_seconds > 0),

    CONSTRAINT chk_monitors_http_method
        CHECK (http_method IN ('GET', 'HEAD'))
);

CREATE INDEX idx_monitors_due ON monitors (next_check_at) WHERE is_active = TRUE;