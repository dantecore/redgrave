-- +goose Up
-- +goose StatementBegin

-- ── monitors ──
-- Each row represents a single HTTP endpoint to health-check on a schedule.

CREATE TABLE monitors (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT        NOT NULL,
    url             TEXT        NOT NULL,
    interval_secs   INTEGER     NOT NULL DEFAULT 60,
    timeout_secs    INTEGER     NOT NULL DEFAULT 10,
    enabled         BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── check_results ──
-- Immutable log of every ping executed against a monitor.

CREATE TABLE check_results (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id      UUID        NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    success         BOOLEAN     NOT NULL,
    status_code     INTEGER,
    latency_ms      BIGINT,
    error           TEXT,
    checked_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_check_results_monitor_time
    ON check_results (monitor_id, checked_at DESC);

-- ── incidents ──
-- Opened after N consecutive failures; closed when the monitor recovers.

CREATE TABLE incidents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id      UUID        NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status          TEXT        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    opened_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at       TIMESTAMPTZ
);

CREATE INDEX idx_incidents_monitor_status
    ON incidents (monitor_id, status);
CREATE UNIQUE INDEX idx_incidents_one_open_per_monitor
    ON incidents (monitor_id)
    WHERE status = 'open';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS incidents;
DROP TABLE IF EXISTS check_results;
DROP TABLE IF EXISTS monitors;
-- +goose StatementEnd
