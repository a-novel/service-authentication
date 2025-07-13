UPDATE short_codes
SET deleted_at = ?0, deleted_comment = ?1
WHERE
  target = ?2
  AND usage = ?3
  AND (deleted_at IS NULL OR deleted_at > CURRENT_TIMESTAMP)
  AND expires_at > CURRENT_TIMESTAMP;
