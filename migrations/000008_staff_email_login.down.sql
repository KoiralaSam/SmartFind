-- Re-introduce employee_id and allow email to be nullable again.
-- Since 000008 makes email NOT NULL and UNIQUE, we can safely backfill employee_id from email.

ALTER TABLE staff
  ADD COLUMN employee_id TEXT;

UPDATE staff
SET employee_id = email
WHERE employee_id IS NULL;

ALTER TABLE staff
  ALTER COLUMN employee_id SET NOT NULL,
  ADD CONSTRAINT staff_employee_id_key UNIQUE (employee_id),
  ALTER COLUMN email DROP NOT NULL;

