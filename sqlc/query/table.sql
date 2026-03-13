-- name: ListTables :many
SELECT t.*
FROM tables t
JOIN branches b ON b.id = t.branch_id
WHERE b.business_id = $1
  AND t.branch_id   = $2
ORDER BY t.name ASC;

-- name: ListTablesByBusiness :many
SELECT t.*
FROM tables t
JOIN branches b ON b.id = t.branch_id
WHERE b.business_id = $1
ORDER BY t.name ASC;

-- name: GetTableByID :one
SELECT t.*
FROM tables t
JOIN branches b ON b.id = t.branch_id
WHERE t.id          = $1
  AND b.business_id = $2
LIMIT 1;

-- name: CreateTable :one
INSERT INTO tables (branch_id, name, qr_code)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateTable :one
UPDATE tables
SET
    name    = COALESCE($2, name),
    qr_code = COALESCE($3, qr_code),
    status  = COALESCE($4, status)
WHERE id = $1
RETURNING *;

-- name: SetTableAvailable :exec
UPDATE tables
SET status = 'available'
WHERE id = $1;
