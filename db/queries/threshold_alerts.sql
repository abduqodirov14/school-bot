-- name: HasThresholdAlertBeenSent :one
SELECT COUNT(*) > 0
FROM threshold_alerts
WHERE student_id = $1 AND year_month = $2;

-- name: CreateThresholdAlert :exec
INSERT INTO threshold_alerts (student_id, year_month, notified_at)
VALUES ($1, $2, NOW())
ON CONFLICT (student_id, year_month) DO NOTHING;
