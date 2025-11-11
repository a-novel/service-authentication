SELECT
  *
FROM
  short_codes
WHERE
  target = ?0
  AND usage = ?1
  AND deleted_at IS NULL
  AND expires_at > CURRENT_TIMESTAMP;
