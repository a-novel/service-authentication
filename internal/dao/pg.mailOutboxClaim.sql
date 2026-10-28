UPDATE mail_outbox
SET
  attempts = attempts + 1,
  due_at = ?2
WHERE
  short_code_id = ?0
  AND due_at <= ?1
RETURNING
  *;
