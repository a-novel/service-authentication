UPDATE credentials
SET password = ?0, updated_at = ?1
WHERE id = ?2
RETURNING *;
