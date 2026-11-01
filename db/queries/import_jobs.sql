-- name: CreateImportJob :one
INSERT INTO import_jobs (school_id, status, total_rows, processed_rows, payload, summary, errors)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetImportJobByID :one
SELECT * FROM import_jobs
WHERE id = $1 AND school_id = $2
LIMIT 1;

-- name: UpdateImportJobProgress :exec
UPDATE import_jobs
SET processed_rows = $3,
    created_count = $4,
    updated_count = $5,
    skipped_count = $6,
    failed_count = $7,
    updated_at = NOW()
WHERE id = $1 AND school_id = $2;

-- name: UpdateImportJobStatus :exec
UPDATE import_jobs
SET status = $3,
    summary = COALESCE($4, summary),
    errors = COALESCE($5, errors),
    updated_at = NOW()
WHERE id = $1 AND school_id = $2;

-- name: FailStaleRunningJobs :exec
UPDATE import_jobs
SET status = 'failed',
    summary = jsonb_build_object('error', 'Server qayta ishga tushdi. Iltimos, qayta urining.'),
    updated_at = NOW()
WHERE status IN ('pending', 'running');
