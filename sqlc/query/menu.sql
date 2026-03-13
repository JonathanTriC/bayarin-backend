-- name: ListMenuItems :many
SELECT *
FROM menu_items
WHERE business_id = $1
ORDER BY category ASC, name ASC;

-- name: GetMenuItemByID :one
SELECT *
FROM menu_items
WHERE id          = $1
  AND business_id = $2
LIMIT 1;

-- name: CreateMenuItem :one
INSERT INTO menu_items (business_id, name, description, price, category, is_available)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateMenuItem :one
UPDATE menu_items
SET
    name         = COALESCE($2, name),
    description  = COALESCE($3, description),
    price        = COALESCE($4, price),
    category     = COALESCE($5, category),
    is_available = COALESCE($6, is_available)
WHERE id          = $1
  AND business_id = $7
RETURNING *;
