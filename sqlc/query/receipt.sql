-- name: GetReceiptData :one
SELECT
    o.id                        AS order_id,
    o.type                      AS order_type,
    o.customer_name,
    o.status                    AS order_status,
    o.subtotal,
    o.tax_amount,
    o.service_charge_amount,
    o.total,
    o.created_at                AS ordered_at,
    b.name                      AS business_name,
    b.image_url                 AS business_logo_url,
    b.tax_percent,
    b.service_charge_percent,
    br.name                     AS branch_name,
    br.address                  AS branch_address,
    u.name                      AS cashier_name,
    p.method                    AS payment_method,
    p.amount_paid,
    p.change_amount,
    p.paid_at
FROM orders o
JOIN businesses b  ON b.id = o.business_id
JOIN branches br   ON br.id = o.branch_id
JOIN users u       ON u.id = o.cashier_id
JOIN payments p    ON p.order_id = o.id
WHERE o.id = $1 AND o.business_id = $2;

-- name: GetReceiptItems :many
SELECT
    oi.id           AS item_id,
    mi.name         AS item_name,
    oi.quantity,
    oi.unit_price,
    oi.subtotal,
    oi.notes
FROM order_items oi
JOIN menu_items mi ON mi.id = oi.menu_item_id
WHERE oi.order_id = $1
ORDER BY oi.id;

-- name: GetReceiptItemModifiers :many
SELECT
    oim.order_item_id,
    mo.name         AS modifier_name,
    oim.extra_price
FROM order_item_modifiers oim
JOIN modifier_options mo ON mo.id = oim.modifier_option_id
WHERE oim.order_item_id = ANY($1::uuid[]);
