-- name: GetSettingsBySchoolID :one
SELECT * FROM settings
WHERE school_id = $1 LIMIT 1;

-- name: UpsertSettings :one
INSERT INTO settings (school_id, absence_threshold, deadline_time, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (school_id) DO UPDATE
SET 
    absence_threshold = EXCLUDED.absence_threshold,
    deadline_time = EXCLUDED.deadline_time,
    updated_at = NOW()
RETURNING *;

