-- name: GetOwnerDashboard :one
SELECT
    COUNT(*)    FILTER (WHERE o.status = 'paid' AND o.created_at >= CURRENT_DATE)                AS total_orders_today,
    COALESCE(
        SUM(t.total) FILTER (WHERE t.paid_at >= CURRENT_DATE),
        0
    )                                                                                             AS total_revenue_today
FROM orders o
LEFT JOIN transactions t ON t.order_id = o.id
WHERE o.business_id = $1;

-- name: GetTopMenuItems :many
SELECT
    oi.menu_item_id,
    mi.name                  AS menu_item_name,
    SUM(oi.quantity)::INT    AS total_sold,
    SUM(oi.subtotal)         AS total_revenue
FROM order_items   oi
JOIN menu_items    mi ON mi.id = oi.menu_item_id
JOIN orders        o  ON o.id  = oi.order_id
WHERE o.business_id = $1
  AND o.status      = 'paid'
  AND o.created_at  >= CURRENT_DATE
GROUP BY oi.menu_item_id, mi.name
ORDER BY total_sold DESC
LIMIT 5;

-- name: GetRevenuePerBranch :many
SELECT
    b.id                              AS branch_id,
    b.name                            AS branch_name,
    COALESCE(SUM(t.total), 0)         AS revenue
FROM branches b
LEFT JOIN transactions t
    ON  t.branch_id   = b.id
    AND t.paid_at     >= CURRENT_DATE
WHERE b.business_id = $1
GROUP BY b.id, b.name
ORDER BY b.name ASC;

-- name: GetCashierDashboard :one
SELECT
    COUNT(*) FILTER (WHERE o.status = 'open')                                                     AS open_orders_count,
    COUNT(*) FILTER (WHERE o.status = 'paid' AND o.created_at >= CURRENT_DATE)                   AS orders_handled_today,
    COALESCE(
        SUM(p.amount_paid) FILTER (WHERE p.paid_at >= CURRENT_DATE),
        0
    )                                                                                             AS total_collected_today
FROM orders o
LEFT JOIN payments p ON p.order_id = o.id
WHERE o.business_id = $1
  AND o.cashier_id  = $2;
