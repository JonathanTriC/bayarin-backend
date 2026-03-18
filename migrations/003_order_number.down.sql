DROP INDEX IF EXISTS idx_orders_branch_number_date;
ALTER TABLE orders DROP COLUMN IF EXISTS order_number;
