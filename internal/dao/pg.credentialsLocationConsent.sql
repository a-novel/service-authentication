-- FOR SHARE waits for a consent update in flight, and holds the next one until the caller's
-- transaction ends.
SELECT
  location_consent
FROM
  credentials
WHERE
  id = ?0
FOR SHARE;
