-- Dropping the column forgets when the credentials last changed: refresh tokens issued before a
-- password or email change renew again until they expire.
ALTER TABLE credentials
DROP COLUMN IF EXISTS credentials_updated_at;
