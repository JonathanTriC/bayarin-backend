-- Add order_number column to orders
ALTER TABLE orders ADD COLUMN IF NOT EXISTS order_number TEXT;

-- Add unique constraint: one order_number per branch per day
CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_branch_number_date
ON orders (branch_id, order_number, DATE(created_at AT TIME ZONE 'Asia/Jakarta'));

WITH numbered_orders AS (
    SELECT id, ROW_NUMBER() OVER (
        PARTITION BY branch_id, DATE(created_at AT TIME ZONE 'Asia/Jakarta')
        ORDER BY created_at
    ) as rnum
    FROM orders
    WHERE order_number IS NULL
)
UPDATE orders o
SET order_number = 'ORD-' || LPAD(n.rnum::TEXT, 4, '0')
FROM numbered_orders n
WHERE o.id = n.id;
