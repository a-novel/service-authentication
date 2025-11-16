UPDATE short_codes
SET
  deleted_at = ?0,
  deleted_comment = ?1
WHERE
  id = ?2
  AND deleted_at IS NULL -- Short code must not be deleted
  AND expires_at > CURRENT_TIMESTAMP -- Short code must not be expired
RETURNING
  *;
