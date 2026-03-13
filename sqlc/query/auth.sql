-- name: CreateBusiness :one
INSERT INTO businesses (name, slug, tax_percent, service_charge_percent)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateUser :one
INSERT INTO users (business_id, branch_id, name, email, password_hash, role)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1
LIMIT 1;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1
LIMIT 1;

-- name: CreateSession :one
INSERT INTO sessions (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSessionByToken :one
SELECT
    s.id,
    s.user_id,
    s.token,
    s.expires_at,
    s.revoked,
    s.created_at
FROM sessions s
WHERE s.token    = $1
  AND s.revoked  = false
  AND s.expires_at > NOW()
LIMIT 1;

-- name: RevokeSession :exec
UPDATE sessions
SET revoked = true
WHERE token = $1;
