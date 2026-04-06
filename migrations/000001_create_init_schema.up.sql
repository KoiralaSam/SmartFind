CREATE TYPE found_item_status AS ENUM ('unclaimed', 'claimed', 'returned', 'archived');
CREATE TYPE claim_status AS ENUM ('pending', 'approved', 'rejected', 'cancelled');
CREATE TYPE lost_report_status AS ENUM ('open', 'matched', 'closed');

CREATE TABLE users (
  id UUID PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  full_name TEXT NOT NULL,
  phone TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE staff (
  id UUID PRIMARY KEY,
  employee_id TEXT UNIQUE NOT NULL,
  full_name TEXT NOT NULL,
  email TEXT UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE found_items (
  id UUID PRIMARY KEY,
  posted_by_staff_id UUID NOT NULL REFERENCES staff(id),
  item_name TEXT NOT NULL,
  description TEXT,
  category TEXT,
  location_found TEXT,
  route_or_station TEXT,
  date_found DATE,
  status found_item_status NOT NULL DEFAULT 'unclaimed',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE lost_reports (
  id UUID PRIMARY KEY,
  reporter_user_id UUID NOT NULL REFERENCES users(id),
  item_name TEXT NOT NULL,
  description TEXT,
  category TEXT,
  location_lost TEXT,
  route_or_station TEXT,
  date_lost DATE,
  status lost_report_status NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE item_claims (
  id UUID PRIMARY KEY,
  item_id UUID NOT NULL REFERENCES found_items(id) ON DELETE CASCADE,
  claimant_user_id UUID NOT NULL REFERENCES users(id),
  lost_report_id UUID REFERENCES lost_reports(id) ON DELETE SET NULL,
  message TEXT,
  status claim_status NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_found_items_posted_by_staff_id ON found_items (posted_by_staff_id);
CREATE INDEX idx_found_items_status ON found_items (status);
CREATE INDEX idx_lost_reports_reporter_user_id ON lost_reports (reporter_user_id);
CREATE INDEX idx_lost_reports_status ON lost_reports (status);
CREATE INDEX idx_item_claims_item_id ON item_claims (item_id);
CREATE INDEX idx_item_claims_claimant_user_id ON item_claims (claimant_user_id);
