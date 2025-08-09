UPDATE short_codes
SET deleted_at = ?0, deleted_comment = ?1
WHERE
  id = ?2 AND deleted_at IS NULL AND expires_at > CURRENT_TIMESTAMP
RETURNING *;
