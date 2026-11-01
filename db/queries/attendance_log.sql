-- name: CreateAttendanceLog :one
INSERT INTO attendance_log (
    attendance_id, student_id, date, changed_by, old_status, new_status, note, changed_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
RETURNING *;

-- name: ListAttendanceLogsByStudent :many
SELECT al.*, u.full_name AS changed_by_name
FROM attendance_log al
LEFT JOIN users u ON al.changed_by = u.id
WHERE al.student_id = $1
ORDER BY al.changed_at DESC;

