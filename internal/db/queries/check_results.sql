-- name: SaveCheckResult :one
INSERT INTO check_results (
    monitor_id,
    success,
    status_code,
    latency_ms,
    error
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: ListLatestCheckResult :many
SELECT *
FROM check_results
WHERE monitor_id = $1
ORDER BY checked_at DESC
LIMIT 1;
