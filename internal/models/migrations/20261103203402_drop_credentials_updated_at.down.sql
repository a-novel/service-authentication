-- The change times are gone: every account takes its creation time, as when the column first
-- appeared.
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
