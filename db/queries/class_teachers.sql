-- name: GetActivePrimaryTeacherForClass :one
SELECT ct.*, u.full_name AS teacher_name, u.telegram_id, u.status AS user_status
FROM class_teachers ct
JOIN users u ON ct.user_id = u.id
WHERE ct.class_id = $1 AND ct.role = 'primary' AND ct.valid_until IS NULL
LIMIT 1;

-- name: GetActiveSubstituteTeacherForClass :one
SELECT ct.*, u.full_name AS teacher_name, u.telegram_id, u.status AS user_status
FROM class_teachers ct
JOIN users u ON ct.user_id = u.id
WHERE ct.class_id = $1 AND ct.role = 'substitute' 
  AND $2::date >= ct.valid_from AND (ct.valid_until IS NULL OR $2::date <= ct.valid_until)
LIMIT 1;

-- name: GetEffectiveTeacherForClass :one
-- Returns active substitute if exists for date, otherwise active primary
SELECT ct.*, u.full_name AS teacher_name, u.telegram_id, u.status AS user_status
FROM class_teachers ct
JOIN users u ON ct.user_id = u.id
WHERE ct.class_id = $1 AND (
    (ct.role = 'substitute' AND $2::date >= ct.valid_from AND (ct.valid_until IS NULL OR $2::date <= ct.valid_until))
    OR (ct.role = 'primary' AND ct.valid_until IS NULL)
)
ORDER BY CASE WHEN ct.role = 'substitute' THEN 1 ELSE 2 END ASC
LIMIT 1;

-- name: CheckTeacherClassAccess :one
SELECT COUNT(*)::bigint AS count
FROM class_teachers ct
WHERE ct.user_id = $1 AND ct.class_id = $2 AND (
    (ct.role = 'primary' AND ct.valid_until IS NULL)
    OR (ct.role = 'substitute' AND $3::date >= ct.valid_from AND (ct.valid_until IS NULL OR $3::date <= ct.valid_until))
);

-- name: ListActiveClassesForTeacher :many
SELECT c.*, ct.role AS teacher_role, ct.valid_from, ct.valid_until,
       (SELECT COUNT(*)::bigint FROM students s WHERE s.class_id = c.id AND s.active = TRUE) AS student_count
FROM class_teachers ct
JOIN classes c ON ct.class_id = c.id
WHERE ct.user_id = $1 AND (
    (ct.role = 'primary' AND ct.valid_until IS NULL)
    OR (ct.role = 'substitute' AND $2::date >= ct.valid_from AND (ct.valid_until IS NULL OR $2::date <= ct.valid_until))
)
ORDER BY c.name ASC;

-- name: ListClassTeachersOverview :many
SELECT 
    c.id AS class_id,
    c.name AS class_name,
    c.school_id,
    (SELECT COUNT(*)::bigint FROM students s WHERE s.class_id = c.id AND s.active = TRUE) AS student_count,
    pt.id AS primary_record_id,
    pu.id AS primary_user_id,
    pu.full_name AS primary_teacher_name,
    pu.telegram_id AS primary_telegram_id,
    pu.status AS primary_user_status,
    st.id AS sub_record_id,
    su.id AS sub_user_id,
    su.full_name AS sub_teacher_name,
    st.valid_from AS sub_valid_from,
    st.valid_until AS sub_valid_until,
    inv.token AS active_invite_token,
    inv.expires_at AS active_invite_expires_at
FROM classes c
LEFT JOIN class_teachers pt ON pt.class_id = c.id AND pt.role = 'primary' AND pt.valid_until IS NULL
LEFT JOIN users pu ON pt.user_id = pu.id
LEFT JOIN class_teachers st ON st.class_id = c.id AND st.role = 'substitute' 
    AND $2::date >= st.valid_from AND (st.valid_until IS NULL OR $2::date <= st.valid_until)
LEFT JOIN users su ON st.user_id = su.id
LEFT JOIN invites inv ON inv.class_id = c.id AND inv.used_at IS NULL AND inv.expires_at > NOW()
WHERE c.school_id = $1
ORDER BY c.name ASC;

-- name: TerminateActivePrimaryTeacher :exec
UPDATE class_teachers
SET valid_until = CURRENT_DATE
WHERE class_id = $1 AND role = 'primary' AND valid_until IS NULL;

-- name: AssignPrimaryTeacher :one
INSERT INTO class_teachers (class_id, user_id, role, valid_from, valid_until, assigned_by)
VALUES ($1, $2, 'primary', CURRENT_DATE, NULL, $3)
RETURNING *;

-- name: AssignSubstituteTeacher :one
INSERT INTO class_teachers (class_id, user_id, role, valid_from, valid_until, assigned_by)
VALUES ($1, $2, 'substitute', $3, $4, $5)
RETURNING *;
