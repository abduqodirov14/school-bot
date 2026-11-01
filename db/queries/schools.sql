-- name: GetFirstSchool :one
SELECT * FROM schools ORDER BY id ASC LIMIT 1;

-- name: GetSchoolByID :one
SELECT * FROM schools WHERE id = $1;

-- name: CreateSchool :one
INSERT INTO schools (name)
VALUES ($1)
RETURNING *;

