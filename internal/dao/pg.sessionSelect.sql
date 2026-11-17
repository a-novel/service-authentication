SELECT
  *
FROM
  session_states
WHERE
  id = ?0
  AND user_id = ?1;
