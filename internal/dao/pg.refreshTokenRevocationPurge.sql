-- The oldest revocations go first, through the expires_at index.
DELETE FROM refresh_token_revocations
WHERE
  jti IN (
    SELECT
      jti
    FROM
      refresh_token_revocations
    WHERE
      expires_at < CURRENT_TIMESTAMP
    ORDER BY
      expires_at
    LIMIT
      ?0
  );
