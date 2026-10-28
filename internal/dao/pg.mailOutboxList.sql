-- The entries due longest go first, through the due_at index.
SELECT
  *
FROM
  mail_outbox
WHERE
  due_at <= ?0
ORDER BY
  due_at
LIMIT
  ?1;
