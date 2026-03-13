-- name: ListStaff :many
SELECT *
FROM users
WHERE business_id = $1
ORDER BY created_at ASC;

-- name: GetStaffByID :one
SELECT *
FROM users
WHERE id          = $1
  AND business_id = $2
LIMIT 1;

-- name: CreateStaff :one
INSERT INTO users (business_id, branch_id, name, email, password_hash, role)
VALUES ($1, $2, $3, $4, $5, 'cashier')
RETURNING *;

-- name: UpdateStaffStatus :one
UPDATE users
SET is_active = $2
WHERE id          = $1
  AND business_id = $3
  AND role        != 'owner'
RETURNING *;
