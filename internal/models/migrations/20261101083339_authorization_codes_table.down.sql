-- Dropping the table forgets every code: a code issued before the rollback no longer exchanges, and
-- a reused one no longer ends its session.
DROP INDEX IF EXISTS authorization_codes_user_id_idx;

DROP INDEX IF EXISTS authorization_codes_expires_at_idx;

DROP TABLE IF EXISTS authorization_codes;
