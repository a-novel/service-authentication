-- The account keeps its logged-out sessions active last, and the others go least recent first. A
-- session is logged out once it ends or a revocation holds its ID.
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
          sessions
        WHERE
          user_id = ?0
          AND (
            expires_at <= CURRENT_TIMESTAMP
            OR EXISTS (
              SELECT
                1
              FROM
                refresh_token_revocations
              WHERE
                jti = sessions.id::text
            )
          )
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
