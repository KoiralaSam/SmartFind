DROP INDEX IF EXISTS idx_lost_reports_route_id;
DROP INDEX IF EXISTS idx_found_items_route_id;
DROP INDEX IF EXISTS idx_routes_created_by_staff_id;
DROP INDEX IF EXISTS idx_routes_route_name;

ALTER TABLE lost_reports
  DROP COLUMN IF EXISTS route_id;

ALTER TABLE found_items
  DROP COLUMN IF EXISTS route_id;

ALTER TABLE staff
  DROP COLUMN IF EXISTS password_hash;

ALTER TABLE passengers
  DROP COLUMN IF EXISTS avatar_url;

DROP TABLE IF EXISTS routes;

