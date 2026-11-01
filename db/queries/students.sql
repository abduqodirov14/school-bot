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

-- name: UpsertStudentNormalized :one
INSERT INTO students (school_id, class_id, full_name, name_normalized, active)
VALUES ($1, $2, $3, $4, TRUE)
ON CONFLICT (class_id, name_normalized)
DO UPDATE SET full_name = EXCLUDED.full_name, active = TRUE, updated_at = NOW()
RETURNING *;

-- name: FindStudentByNormalizedName :one
SELECT * FROM students
WHERE class_id = $1 AND name_normalized = $2 LIMIT 1;

-- name: DeactivateMissingStudents :exec
UPDATE students
SET active = FALSE, updated_at = NOW()
WHERE school_id = $1 AND id = ANY($2::bigint[]);

-- name: CountActiveStudentsBySchool :one
SELECT COUNT(*) FROM students s
JOIN classes c ON s.class_id = c.id
WHERE c.school_id = $1 AND s.active = TRUE;
