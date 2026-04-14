-- Switch staff login identifier from employee_id to email.
-- NOTE: This will fail if any existing staff rows have email IS NULL.

ALTER TABLE staff
  ALTER COLUMN email SET NOT NULL,
  DROP COLUMN employee_id;

