-- name: GetClassAttendanceByDate :many
SELECT 
    s.id AS student_id,
    s.full_name,
    s.class_id,
    COALESCE(a.id, 0::bigint) AS attendance_id,
    COALESCE(a.status, 'unmarked') AS status,
    COALESCE(a.note, '') AS note,
    a.marked_by,
    a.marked_at,
    a.updated_by,
    a.updated_at
FROM students s
LEFT JOIN attendance a ON s.id = a.student_id AND a.date = $2
WHERE s.class_id = $1 AND s.active = TRUE
ORDER BY s.full_name ASC;

-- name: UpsertAttendance :one
INSERT INTO attendance (
    student_id, date, status, note, marked_by, marked_at, updated_by, updated_at
)
VALUES ($1, $2, $3, $4, $5, NOW(), $5, NOW())
ON CONFLICT (student_id, date) DO UPDATE
SET 
    status = EXCLUDED.status,
    note = EXCLUDED.note,
    updated_by = EXCLUDED.updated_by,
    updated_at = NOW()
RETURNING *;

-- name: GetAttendanceRecord :one
SELECT * FROM attendance
WHERE student_id = $1 AND date = $2
LIMIT 1;

-- name: GetTodaySchoolStats :one
SELECT 
    COUNT(DISTINCT s.id)::bigint AS total_students,
    COUNT(DISTINCT CASE WHEN a.status = 'present' THEN s.id END)::bigint AS present_count,
    COUNT(DISTINCT CASE WHEN a.status = 'absent' THEN s.id END)::bigint AS absent_count,
    COUNT(DISTINCT CASE WHEN a.status = 'late' THEN s.id END)::bigint AS late_count,
    COUNT(DISTINCT CASE WHEN a.status = 'excused' THEN s.id END)::bigint AS excused_count
FROM students s
JOIN classes c ON s.class_id = c.id
LEFT JOIN attendance a ON s.id = a.student_id AND a.date = $2
WHERE c.school_id = $1 AND s.active = TRUE;

-- name: ListAbsentStudentsToday :many
SELECT 
    s.id AS student_id,
    s.full_name AS student_name,
    c.id AS class_id,
    c.name AS class_name,
    a.status,
    COALESCE(a.note, '') AS note
FROM attendance a
JOIN students s ON a.student_id = s.id
JOIN classes c ON s.class_id = c.id
WHERE c.school_id = $1 AND a.date = $2 AND a.status = 'absent'
ORDER BY c.name ASC, s.full_name ASC;

-- name: ListLateStudentsToday :many
SELECT 
    s.id AS student_id,
    s.full_name AS student_name,
    c.id AS class_id,
    c.name AS class_name,
    a.status,
    COALESCE(a.note, '') AS note
FROM attendance a
JOIN students s ON a.student_id = s.id
JOIN classes c ON s.class_id = c.id
WHERE c.school_id = $1 AND a.date = $2 AND a.status = 'late'
ORDER BY c.name ASC, s.full_name ASC;

-- name: ListClassesWithAttendanceStatusToday :many
SELECT 
    c.id AS class_id,
    c.name AS class_name,
    c.teacher_id,
    COALESCE(u.full_name, '') AS teacher_name,
    COALESCE(u.telegram_id, 0::bigint) AS teacher_telegram_id,
    COUNT(s.id)::bigint AS total_students,
    COUNT(CASE WHEN a.status IS NOT NULL AND a.status != 'unmarked' THEN 1 END)::bigint AS marked_students
FROM classes c
LEFT JOIN users u ON c.teacher_id = u.id
LEFT JOIN students s ON s.class_id = c.id AND s.active = TRUE
LEFT JOIN attendance a ON a.student_id = s.id AND a.date = $2
WHERE c.school_id = $1
GROUP BY c.id, c.name, c.teacher_id, u.full_name, u.telegram_id
ORDER BY c.name ASC;

-- name: GetStudentAttendanceHistory :many
SELECT a.*
FROM attendance a
WHERE a.student_id = $1 AND a.date >= $2 AND a.date <= $3
ORDER BY a.date ASC;

-- name: GetMonthlyAttendanceForSchool :many
SELECT 
    s.id AS student_id,
    s.full_name AS student_name,
    s.class_id,
    c.name AS class_name,
    a.date,
    a.status
FROM students s
JOIN classes c ON s.class_id = c.id
LEFT JOIN attendance a ON s.id = a.student_id AND a.date >= $2 AND a.date <= $3
WHERE c.school_id = $1 AND s.active = TRUE
ORDER BY c.name ASC, s.full_name ASC, a.date ASC;

-- name: CountUnexcusedAbsencesInMonth :one
SELECT COUNT(*)::bigint
FROM attendance
WHERE student_id = $1 
  AND date >= $2 
  AND date <= $3 
  AND status = 'absent';

-- name: GetStudentsExceedingThreshold :many
SELECT 
    s.id AS student_id,
    s.full_name AS student_name,
    c.id AS class_id,
    c.name AS class_name,
    COUNT(a.id)::bigint AS absent_count
FROM students s
JOIN classes c ON s.class_id = c.id
JOIN attendance a ON s.id = a.student_id
WHERE c.school_id = $1
  AND a.date >= $2
  AND a.date <= $3
  AND a.status = 'absent'
  AND s.active = TRUE
GROUP BY s.id, s.full_name, c.id, c.name
HAVING COUNT(a.id) >= sqlc.arg(min_absences)::bigint;
