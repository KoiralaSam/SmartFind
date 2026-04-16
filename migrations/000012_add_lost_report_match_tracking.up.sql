ALTER TABLE lost_reports
  ADD COLUMN match_last_checked_at TIMESTAMPTZ,
  ADD COLUMN match_last_emailed_at TIMESTAMPTZ;

