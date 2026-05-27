-- name: CreateOrGetOpenIncident :one
INSERT INTO incidents (
    monitor_id
) VALUES (
    $1
)
ON CONFLICT (monitor_id) WHERE status = 'open'
-- Intentionally uses a no-op UPDATE so concurrent callers can return the open incident.
DO UPDATE SET status = incidents.status
RETURNING *;

-- name: GetOpenIncident :one
SELECT *
FROM incidents
WHERE monitor_id = $1
  AND status = 'open';

-- name: CloseOpenIncidentForMonitor :many
UPDATE incidents
SET
    status = 'closed',
    closed_at = now()
WHERE monitor_id = $1
  AND status = 'open'
RETURNING *;
