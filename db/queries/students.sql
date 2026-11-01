-- name: GetStudentByID :one
SELECT s.*, c.name AS class_name
FROM students s
JOIN classes c ON s.class_id = c.id
WHERE s.id = $1;

-- name: ListStudentsByClass :many
SELECT * FROM students
WHERE class_id = $1 AND active = TRUE
ORDER BY full_name ASC;

-- name: ListStudentsBySchool :many
SELECT s.*, c.name AS class_name
FROM students s
JOIN classes c ON s.class_id = c.id
WHERE c.school_id = $1 AND s.active = TRUE
ORDER BY c.name ASC, s.full_name ASC;

-- name: CreateStudent :one
INSERT INTO students (class_id, full_name, active)
VALUES ($1, $2, TRUE)
RETURNING *;

-- name: UpdateStudent :one
UPDATE students
SET full_name = $2, class_id = $3, active = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateStudent :exec
UPDATE students
SET active = FALSE, updated_at = NOW()
WHERE id = $1;

-- name: DeleteStudent :exec
DELETE FROM students WHERE id = $1;

-- name: CountActiveStudentsBySchool :one
SELECT COUNT(*) FROM students s
JOIN classes c ON s.class_id = c.id
WHERE c.school_id = $1 AND s.active = TRUE;
