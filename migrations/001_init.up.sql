-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ENUM types
CREATE TYPE user_role AS ENUM ('owner', 'cashier');
CREATE TYPE table_status AS ENUM ('available', 'occupied');
CREATE TYPE order_type AS ENUM ('dine_in', 'takeaway');
CREATE TYPE order_status AS ENUM ('open', 'paid', 'cancelled');
CREATE TYPE payment_method AS ENUM ('cash', 'qris', 'transfer');

-- businesses
CREATE TABLE businesses (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    TEXT NOT NULL,
    slug                    TEXT NOT NULL UNIQUE,
    tax_percent             NUMERIC(5, 2) NOT NULL DEFAULT 0,
    service_charge_percent  NUMERIC(5, 2) NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- branches
CREATE TABLE branches (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    address     TEXT NOT NULL DEFAULT '',
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- users
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id   UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    branch_id     UUID REFERENCES branches(id) ON DELETE SET NULL,
    name          TEXT NOT NULL,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          user_role NOT NULL DEFAULT 'cashier',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_business_id ON users(business_id);
CREATE INDEX idx_users_email ON users(email);

-- sessions
CREATE TABLE sessions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

-- menu_items
CREATE TABLE menu_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id  UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    price        NUMERIC(12, 2) NOT NULL DEFAULT 0,
    category     TEXT NOT NULL DEFAULT '',
    is_available BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_menu_items_business_id ON menu_items(business_id);

-- modifier_groups
CREATE TABLE modifier_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    is_required BOOLEAN NOT NULL DEFAULT false,
    max_select  INT NOT NULL DEFAULT 1
);

CREATE INDEX idx_modifier_groups_business_id ON modifier_groups(business_id);

-- modifier_options
CREATE TABLE modifier_options (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id    UUID NOT NULL REFERENCES modifier_groups(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    extra_price NUMERIC(12, 2) NOT NULL DEFAULT 0
);

-- tables
CREATE TABLE tables (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id  UUID NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    qr_code    TEXT NOT NULL DEFAULT '',
    status     table_status NOT NULL DEFAULT 'available',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tables_branch_id ON tables(branch_id);

-- orders
CREATE TABLE orders (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id            UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    branch_id              UUID NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    cashier_id             UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    table_id               UUID REFERENCES tables(id) ON DELETE SET NULL,
    type                   order_type NOT NULL DEFAULT 'dine_in',
    customer_name          TEXT NOT NULL DEFAULT '',
    status                 order_status NOT NULL DEFAULT 'open',
    subtotal               NUMERIC(12, 2) NOT NULL DEFAULT 0,
    tax_amount             NUMERIC(12, 2) NOT NULL DEFAULT 0,
    service_charge_amount  NUMERIC(12, 2) NOT NULL DEFAULT 0,
    total                  NUMERIC(12, 2) NOT NULL DEFAULT 0,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_business_id ON orders(business_id);
CREATE INDEX idx_orders_branch_id ON orders(branch_id);
CREATE INDEX idx_orders_cashier_id ON orders(cashier_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at);

-- order_items
CREATE TABLE order_items (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id       UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    menu_item_id   UUID NOT NULL REFERENCES menu_items(id) ON DELETE RESTRICT,
    quantity       INT NOT NULL DEFAULT 1,
    unit_price     NUMERIC(12, 2) NOT NULL DEFAULT 0,
    notes          TEXT NOT NULL DEFAULT '',
    subtotal       NUMERIC(12, 2) NOT NULL DEFAULT 0
);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);

-- order_item_modifiers
CREATE TABLE order_item_modifiers (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_item_id      UUID NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    modifier_option_id UUID NOT NULL REFERENCES modifier_options(id) ON DELETE RESTRICT,
    extra_price        NUMERIC(12, 2) NOT NULL DEFAULT 0
);

-- payments
CREATE TABLE payments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id       UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    method         payment_method NOT NULL DEFAULT 'cash',
    amount_paid    NUMERIC(12, 2) NOT NULL DEFAULT 0,
    change_amount  NUMERIC(12, 2) NOT NULL DEFAULT 0,
    paid_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- transactions
CREATE TABLE transactions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    order_id    UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    branch_id   UUID NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    total       NUMERIC(12, 2) NOT NULL DEFAULT 0,
    paid_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transactions_business_id ON transactions(business_id);
CREATE INDEX idx_transactions_branch_id ON transactions(branch_id);
CREATE INDEX idx_transactions_paid_at ON transactions(paid_at);
