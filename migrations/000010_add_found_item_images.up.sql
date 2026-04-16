ALTER TABLE found_items
  ADD COLUMN IF NOT EXISTS image_keys TEXT[] NOT NULL DEFAULT '{}'::text[],
  ADD COLUMN IF NOT EXISTS primary_image_key TEXT;

