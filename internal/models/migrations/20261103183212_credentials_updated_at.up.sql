-- A password or email change ends every session opened before it: a refresh token issued before
-- credentials_updated_at no longer renews. Existing accounts take their creation time, so no session
-- ends when the column appears, and an account an older release inserts takes the time of its insert.
-- Tokens carry their issue time in whole seconds, and timestamp(0) rounds, so the default truncates:
-- a token issued in the second of the insert stays valid.
ALTER TABLE credentials
ADD COLUMN credentials_updated_at timestamp(0) with time zone;

UPDATE credentials
SET
  credentials_updated_at = created_at;

ALTER TABLE credentials
ALTER COLUMN credentials_updated_at
SET DEFAULT date_trunc('second', CURRENT_TIMESTAMP),
ALTER COLUMN credentials_updated_at
SET NOT NULL;
