ALTER TABLE lost_reports
  DROP COLUMN IF EXISTS item_condition,
  DROP COLUMN IF EXISTS item_description_text,
  DROP COLUMN IF EXISTS material,
  DROP COLUMN IF EXISTS color,
  DROP COLUMN IF EXISTS model,
  DROP COLUMN IF EXISTS brand,
  DROP COLUMN IF EXISTS item_type,
  ADD COLUMN IF NOT EXISTS item_description TEXT;

ALTER TABLE found_items
  DROP COLUMN IF EXISTS item_condition,
  DROP COLUMN IF EXISTS item_description_text,
  DROP COLUMN IF EXISTS material,
  DROP COLUMN IF EXISTS color,
  DROP COLUMN IF EXISTS model,
  DROP COLUMN IF EXISTS brand,
  DROP COLUMN IF EXISTS item_type,
  ADD COLUMN IF NOT EXISTS item_description TEXT;

