DROP VIEW IF EXISTS active_short_codes;

DROP INDEX IF EXISTS short_codes_created_at_idx;

DROP INDEX IF EXISTS short_codes_target_usage_idx;

DROP TABLE IF EXISTS short_codes;
