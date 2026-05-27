-- name: CreateMonitor :one
INSERT INTO monitors (
    name,
    url,
    interval_secs,
    timeout_secs,
    enabled
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetMonitor :one
SELECT *
FROM monitors
WHERE id = $1;

-- name: ListMonitors :many
SELECT *
FROM monitors
ORDER BY created_at DESC;
