-- The statement deletes a logged-out session alone: removing an active one would hide a live
-- session from its account. The revocation stays until its own purge, so the session stays signed
-- out.
DELETE FROM sessions
WHERE
  id IN (
    SELECT
      id
    FROM
      session_states
    WHERE
      id = ?0
      AND user_id = ?1
      AND logged_out
  );
