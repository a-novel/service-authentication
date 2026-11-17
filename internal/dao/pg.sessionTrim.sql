-- The account keeps its logged-out sessions active last, and the others go least recent first.
DELETE FROM sessions
WHERE
  id IN (
    SELECT
      excess.id
    FROM
      (
        SELECT
          id,
          last_active_at
        FROM
          session_states
        WHERE
          user_id = ?0
          AND logged_out
        ORDER BY
          last_active_at DESC,
          id DESC
        OFFSET
          ?1
      ) AS excess
    ORDER BY
      excess.last_active_at,
      excess.id
    LIMIT
      ?2
  );
