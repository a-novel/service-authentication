-- A session is logged out once it ends or a revocation holds its ID.
SELECT
  *,
  (
    expires_at <= CURRENT_TIMESTAMP
    OR EXISTS (
      SELECT
        1
      FROM
        refresh_token_revocations
      WHERE
        jti = sessions.id::text
    )
  ) AS logged_out
FROM
  sessions
WHERE
  id = ?0
  AND user_id = ?1;
