ALTER TABLE lost_reports
  DROP COLUMN IF EXISTS match_last_checked_at,
  DROP COLUMN IF EXISTS match_last_emailed_at;

