-- A password or email change no longer ends the sessions opened before it, so nothing reads the time
-- of the last change. Dropping the column changes only the catalog.
ALTER TABLE credentials
DROP COLUMN credentials_updated_at;
