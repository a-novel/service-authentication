UPDATE short_codes
SET
  code = ?1,
  created_at = ?2,
  expires_at = ?3
WHERE
  id = ?0
  AND deleted_at IS NULL
RETURNING
  *;
