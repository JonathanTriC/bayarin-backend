-- name: GetBusiness :one
SELECT *
FROM businesses
WHERE id = $1
LIMIT 1;

-- name: UpdateBusiness :one
UPDATE businesses
SET
    name                   = COALESCE($2, name),
    tax_percent            = COALESCE($3, tax_percent),
    service_charge_percent = COALESCE($4, service_charge_percent)
WHERE id = $1
RETURNING *;
