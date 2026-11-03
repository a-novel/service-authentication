-- Each revocation lasts until its session ends, plus the leeway the verifier grants a refresh token.
-- A session already revoked keeps its first row. The statement returns no row, so the number of
-- sessions it revokes costs the service no memory.
INSERT INTO
  refresh_token_revocations (jti, expires_at, created_at)
SELECT
  sessions.id::text,
  sessions.expires_at + make_interval(secs => ?2),
  ?3
FROM
  sessions
WHERE
  sessions.user_id = ?0
  AND sessions.id <> ?1
  AND sessions.expires_at > CURRENT_TIMESTAMP
ON CONFLICT (jti) DO NOTHING;
