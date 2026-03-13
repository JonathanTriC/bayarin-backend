-- name: ListModifierGroups :many
SELECT *
FROM modifier_groups
WHERE business_id = $1
ORDER BY name ASC;

-- name: GetModifierGroupByID :one
SELECT *
FROM modifier_groups
WHERE id          = $1
  AND business_id = $2
LIMIT 1;

-- name: CreateModifierGroup :one
INSERT INTO modifier_groups (business_id, name, is_required, max_select)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateModifierGroup :one
UPDATE modifier_groups
SET
    name        = COALESCE($2, name),
    is_required = COALESCE($3, is_required),
    max_select  = COALESCE($4, max_select)
WHERE id          = $1
  AND business_id = $5
RETURNING *;

-- name: ListModifierOptions :many
SELECT *
FROM modifier_options
WHERE group_id = $1
ORDER BY name ASC;

-- name: GetModifierOptionByID :one
SELECT *
FROM modifier_options
WHERE id = $1
LIMIT 1;

-- name: CreateModifierOption :one
INSERT INTO modifier_options (group_id, name, extra_price)
VALUES ($1, $2, $3)
RETURNING *;
