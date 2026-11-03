-- The sessions that ended first go first, through the expires_at index.
DELETE FROM sessions
WHERE
  id IN (
    SELECT
      id
    FROM
      sessions
    WHERE
      expires_at < ?0
    ORDER BY
      expires_at
    LIMIT
      ?1
  );
