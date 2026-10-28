DELETE FROM mail_outbox
WHERE
  short_code_id = ?0
RETURNING
  *;
