ALTER TABLE lost_reports
    ALTER COLUMN date_lost TYPE DATE
    USING date_lost::DATE;
