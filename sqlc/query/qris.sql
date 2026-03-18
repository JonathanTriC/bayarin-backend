-- name: DeactivateBranchQRIS :exec
UPDATE branch_qris
SET is_active = false
WHERE branch_id = $1;

-- name: InsertBranchQRIS :one
INSERT INTO branch_qris (id, branch_id, qris_string, image_path, uploaded_by, is_active, created_at)
VALUES ($1, $2, $3, $4, $5, true, NOW())
RETURNING *;

-- name: GetActiveBranchQRIS :one
SELECT * FROM branch_qris
WHERE branch_id = $1 AND is_active = true
LIMIT 1;

-- name: ListBranchQRISHistory :many
SELECT * FROM branch_qris
WHERE branch_id = $1
ORDER BY created_at DESC;
