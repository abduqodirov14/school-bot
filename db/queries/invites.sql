-- name: CreateInvite :one
INSERT INTO invites (token, school_id, class_id, role, created_by, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetInviteByToken :one
SELECT inv.*, c.name AS class_name, s.name AS school_name
FROM invites inv
JOIN classes c ON inv.class_id = c.id
JOIN schools s ON inv.school_id = s.id
WHERE inv.token = $1
LIMIT 1;

-- name: MarkInviteUsed :one
UPDATE invites
SET used_at = NOW(), used_by = $2
WHERE id = $1 AND used_at IS NULL
RETURNING *;

-- name: GetActiveInviteByClass :one
SELECT * FROM invites
WHERE class_id = $1 AND used_at IS NULL AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;
