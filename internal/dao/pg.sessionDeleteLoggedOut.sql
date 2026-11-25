-- The statement deletes logged-out sessions alone, so every active session of the account keeps its
-- row. Each revocation stays until its own purge, so the sessions stay signed out. The statement
-- returns no row, so the number of sessions it deletes costs the service no memory.
DELETE FROM sessions
WHERE
  id IN (
    SELECT
      id
    FROM
      session_states
    WHERE
      user_id = ?0
      AND logged_out
  );
