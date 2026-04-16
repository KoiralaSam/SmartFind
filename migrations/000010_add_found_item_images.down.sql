ALTER TABLE found_items
  DROP COLUMN IF EXISTS primary_image_key,
  DROP COLUMN IF EXISTS image_keys;

