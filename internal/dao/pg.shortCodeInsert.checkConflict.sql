SELECT
  1
FROM
  short_codes
WHERE
  target = ?0
  AND usage = ?1
  AND (
    deleted_at IS NULL
    OR deleted_at > CURRENT_TIMESTAMP
  )
  AND expires_at > CURRENT_TIMESTAMP;
