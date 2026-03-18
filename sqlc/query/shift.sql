-- name: OpenShift :one
INSERT INTO shifts (id, business_id, branch_id, cashier_id, started_at, is_open, created_at)
VALUES ($1, $2, $3, $4, NOW(), true, NOW())
RETURNING *;

-- name: GetOpenShiftByCashier :one
SELECT * FROM shifts
WHERE cashier_id = $1
  AND branch_id = $2
  AND is_open = true
LIMIT 1;

-- name: CloseShift :one
UPDATE shifts
SET
    is_open          = false,
    ended_at         = NOW(),
    total_orders     = $2,
    total_revenue    = $3,
    cash_revenue     = $4,
    qris_revenue     = $5,
    transfer_revenue = $6,
    cancelled_orders = $7
WHERE id = $1
RETURNING *;

-- name: GetShiftByID :one
SELECT * FROM shifts
WHERE id = $1 AND business_id = $2
LIMIT 1;

-- name: ListShiftsByCashierPaginated :many
SELECT * FROM shifts
WHERE cashier_id = $1
  AND business_id = $2
ORDER BY started_at DESC
LIMIT $3 OFFSET $4;

-- name: CountShiftsByCashier :one
SELECT COUNT(*) FROM shifts
WHERE cashier_id = $1
  AND business_id = $2;

-- name: ListShiftsByBranchPaginated :many
SELECT s.*, u.name as cashier_name
FROM shifts s
JOIN users u ON u.id = s.cashier_id
WHERE s.branch_id = $1
  AND s.business_id = $2
ORDER BY s.started_at DESC
LIMIT $3 OFFSET $4;

-- name: CountShiftsByBranch :one
SELECT COUNT(*) FROM shifts
WHERE branch_id = $1
  AND business_id = $2;

-- name: GetShiftOrderStats :one
-- Aggregate orders within shift time range for a cashier
SELECT
    COUNT(*) FILTER (WHERE o.status = 'paid')       AS total_orders,
    COUNT(*) FILTER (WHERE o.status = 'cancelled')  AS cancelled_orders,
    COALESCE(SUM(t.total) FILTER (WHERE o.status = 'paid'), 0) AS total_revenue,
    COALESCE(SUM(t.total) FILTER (
        WHERE o.status = 'paid' AND p.method = 'cash'
    ), 0) AS cash_revenue,
    COALESCE(SUM(t.total) FILTER (
        WHERE o.status = 'paid' AND p.method = 'qris'
    ), 0) AS qris_revenue,
    COALESCE(SUM(t.total) FILTER (
        WHERE o.status = 'paid' AND p.method = 'transfer'
    ), 0) AS transfer_revenue
FROM orders o
LEFT JOIN transactions t ON t.order_id = o.id
LEFT JOIN payments p ON p.order_id = o.id
WHERE o.cashier_id = $1
  AND o.branch_id  = $2
  AND o.created_at >= $3
  AND o.created_at <= $4;

-- name: GetShiftTopItems :many
-- Top 5 selling items within shift time range
SELECT
    mi.id           AS menu_item_id,
    mi.name         AS menu_item_name,
    mi.category,
    SUM(oi.quantity)    AS total_qty,
    SUM(oi.subtotal)    AS total_revenue
FROM order_items oi
JOIN menu_items mi ON mi.id = oi.menu_item_id
JOIN orders o      ON o.id  = oi.order_id
WHERE o.cashier_id = $1
  AND o.branch_id  = $2
  AND o.status     = 'paid'
  AND o.created_at >= $3
  AND o.created_at <= $4
GROUP BY mi.id, mi.name, mi.category
ORDER BY total_qty DESC
LIMIT 5;

-- name: GetBranchShiftOrderStats :one
-- Aggregate for all cashiers in a branch within a time range
SELECT
    COUNT(*) FILTER (WHERE o.status = 'paid')       AS total_orders,
    COUNT(*) FILTER (WHERE o.status = 'cancelled')  AS cancelled_orders,
    COALESCE(SUM(t.total) FILTER (WHERE o.status = 'paid'), 0) AS total_revenue,
    COALESCE(SUM(t.total) FILTER (
        WHERE o.status = 'paid' AND p.method = 'cash'
    ), 0) AS cash_revenue,
    COALESCE(SUM(t.total) FILTER (
        WHERE o.status = 'paid' AND p.method = 'qris'
    ), 0) AS qris_revenue,
    COALESCE(SUM(t.total) FILTER (
        WHERE o.status = 'paid' AND p.method = 'transfer'
    ), 0) AS transfer_revenue
FROM orders o
LEFT JOIN transactions t ON t.order_id = o.id
LEFT JOIN payments p ON p.order_id = o.id
WHERE o.branch_id   = $1
  AND o.business_id = $2
  AND o.created_at >= $3
  AND o.created_at <= $4;
