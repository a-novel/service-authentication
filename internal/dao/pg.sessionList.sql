-- Active sessions come first.
SELECT
  *
FROM
  session_states
WHERE
  user_id = ?0
ORDER BY
  logged_out,
  last_active_at DESC,
  id DESC
LIMIT
  ?1;
