-- The 'user' -> 'auth:user' backfill is not reversed: which auth:user rows were once 'user' is not
-- recoverable, and 'user' is the broken value this migration exists to remove.
ALTER TABLE credentials
DROP CONSTRAINT IF EXISTS credentials_role_check;

ALTER TABLE credentials
ALTER COLUMN role
SET DEFAULT 'user';
