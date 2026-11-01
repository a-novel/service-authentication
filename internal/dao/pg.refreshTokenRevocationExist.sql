SELECT
  1
FROM
  refresh_token_revocations
WHERE
  jti IN (?0)
LIMIT
  1;
