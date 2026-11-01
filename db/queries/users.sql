-- name: GetUserByTelegramID :one
SELECT * FROM users WHERE telegram_id = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (school_id, telegram_id, username, full_name, role, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateUserStatus :one
UPDATE users
SET status = $2, role = COALESCE($3, role), updated_at = NOW()
WHERE telegram_id = $1
RETURNING *;

-- name: UpdateUserStatusByID :one
UPDATE users
SET status = $2, role = COALESCE($3, role), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListPendingUsers :many
SELECT * FROM users
WHERE status = 'pending' AND school_id = $1
ORDER BY created_at ASC;

-- name: ListTeachersBySchool :many
SELECT * FROM users
WHERE school_id = $1 AND role = 'teacher' AND status = 'active'
ORDER BY full_name ASC;

-- name: ListAllUsersBySchool :many
SELECT * FROM users
WHERE school_id = $1
ORDER BY full_name ASC;

-- name: UpsertInvitedUser :one
INSERT INTO users (school_id, telegram_id, full_name, role, status)
VALUES ($1, $2, $3, 'teacher', 'invited')
ON CONFLICT (telegram_id) DO UPDATE
SET school_id = EXCLUDED.school_id, full_name = EXCLUDED.full_name, role = 'teacher', updated_at = NOW()
RETURNING *;

-- name: ActivateInvitedUser :one
UPDATE users
SET status = 'active', updated_at = NOW()
WHERE telegram_id = $1 AND status = 'invited'
RETURNING *;

-- name: GetDirectorBySchool :one
SELECT * FROM users
WHERE school_id = $1 AND role = 'director' AND status = 'active'
LIMIT 1;
