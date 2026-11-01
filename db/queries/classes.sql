-- name: GetClassByID :one
SELECT c.*, u.full_name AS teacher_name
FROM classes c
LEFT JOIN users u ON c.teacher_id = u.id
WHERE c.id = $1;

-- name: GetClassByName :one
SELECT * FROM classes
WHERE school_id = $1 AND name = $2 LIMIT 1;

-- name: ListClassesBySchool :many
SELECT c.*, u.full_name AS teacher_name
FROM classes c
LEFT JOIN users u ON c.teacher_id = u.id
WHERE c.school_id = $1
ORDER BY c.name ASC;

-- name: ListClassesWithStudentCount :many
SELECT 
    c.id,
    c.school_id,
    c.teacher_id,
    c.name,
    c.created_at,
    c.updated_at,
    COALESCE(u.full_name, '') AS teacher_name,
    COUNT(s.id)::bigint AS student_count
FROM classes c
LEFT JOIN users u ON c.teacher_id = u.id
LEFT JOIN students s ON s.class_id = c.id AND s.active = TRUE
WHERE c.school_id = $1
GROUP BY c.id, c.school_id, c.teacher_id, c.name, c.created_at, c.updated_at, u.full_name
ORDER BY c.name ASC;

-- name: ListClassesByTeacherID :many
SELECT * FROM classes
WHERE teacher_id = $1
ORDER BY name ASC;

-- name: CreateClass :one
INSERT INTO classes (school_id, teacher_id, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateClass :one
UPDATE classes
SET name = $2, teacher_id = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateClassTeacher :one
UPDATE classes
SET teacher_id = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteClass :exec
DELETE FROM classes WHERE id = $1;
