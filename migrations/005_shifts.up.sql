CREATE TABLE shifts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id),
    branch_id   UUID NOT NULL REFERENCES branches(id),
    cashier_id  UUID NOT NULL REFERENCES users(id),
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at    TIMESTAMPTZ,
    is_open     BOOLEAN NOT NULL DEFAULT true,

    -- Summary fields (populated on close)
    total_orders        INT,
    total_revenue       NUMERIC(12,2),
    cash_revenue        NUMERIC(12,2),
    qris_revenue        NUMERIC(12,2),
    transfer_revenue    NUMERIC(12,2),
    cancelled_orders    INT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shifts_cashier    ON shifts(cashier_id);
CREATE INDEX idx_shifts_branch     ON shifts(branch_id);
CREATE INDEX idx_shifts_open       ON shifts(cashier_id, branch_id, is_open);
