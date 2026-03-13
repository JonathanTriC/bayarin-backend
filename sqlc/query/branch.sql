-- name: ListBranches :many
SELECT *
FROM branches
WHERE business_id = $1
ORDER BY created_at ASC;

-- name: GetBranchByID :one
SELECT *
FROM branches
WHERE id          = $1
  AND business_id = $2
LIMIT 1;

-- name: CreateBranch :one
INSERT INTO branches (business_id, name, address)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateBranch :one
UPDATE branches
SET
    name      = COALESCE($2, name),
    address   = COALESCE($3, address),
    is_active = COALESCE($4, is_active)
WHERE id          = $1
  AND business_id = $5
RETURNING *;
