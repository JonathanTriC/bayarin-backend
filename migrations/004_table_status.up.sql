-- Add 'reserved' to table_status enum
ALTER TYPE table_status ADD VALUE IF NOT EXISTS 'reserved';

-- Add reserved_by and reserved_note columns for reservation context
ALTER TABLE tables ADD COLUMN IF NOT EXISTS reserved_by   UUID REFERENCES users(id);
ALTER TABLE tables ADD COLUMN IF NOT EXISTS reserved_note TEXT;
ALTER TABLE tables ADD COLUMN IF NOT EXISTS updated_at    TIMESTAMPTZ DEFAULT NOW();
