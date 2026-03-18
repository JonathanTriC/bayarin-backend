ALTER TABLE tables DROP COLUMN IF EXISTS reserved_by;
ALTER TABLE tables DROP COLUMN IF EXISTS reserved_note;
ALTER TABLE tables DROP COLUMN IF EXISTS updated_at;
-- NOTE: cannot remove enum value in PostgreSQL without recreating type
-- Removing enum value is intentionally skipped in down migration
