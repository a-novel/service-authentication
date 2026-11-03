-- The account keeps its logged-out sessions active last. A session is logged out once it ends or a
-- revocation holds its ID.
DELETE FROM sessions
WHERE
  id IN (
    SELECT
      id
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
    LIMIT
      ?2
  );
