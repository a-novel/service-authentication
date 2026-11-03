-- The account keeps its live sessions active last, and the others go least recent first. Each
-- revocation lasts until its session ends, plus the leeway the verifier grants a refresh token.
INSERT INTO
  refresh_token_revocations (jti, expires_at, created_at)
SELECT
  excess.id::text,
  excess.expires_at + make_interval(secs => ?3),
  ?4
FROM
  (
    SELECT
      id,
      expires_at,
      last_active_at
    FROM
      sessions
    WHERE
      user_id = ?0
      AND expires_at > CURRENT_TIMESTAMP
      AND NOT EXISTS (
        SELECT
          1
        FROM
          refresh_token_revocations
        WHERE
          jti = sessions.id::text
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
ON CONFLICT (jti) DO NOTHING;
