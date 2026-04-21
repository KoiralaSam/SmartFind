CREATE TABLE IF NOT EXISTS passenger_match_notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  passenger_id UUID NOT NULL REFERENCES passengers(id) ON DELETE CASCADE,
  lost_report_id UUID NOT NULL REFERENCES lost_reports(id) ON DELETE CASCADE,
  found_item_id UUID NOT NULL,
  similarity_score FLOAT8 NOT NULL,
  item_name TEXT NOT NULL,
  image_urls TEXT[] NOT NULL DEFAULT '{}'::text[],
  primary_image_url TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  read_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_passenger_match_notifications_lost_found
  ON passenger_match_notifications (lost_report_id, found_item_id);

CREATE INDEX IF NOT EXISTS idx_passenger_match_notifications_poll
  ON passenger_match_notifications (passenger_id, read_at, created_at DESC);
