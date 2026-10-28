UPDATE mail_outbox
SET
  due_at = ?1
WHERE
  short_code_id = ?0
RETURNING
  *;
