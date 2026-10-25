DELETE FROM refresh_token_revocations
WHERE
  jti IN (
    SELECT
      jti
    FROM
      refresh_token_revocations
    WHERE
      expires_at < CURRENT_TIMESTAMP
    LIMIT
      ?0
  );
