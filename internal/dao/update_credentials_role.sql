UPDATE credentials
SET role = ?0, updated_at = ?1
WHERE id = ?2
RETURNING *;
