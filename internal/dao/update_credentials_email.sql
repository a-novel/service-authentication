UPDATE credentials
SET email = ?0, updated_at = ?1
WHERE id = ?2
RETURNING *;
