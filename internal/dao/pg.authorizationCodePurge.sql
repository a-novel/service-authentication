-- The oldest codes go first, through the expires_at index.
DELETE FROM authorization_codes
WHERE
  id IN (
    SELECT
      id
    FROM
      authorization_codes
    WHERE
      expires_at < CURRENT_TIMESTAMP
    ORDER BY
      expires_at
    LIMIT
      ?0
  );
