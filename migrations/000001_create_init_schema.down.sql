DROP INDEX IF EXISTS idx_item_claims_claimant_passenger_id;
DROP INDEX IF EXISTS idx_item_claims_item_id;
DROP INDEX IF EXISTS idx_lost_reports_status;
DROP INDEX IF EXISTS idx_lost_reports_reporter_passenger_id;
DROP INDEX IF EXISTS idx_found_items_status;
DROP INDEX IF EXISTS idx_found_items_posted_by_staff_id;

DROP TABLE IF EXISTS item_claims;
DROP TABLE IF EXISTS lost_reports;
DROP TABLE IF EXISTS found_items;
DROP TABLE IF EXISTS staff;
DROP TABLE IF EXISTS passengers;

DROP TYPE IF EXISTS lost_report_status;
DROP TYPE IF EXISTS claim_status;
DROP TYPE IF EXISTS found_item_status;
