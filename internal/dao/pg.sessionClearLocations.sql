-- A row without a location stays as it is, so a repeated withdrawal rewrites only the rows that
-- recorded one since.
UPDATE sessions
SET
  ip_address = NULL,
  region = NULL,
  city = NULL
WHERE
  user_id = ?0
  AND (
    ip_address IS NOT NULL
    OR region IS NOT NULL
    OR city IS NOT NULL
  );
