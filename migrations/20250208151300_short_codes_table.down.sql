DROP TRIGGER IF EXISTS unique_active_short_codes ON short_codes;

DROP FUNCTION IF EXISTS check_unique_active_short_codes();

DROP VIEW IF EXISTS active_short_codes;

DROP TABLE IF EXISTS short_codes;
