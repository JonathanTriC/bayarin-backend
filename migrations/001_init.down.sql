-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS order_item_modifiers;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS tables;
DROP TABLE IF EXISTS modifier_options;
DROP TABLE IF EXISTS modifier_groups;
DROP TABLE IF EXISTS menu_items;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS branches;
DROP TABLE IF EXISTS businesses;

-- Drop ENUM types
DROP TYPE IF EXISTS payment_method;
DROP TYPE IF EXISTS order_status;
DROP TYPE IF EXISTS order_type;
DROP TYPE IF EXISTS table_status;
DROP TYPE IF EXISTS user_role;
