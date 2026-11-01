-- +goose Up
-- +goose StatementBegin

-- 1. Update students table: add name_normalized, school_id, unique constraint and index
ALTER TABLE students ADD COLUMN IF NOT EXISTS name_normalized VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE students ADD COLUMN IF NOT EXISTS school_id BIGINT REFERENCES schools(id) ON DELETE CASCADE;

-- Backfill school_id and name_normalized for any existing students
UPDATE students s
SET school_id = c.school_id,
    name_normalized = LOWER(TRIM(REGEXP_REPLACE(s.full_name, '\s+', ' ', 'g')))
FROM classes c
WHERE s.class_id = c.id AND (s.school_id IS NULL OR s.name_normalized = '');

-- Add unique constraint on (class_id, name_normalized)
ALTER TABLE students DROP CONSTRAINT IF EXISTS uq_students_class_name_norm;
ALTER TABLE students ADD CONSTRAINT uq_students_class_name_norm UNIQUE (class_id, name_normalized);

-- Create composite index on (school_id, class_id, active)
CREATE INDEX IF NOT EXISTS idx_students_school_class_active ON students(school_id, class_id, active);

-- 2. Create class_teachers table
CREATE TABLE IF NOT EXISTS class_teachers (
    id BIGSERIAL PRIMARY KEY,
    class_id BIGINT NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'primary', -- 'primary', 'substitute'
    valid_from DATE NOT NULL DEFAULT CURRENT_DATE,
    valid_until DATE, -- NULL for primary, date for substitute
    assigned_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Backfill existing teacher_id from classes into class_teachers as primary
INSERT INTO class_teachers (class_id, user_id, role, valid_from, valid_until, created_at)
SELECT c.id, c.teacher_id, 'primary', CURRENT_DATE, NULL, NOW()
FROM classes c
WHERE c.teacher_id IS NOT NULL
ON CONFLICT DO NOTHING;

-- One primary per class at a time (partial unique index)
CREATE UNIQUE INDEX IF NOT EXISTS idx_class_teachers_primary_active
ON class_teachers (class_id)
WHERE role = 'primary' AND valid_until IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_teachers_user ON class_teachers(user_id);
CREATE INDEX IF NOT EXISTS idx_class_teachers_lookup ON class_teachers(class_id, user_id, valid_from, valid_until);

-- 3. Create invites table
CREATE TABLE IF NOT EXISTS invites (
    id BIGSERIAL PRIMARY KEY,
    token VARCHAR(64) NOT NULL UNIQUE,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    class_id BIGINT NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'primary',
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    used_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invites_token ON invites(token);
CREATE INDEX IF NOT EXISTS idx_invites_school_class ON invites(school_id, class_id);

-- 4. Create import_jobs table
CREATE TABLE IF NOT EXISTS import_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'done', 'failed'
    total_rows INT NOT NULL DEFAULT 0,
    processed_rows INT NOT NULL DEFAULT 0,
    created_count INT NOT NULL DEFAULT 0,
    updated_count INT NOT NULL DEFAULT 0,
    skipped_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    payload JSONB,
    errors JSONB,
    summary JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_import_jobs_school_status ON import_jobs(school_id, status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS import_jobs;
DROP TABLE IF EXISTS invites;
DROP TABLE IF EXISTS class_teachers;
ALTER TABLE students DROP CONSTRAINT IF EXISTS uq_students_class_name_norm;
DROP INDEX IF EXISTS idx_students_school_class_active;
ALTER TABLE students DROP COLUMN IF EXISTS name_normalized;
ALTER TABLE students DROP COLUMN IF EXISTS school_id;
-- +goose StatementEnd
