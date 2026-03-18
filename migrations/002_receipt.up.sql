-- Add business logo URL to businesses table
ALTER TABLE businesses ADD COLUMN IF NOT EXISTS image_url TEXT;
