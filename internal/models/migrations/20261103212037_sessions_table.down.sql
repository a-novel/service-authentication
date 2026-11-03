-- Dropping the table forgets every device an account listed, and dropping the column every consent.
-- The sessions themselves keep working: their tokens and revocations carry them.
ALTER TABLE credentials
DROP COLUMN IF EXISTS location_consent;

DROP INDEX IF EXISTS sessions_last_active_at_idx;

DROP INDEX IF EXISTS sessions_user_id_expires_at_idx;

DROP TABLE IF EXISTS sessions;
