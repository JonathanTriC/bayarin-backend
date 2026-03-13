-- name: CreatePayment :one
INSERT INTO payments (order_id, method, amount_paid, change_amount)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPaymentByOrderID :one
SELECT *
FROM payments
WHERE order_id = $1
LIMIT 1;

-- name: CreateTransaction :one
INSERT INTO transactions (business_id, order_id, branch_id, total)
VALUES ($1, $2, $3, $4)
RETURNING *;
