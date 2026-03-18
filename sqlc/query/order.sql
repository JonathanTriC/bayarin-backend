-- name: CreateOrder :one
INSERT INTO orders (
    id, business_id, branch_id, cashier_id, table_id,
    type, customer_name, status, order_number,
    subtotal, tax_amount, service_charge_amount, total,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'open', $8, 0, 0, 0, 0, NOW())
RETURNING *;

-- name: GetNextOrderNumber :one
SELECT COALESCE(MAX(
    CAST(SUBSTRING(order_number FROM 5) AS INTEGER)
), 0) + 1 AS next_seq
FROM orders
WHERE branch_id = $1
  AND DATE(created_at AT TIME ZONE 'Asia/Jakarta') = DATE(NOW() AT TIME ZONE 'Asia/Jakarta')
  AND order_number IS NOT NULL;

-- name: GetOrderByID :one
SELECT *
FROM orders
WHERE id          = $1
  AND business_id = $2
LIMIT 1;

-- name: GetOrderByIDForUpdate :one
SELECT *
FROM orders
WHERE id          = $1
  AND business_id = $2
LIMIT 1
FOR UPDATE;

-- name: ListOrdersPaginated :many
SELECT * FROM orders
WHERE business_id = $1
  AND ($2::text = '' OR status::text = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListOrdersByBranchPaginated :many
SELECT * FROM orders
WHERE business_id = $1
  AND branch_id = $2
  AND ($3::text = '' OR status::text = $3)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: CountOrders :one
SELECT COUNT(*) FROM orders
WHERE business_id = $1
  AND ($2::text = '' OR status::text = $2);

-- name: CountOrdersByBranch :one
SELECT COUNT(*) FROM orders
WHERE business_id = $1
  AND branch_id = $2
  AND ($3::text = '' OR status::text = $3);

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2
WHERE id          = $1
  AND business_id = $3
RETURNING *;

-- name: UpdateOrderTotals :one
UPDATE orders
SET
    subtotal              = $2,
    tax_amount            = $3,
    service_charge_amount = $4,
    total                 = $5
WHERE id = $1
RETURNING *;

-- name: AddOrderItem :one
INSERT INTO order_items (order_id, menu_item_id, quantity, unit_price, notes, subtotal)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetOrderItemByID :one
SELECT *
FROM order_items
WHERE id       = $1
  AND order_id = $2
LIMIT 1;

-- name: ListOrderItems :many
SELECT *
FROM order_items
WHERE order_id = $1
ORDER BY id;

-- name: UpdateOrderItem :one
UPDATE order_items
SET
    quantity = COALESCE($2, quantity),
    notes    = COALESCE($3, notes),
    subtotal = COALESCE($4, subtotal)
WHERE id       = $1
  AND order_id = $5
RETURNING *;

-- name: DeleteOrderItem :exec
DELETE FROM order_items
WHERE id       = $1
  AND order_id = $2;

-- name: AddOrderItemModifier :one
INSERT INTO order_item_modifiers (order_item_id, modifier_option_id, extra_price)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListOrderItemModifiers :many
SELECT *
FROM order_item_modifiers
WHERE order_item_id = $1;

-- name: DeleteOrderItemModifiers :exec
DELETE FROM order_item_modifiers
WHERE order_item_id = $1;
