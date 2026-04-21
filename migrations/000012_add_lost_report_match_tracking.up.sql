ALTER TABLE lost_reports
  ADD COLUMN IF NOT EXISTS match_last_checked_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS match_last_emailed_at TIMESTAMPTZ;
