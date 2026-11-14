CREATE TABLE check_results
(
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    monitor_id       BIGINT      NOT NULL,
    status_code      SMALLINT,
    response_time_ms BIGINT,
    is_success       BOOLEAN     NOT NULL,
    error_type       VARCHAR(50) NOT NULL,
    error_message    TEXT        NOT NULL,
    started_at       TIMESTAMPTZ NOT NULL,
    finished_at      TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_check_results_monitor FOREIGN KEY (monitor_id) REFERENCES monitors (id) ON DELETE CASCADE,
    CONSTRAINT chk_check_results_status_code CHECK (status_code IS NULL OR status_code BETWEEN 100 AND 599),
    CONSTRAINT chk_check_results_response_time CHECK (response_time_ms IS NULL OR response_time_ms >= 0),
    CONSTRAINT chk_check_results_time CHECK (finished_at >= started_at)
);

CREATE INDEX idx_check_results_monitor_history ON check_results (monitor_id, started_at DESC)