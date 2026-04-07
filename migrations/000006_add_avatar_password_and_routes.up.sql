CREATE TABLE IF NOT EXISTS routes (
  id UUID PRIMARY KEY,
  route_name TEXT UNIQUE NOT NULL,
  created_by_staff_id UUID REFERENCES staff(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE passengers
  ADD COLUMN IF NOT EXISTS avatar_url TEXT;

ALTER TABLE staff
  ADD COLUMN IF NOT EXISTS password_hash TEXT;

ALTER TABLE found_items
  ADD COLUMN IF NOT EXISTS route_id UUID REFERENCES routes(id) ON DELETE SET NULL;

ALTER TABLE lost_reports
  ADD COLUMN IF NOT EXISTS route_id UUID REFERENCES routes(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_routes_route_name ON routes (route_name);
CREATE INDEX IF NOT EXISTS idx_routes_created_by_staff_id ON routes (created_by_staff_id);
CREATE INDEX IF NOT EXISTS idx_found_items_route_id ON found_items (route_id);
CREATE INDEX IF NOT EXISTS idx_lost_reports_route_id ON lost_reports (route_id);

